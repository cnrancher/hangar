package archive

import (
	"encoding/json"
	"fmt"
	"io"
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
}

// NewWriter constructs a new Writer object.
func NewWriter(name string) (*Writer, error) {
	f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q: %w", name, err)
	}

	return &Writer{
		f:  f,
		zw: zip.NewWriter(f),
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
		return w.writeFile(name)
	}

	return w.writeDir(name)
}

func (w *Writer) writeFile(name string) error {
	writer, err := w.zw.Create(name)
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
		writer, err := w.zw.Create(fname)
		if err != nil {
			return fmt.Errorf("zip create failed: %w", err)
		}
		if fi.IsDir() {
			logrus.Debugf("compress dir: %v", fname)
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
	return nil
}

func (w *Writer) Close() error {
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
