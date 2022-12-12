package utils

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

func Compress(src string, dst string) error {
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("Compress: os.Open: %w", err)
	}
	zr := gzip.NewWriter(dstFile)
	tw := tar.NewWriter(zr)

	fi, err := os.Stat(src)
	if err != nil {
		return err
	}
	mode := fi.Mode()
	// Compress single file
	if mode.IsRegular() {
		// Write header
		header, err := tar.FileInfoHeader(fi, src)
		if err != nil {
			return err
		}
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		// Write content
		data, err := os.Open(src)
		if err != nil {
			return err
		}
		if _, err := io.Copy(tw, data); err != nil {
			return err
		}
		if err := tw.Close(); err != nil {
			return err
		}
		if err := zr.Close(); err != nil {
			return err
		}
		return nil
	}

	// Compress directory.
	// Walk through every file in the folder
	err = filepath.Walk(src, func(file string, fi os.FileInfo, e error) error {
		if e != nil {
			logrus.Warnf("failed to open %s: %v", file, e)
		}

		// write header
		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return fmt.Errorf("tar.FileInfoHeader: %w", err)
		}

		// must provide real name
		// (see https://golang.org/src/archive/tar/common.go?#L626)
		header.Name = filepath.ToSlash(file)
		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("tar.WriteHeader: %w", err)
		}
		// if not a dir, write file content
		if !fi.IsDir() {
			data, err := os.Open(file)
			if err != nil {
				return fmt.Errorf("os.Open: %w", err)
			}
			logrus.Debugf("Compress: %s", file)
			if _, err := io.Copy(tw, data); err != nil {
				return fmt.Errorf("io.Copy: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Compress: filepath.Walk: %w", err)
	}

	if err := tw.Close(); err != nil {
		return fmt.Errorf("Compress: %w", err)
	}
	if err := zr.Close(); err != nil {
		return fmt.Errorf("Compress: %w", err)
	}
	return nil
}

func Decompress(src string, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("Decompress: os.Open: %w", err)
	}
	zr, err := gzip.NewReader(srcFile)
	if err != nil {
		return fmt.Errorf("Decompress: gzip.NewReader: %w", err)
	}
	tr := tar.NewReader(zr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		target := header.Name

		// validate name against path traversal
		if !validRelPath(header.Name) {
			return fmt.Errorf("tar contained invalid name error %q", target)
		}

		target = filepath.Join(dst, header.Name)
		// if no join is needed, replace with ToSlash:
		// target = filepath.ToSlash(header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if os.IsNotExist(err) {
					if err := os.MkdirAll(target, 0755); err != nil {
						return fmt.Errorf("Decompress: os.MkdirAll: %w", err)
					}
				} else {
					return fmt.Errorf("Decompress: os.Stat: %w", err)
				}
			}
		case tar.TypeReg:
			fileToWrite, err := os.OpenFile(
				target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("Decompress: os.OpenFile: %w", err)
			}
			logrus.Debugf("Decompress: %s", target)
			if _, err := io.Copy(fileToWrite, tr); err != nil {
				return fmt.Errorf("Decompress: io.Copy: %w", err)
			}
			// manually close here after each file operation;
			// defering would cause each file close
			// to wait until all operations have completed.
			fileToWrite.Close()
		}
	}

	return nil
}

// check for path traversal and correct forward slashes
func validRelPath(p string) bool {
	if p == "" || strings.Contains(p, `\`) || strings.HasPrefix(p, "/") ||
		strings.Contains(p, "../") {
		return false
	}
	return true
}
