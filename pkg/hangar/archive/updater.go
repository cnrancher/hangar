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

// Updater is the updater for update hangar zip archive.
type Updater struct {
	f     *os.File
	zu    *zip.Updater
	index *Index
}

// NewUpdater constructs a new Updater object.
func NewUpdater(name string) (*Updater, error) {
	f, err := os.OpenFile(name, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open %q: %w", name, err)
	}
	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to get %q stat: %w", name, err)
	}

	zr, err := zip.NewReader(f, fi.Size())
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to create zip reader: %w", err)
	}
	index, err := initIndexFile(zr)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to init zip index: %w", err)
	}
	zu, err := zip.NewUpdater(f)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to create zip updater: %w", err)
	}
	return &Updater{
		f:     f,
		zu:    zu,
		index: index,
	}, nil
}

func initIndexFile(zr *zip.Reader) (*Index, error) {
	var f *zip.File
	for _, file := range zr.File {
		if file.Name == IndexFileName {
			f = file
			break
		}
	}
	if f == nil {
		return nil, fmt.Errorf("failed to find %q from zip file", IndexFileName)
	}
	r, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open %q from zip file", IndexFileName)
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read %q from zip file", IndexFileName)
	}
	index, err := UnmarshalIndex(b)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal index file: %w", err)
	}
	if err := CompareIndexVersion(index); err != nil {
		return nil, err
	}
	return index, nil
}

func (u *Updater) Index() *Index {
	return u.index
}

func (u *Updater) SetIndex(i *Index) {
	u.index = i
}

func (u *Updater) UpdateIndex() error {
	var err error
	data, err := json.Marshal(u.index)
	if err != nil {
		return fmt.Errorf("updateIndex: %w", err)
	}
	writer, err := u.zu.AppendHeader(&zip.FileHeader{
		Name:   IndexFileName,
		Method: zip.Store,
	}, zip.APPEND_MODE_OVERWRITE)
	if err != nil {
		return fmt.Errorf("updateIndex: failed to append file in zip: %w", err)
	}
	_, err = writer.Write(data)
	if err != nil {
		return fmt.Errorf("updateIndex: zip write failed: %w", err)
	}
	logrus.Infof("Updated index file %q of %q, size %.2fK",
		IndexFileName, u.f.Name(), float32(len(data))/1024)
	return nil
}

// Append will overwrite from the existing index.json file in zip archive.
func (u *Updater) Append(name string) error {
	fi, err := os.Stat(name)
	if err != nil {
		return err
	}
	mode := fi.Mode()
	if mode.IsRegular() {
		return u.appendFile(name, fi)
	}
	return u.appendDir(name)
}

func (u *Updater) appendFile(name string, fi fs.FileInfo) error {
	writer, err := u.zu.AppendHeader(&zip.FileHeader{
		Name:     name,
		Method:   zip.Store,
		Modified: fi.ModTime(),
	}, zip.APPEND_MODE_OVERWRITE)
	if err != nil {
		return fmt.Errorf("zip append failed: %w", err)
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

func (u *Updater) appendDir(base string) error {
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
		writer, err := u.zu.AppendHeader(&zip.FileHeader{
			Name:     fname,
			Method:   zip.Store,
			Modified: fi.ModTime(),
		}, zip.APPEND_MODE_OVERWRITE)
		if err != nil {
			return fmt.Errorf("zip append failed: %w", err)
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
		logrus.Debugf("appendDir compress file: %v", fname)
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk: %w", err)
	}
	return nil
}

func (u *Updater) Close() error {
	if u == nil {
		return nil
	}
	if u.zu != nil {
		if err := u.zu.Close(); err != nil {
			return err
		}
		u.zu = nil
	}
	if u.f != nil {
		if err := u.f.Close(); err != nil {
			return err
		}
		u.f = nil
	}
	return nil
}
