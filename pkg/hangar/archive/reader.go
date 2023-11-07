package archive

import (
	"archive/tar"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/klauspost/pgzip"
	"github.com/sirupsen/logrus"
)

type Reader struct {
	f      *os.File
	tr     *tar.Reader
	gr     *pgzip.Reader
	format Format
}

func NewReader(fname string) (*Reader, error) {
	reader := &Reader{}
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	reader.f = f
	if err = reader.init(); err != nil {
		f.Close()
		return nil, err
	}

	return reader, nil
}

func (r *Reader) Format() Format {
	return r.format
}

func (r *Reader) Ls() error {
	var err error
	if err = r.reset(); err != nil {
		return err
	}

	if err = r.walk(lscb); err != nil {
		return err
	}

	return nil
}

func (r *Reader) DecompressFile(name string, dst string) error {
	name = strings.TrimPrefix(name, "./")
	logrus.Debugf("name: %v", name)

	var err error
	if dst == "" {
		dst = "."
	} else if err = os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	if err = r.reset(); err != nil {
		return err
	}

	if err = r.walk(func(h *tar.Header) error {
		headerName := strings.TrimPrefix(h.Name, "./")
		if !strings.HasPrefix(headerName, name) {
			// logrus.Debugf("skip: %v", headerName)
			return nil
		}
		baseDir := path.Dir(name)
		target := filepath.Join(dst, strings.TrimPrefix(headerName, baseDir))
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
				target, os.O_CREATE|os.O_RDWR, os.FileMode(h.Mode))
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
	}); err != nil {
		return err
	}

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
		gr     *pgzip.Reader
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
		gr, err = pgzip.NewReader(r.f)
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

func (r *Reader) reset() error {
	var err error
	if _, err = r.f.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to get set file seek offset: %w", err)
	}

	if r.gr != nil {
		if err = r.gr.Close(); err != nil {
			return err
		}
	}
	if r.tr != nil {
		r.tr = nil
	}
	if err = r.init(); err != nil {
		return err
	}

	return nil
}

func (r *Reader) walk(cb func(*tar.Header) error) error {
	for {
		header, err := r.tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if err = cb(header); err != nil {
			return err
		}
	}

	return nil
}

func lscb(header *tar.Header) error {
	var t string = " "
	switch header.Typeflag {
	case tar.TypeReg:
		t = "f"
	case tar.TypeLink:
		t = "H"
	case tar.TypeSymlink:
		t = "l"
	case tar.TypeChar:
		t = "c"
	case tar.TypeBlock:
		t = "b"
	case tar.TypeDir:
		t = "d"
	case tar.TypeFifo:
		t = "n"
	}
	fmt.Printf("%v %v\n", t, header.Name)
	return nil
}
