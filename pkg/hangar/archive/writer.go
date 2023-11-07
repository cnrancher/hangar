package archive

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/pgzip"
	"github.com/sirupsen/logrus"
)

type Writer struct {
	f      *os.File
	tw     *tar.Writer
	gw     *pgzip.Writer
	format Format
}

func NewWriter(fname string, format Format) (*Writer, error) {
	if format.String() == "" || format.String() == "zstd" {
		return nil, fmt.Errorf("unsupported format")
	}

	writer := &Writer{}
	f, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	writer.f = f
	if err := writer.init(format); err != nil {
		f.Close()
		return nil, err
	}
	writer.format = format

	return writer, nil
}

func (w *Writer) init(format Format) error {
	var (
		gw *pgzip.Writer
		tw *tar.Writer
	)

	switch format {
	case GZIP:
		gw = pgzip.NewWriter(w.f)
		tw = tar.NewWriter(gw)
	default:
		tw = tar.NewWriter(w.f)
	}

	w.gw = gw
	w.tw = tw

	return nil
}

// Write writes a single file or a directory (recursive) to archive file.
func (w *Writer) Write(fname string) error {
	fi, err := os.Stat(fname)
	if err != nil {
		return err
	}
	mode := fi.Mode()
	if mode.IsRegular() {
		header, err := tar.FileInfoHeader(fi, fname)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(fname)
		return w.writeFile(fname, header)
	}

	return w.writeDir(fname)
}

func (w *Writer) writeFile(fname string, header *tar.Header) error {
	if err := w.tw.WriteHeader(header); err != nil {
		return err
	}
	data, err := os.Open(fname)
	if err != nil {
		return err
	}
	if _, err := io.Copy(w.tw, data); err != nil {
		return err
	}
	logrus.Debugf("write: %v", fname)
	return nil
}

func (w *Writer) writeDir(base string) error {
	err := filepath.Walk(base, func(file string, fi os.FileInfo, e error) error {
		if e != nil {
			logrus.Warnf("writeDir: failed to open %s: %e", file, e)
		}

		// write header
		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return fmt.Errorf("tar.FileInfoHeader: %w", err)
		}
		header.Name = strings.TrimPrefix(file, base)
		header.Name = strings.TrimPrefix(header.Name, string(os.PathSeparator))
		if header.Name == "" {
			return nil
		}
		if err := w.tw.WriteHeader(header); err != nil {
			return fmt.Errorf("tar.WriteHeader: %w", err)
		}
		// if not a dir, write file content
		if !fi.IsDir() {
			data, err := os.Open(file)
			if err != nil {
				return fmt.Errorf("os.Open: %w", err)
			}
			if _, err := io.Copy(w.tw, data); err != nil {
				return fmt.Errorf("io.Copy: %w", err)
			}
		}
		logrus.Debugf("write: %v", file)
		return nil
	})
	if err != nil {
		return fmt.Errorf("filepath.Walk: %w", err)
	}

	return nil
}

func (w *Writer) Close() error {
	var err error
	if w.gw != nil {
		err = w.gw.Close()
		if err != nil {
			return err
		}
		w.gw = nil
	}
	if w.tw != nil {
		err = w.tw.Close()
		if err != nil {
			return err
		}
		w.tw = nil
	}
	return nil
}
