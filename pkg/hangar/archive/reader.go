package archive

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

type header struct {
	*tar.Header

	offset int64
}

type Reader struct {
	f      *os.File
	tr     *tar.Reader
	gr     *gzip.Reader
	format Format

	filemap map[string]*header
}

// NewReader constructs a new Archive Reader object.
// Needs to call Close() method to release resource after usage.
// This method will take much time to build file offset cache.
func NewReader(fname string) (*Reader, error) {
	reader := &Reader{
		filemap: make(map[string]*header),
	}
	f, err := os.Open(fname)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	reader.f = f
	if err = reader.init(); err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to init hangar archive reader: %w", err)
	}
	if err = reader.buildCache(); err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to build cache for hangar archive file: %w", err)
	}

	return reader, nil
}

func (r *Reader) Format() Format {
	return r.format
}

func (r *Reader) buildCache() error {
	var err error

	if r.gr != nil {
		r.gr.Multistream(false)
	}
	r.f.Sync()
	if err = r.walk(func(h *tar.Header) error {
		offset, err := r.f.Seek(0, io.SeekCurrent)
		if err != nil {
			return fmt.Errorf("failed to get current file seek: %w", err)
		}
		r.filemap[h.Name] = &header{
			Header: h,
			offset: offset,
		}
		logrus.Debugf("Get offset of file %q: %v", h.Name, offset)

		return nil
	}); err != nil {
		return fmt.Errorf("walk failed: %w", err)
	}

	return nil
}

func (r *Reader) getIndexHeader() (*header, error) {
	indexName := IndexFileName
	h, ok := r.filemap[indexName]
	if !ok {
		return nil, fmt.Errorf("getIndexOffset: unable to find file %q from tarball", indexName)
	}
	return h, nil
}

// DecompressFile decompresses the file/directory in archive to the destination
// directory. This function use cache logic to increase performence.
func (r *Reader) DecompressFile(name string, destination string) error {
	logrus.Debugf("DecompressFile name: %v", name)

	var err error
	h, ok := r.filemap[name]
	if !ok {
		return os.ErrNotExist
	}

	logrus.Debugf("find offset [%v] of file [%v]", h.offset, name)
	_, err = r.f.Seek(h.offset, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek: %w", err)
	}

	baseDir := path.Dir(name)
	target := filepath.Join(destination, strings.TrimPrefix(h.Name, baseDir))
	switch h.Typeflag {
	case tar.TypeDir:
		if err = os.MkdirAll(target, fs.FileMode(h.Mode)); err != nil {
			return err
		}
	case tar.TypeReg:
		if err = os.MkdirAll(path.Dir(target), 0755); err != nil {
			return err
		}
		f, err := os.OpenFile(
			target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(h.Mode))
		if err != nil {
			return fmt.Errorf("os.OpenFile: %w", err)
		}
		defer f.Close()
		if _, err := io.Copy(f, r.tr); err != nil {
			return fmt.Errorf("io.Copy: %w", err)
		}
	}
	logrus.Debugf("decompress: %v", target)

	return nil
}

func (r *Reader) DecompressFileTmp(name string) (string, error) {
	err := os.MkdirAll(cacheDir, 0755)
	if err != nil {
		return "", fmt.Errorf("mkdir: %v", err)
	}
	tmpDir, err := os.MkdirTemp(cacheDir, "*")
	if err != nil {
		return "", fmt.Errorf("failed to create tmp dir: %v", err)
	}
	err = r.DecompressFile(name, tmpDir)
	if err != nil {
		return "", err
	}
	return tmpDir, err
}

func (r *Reader) Close() error {
	if r.gr == nil {
		return nil
	}
	if err := r.gr.Close(); err != nil {
		return err
	}
	r.gr = nil
	r.tr = nil
	if err := r.f.Close(); err != nil {
		return err
	}
	r.f = nil
	return nil
}

func (r *Reader) init() error {
	var (
		gr     *gzip.Reader
		tr     *tar.Reader
		format Format = TAR
	)
	tr = tar.NewReader(r.f)
	_, err := tr.Next()
	if err != nil {
		// restore previous file offset.
		_, err = r.f.Seek(0, io.SeekStart)
		if err != nil {
			return err
		}
		err = r.f.Sync()
		if err != nil {
			return err
		}
		// file is not tar format, check if it's gzip format.
		gr, err = gzip.NewReader(r.f)
		if err != nil {
			return fmt.Errorf("gzip error: %w", err)
		}
		tr = tar.NewReader(gr)
		format = GZIP
	} else {
		// restore previous file offset.
		_, err = r.f.Seek(0, io.SeekStart)
		if err != nil {
			return err
		}
		tr = tar.NewReader(r.f)
	}

	r.tr = tr
	r.gr = gr
	r.format = format
	return nil
}

func (r *Reader) walk(cb func(*tar.Header) error) error {
	for {
		header, err := r.tr.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}

		if err = cb(header); err != nil {
			return err
		}
	}

	return nil
}

func (r *Reader) Ls() {
	for _, h := range r.filemap {
		var t = " "
		switch h.Typeflag {
		case tar.TypeReg:
			t = "r"
		case tar.TypeDir:
			t = "d"
		}
		logrus.Infof(" %v %v", t, h.Name)
	}
}
