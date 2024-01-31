package archive

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/STARRY-S/zip"
	"github.com/sirupsen/logrus"
)

// Writer creates a new Hangar archive (zip) file and write files into it.
type Writer struct {
	f  *os.File
	zw *zip.Writer

	// Record the wrote file name.
	fileNameSet map[string]bool
}

// NewWriter constructs a new Writer object.
func NewWriter(name string) (*Writer, error) {
	f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q: %w", name, err)
	}

	return &Writer{
		f:           f,
		zw:          zip.NewWriter(f),
		fileNameSet: make(map[string]bool),
	}, nil
}

// Write writes a single file or a directory (recursive) to archive file.
func (w *Writer) Write(name string) error {
	fi, err := os.Stat(name)
	if err != nil {
		return err
	}
	mode := fi.Mode()
	if mode.IsRegular() {
		return w.writeFile(name, fi)
	}

	return w.writeDir(name)
}

func (w *Writer) writeFile(name string, fi fs.FileInfo) error {
	writer, err := w.zw.CreateHeader(&zip.FileHeader{
		Name:     name,
		Method:   zip.Store,
		Modified: fi.ModTime(),
	})
	if err != nil {
		return fmt.Errorf("zip create failed: %w", err)
	}
	file, err := os.Open(name)
	if err != nil {
		return fmt.Errorf("failed to open %q: %w", name, err)
	}
	defer file.Close()
	_, err = io.Copy(writer, file)
	if err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}
	w.fileNameSet[name] = true
	return nil
}

func (w *Writer) writeDir(base string) error {
	err := filepath.Walk(base, func(name string, fi os.FileInfo, e error) error {
		if e != nil {
			logrus.Warnf("writeDir: failed to open %s: %v", name, e)
			return nil
		}

		fname := strings.TrimPrefix(name, base)
		fname = strings.TrimPrefix(fname, string(os.PathSeparator))
		if fname == "" {
			return nil
		}
		// if not a dir, write file content
		if fi.IsDir() && !strings.HasSuffix(fname, string(os.PathSeparator)) {
			fname += string(os.PathSeparator)
		}
		writer, err := w.zw.CreateHeader(&zip.FileHeader{
			Name:     fname,
			Method:   zip.Store,
			Modified: fi.ModTime(),
		})
		if err != nil {
			return fmt.Errorf("zip create failed: %w", err)
		}
		if fi.IsDir() {
			logrus.Debugf("compress dir: %v", fname)
			w.fileNameSet[name] = true
			return nil
		}
		file, err := os.Open(name)
		if err != nil {
			return fmt.Errorf("failed to open %q: %w", fname, err)
		}
		_, err = io.Copy(writer, file)
		if err != nil {
			return fmt.Errorf("failed to copy data: %w", err)
		}
		logrus.Debugf("compress file: %v", fname)
		w.fileNameSet[name] = true
		return nil
	})
	if err != nil {
		return fmt.Errorf("writeDir walk: %w", err)
	}

	return w.zw.Flush()
}

// WriteIndex writes the index json file into the end of the zip archive.
func (w *Writer) WriteIndex(index *Index) error {
	var err error
	data, err := json.Marshal(index)
	if err != nil {
		return fmt.Errorf("writeIndex: %w", err)
	}
	writer, err := w.zw.CreateHeader(&zip.FileHeader{
		Name:   IndexFileName,
		Method: zip.Store,
	})
	if err != nil {
		return fmt.Errorf("writeIndex: failed to create file in zip: %w", err)
	}
	_, err = writer.Write(data)
	if err != nil {
		return fmt.Errorf("writeIndex: zip write failed: %w", err)
	}
	logrus.Infof("Write index file %q to [%s], size %.2fK",
		IndexFileName, w.f.Name(), float32(len(data))/1024)
	w.fileNameSet[IndexFileName] = true
	return nil
}

// CopyImage copy image blobs data from archive reader to writer.
func (w *Writer) CopyImage(image *Image, ar *Reader) error {
	for _, img := range image.Images {
		// Copy all blobs files.
		var fnames = make([]string, 0)
		fnames = append(fnames, fmt.Sprintf("%s/", img.Digest.Encoded()))
		fnames = append(fnames, fmt.Sprintf("%s/blobs/", img.Digest.Encoded()))
		fnames = append(fnames, fmt.Sprintf("%s/index.json", img.Digest.Encoded()))
		fnames = append(fnames, fmt.Sprintf("%s/oci-layout", img.Digest.Encoded()))
		for _, layer := range img.Layers {
			fname := fmt.Sprintf("%s/%s/%s", SharedBlobDir, layer.Algorithm(), layer.Encoded())
			fnames = append(fnames, fname)
		}
		fnames = append(fnames, fmt.Sprintf("%s/%s/%s",
			SharedBlobDir, img.Digest.Algorithm(), img.Digest.Encoded()))
		if img.Config != "" {
			fnames = append(fnames, fmt.Sprintf("%s/%s/%s",
				SharedBlobDir, img.Config.Algorithm(), img.Config.Encoded()))
		}

		for _, fname := range fnames {
			if w.fileNameSet[fname] == true {
				continue
			}

			var zf *zip.File
			for _, file := range ar.zr.File {
				if file.Name == fname {
					zf = file
					break
				}
			}
			if zf == nil {
				return fmt.Errorf("file %q not found in archive %q", fname, ar.f.Name())
			}
			iw, err := w.zw.CreateHeader(&zip.FileHeader{
				Name:    zf.Name,
				Comment: zf.Comment,
				Method:  zf.Method,
			})
			if err != nil {
				return fmt.Errorf("failed to create zip header: %w", err)
			}
			r, err := zf.Open()
			if err != nil {
				return fmt.Errorf("failed to read file %q from archive: %w", fname, err)
			}
			_, err = io.Copy(iw, r)
			if err != nil {
				return fmt.Errorf("failed to copy %q from archive: %w", fname, err)
			}
			w.fileNameSet[fname] = true
		}
	}
	return nil
}

func (w *Writer) Close() error {
	if w == nil {
		return nil
	}
	var err error
	if w.zw != nil {
		if err = w.zw.Close(); err != nil {
			return err
		}
		w.zw = nil
	}
	if w.f != nil {
		if err = w.f.Close(); err != nil {
			return err
		}
		w.f = nil
	}
	return nil
}
