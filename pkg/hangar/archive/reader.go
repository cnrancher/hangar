package archive

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/STARRY-S/zip"
	"github.com/sirupsen/logrus"
)

type Reader struct {
	f  *os.File
	zr *zip.Reader
}

// NewReader constructs a new Archive Reader object.
// Needs to call Close() method to release resource after usage.
func NewReader(name string) (*Reader, error) {
	reader := &Reader{}
	f, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	reader.f = f
	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("fstat failed: %w", err)
	}
	reader.zr, err = zip.NewReader(f, fi.Size())
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to create zip reader: %w", err)
	}
	return reader, nil
}

func (r *Reader) Index() ([]byte, error) {
	var f *zip.File
	for _, file := range r.zr.File {
		if file.Name == IndexFileName {
			f = file
			break
		}
	}
	if f == nil {
		return nil, os.ErrNotExist
	}
	rw, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open %v in %v: %w",
			IndexFileName, r.f.Name(), err)
	}
	defer rw.Close()
	b, err := io.ReadAll(rw)
	if err != nil {
		return nil, fmt.Errorf("failed to read %v in %v: %w",
			IndexFileName, r.f.Name(), err)
	}
	return b, nil
}

// Decompress decompresses the file/directory in archive.
func (r *Reader) Decompress(name string, destination string) error {
	var file *zip.File
	for _, f := range r.zr.File {
		if f.Name != name {
			continue
		}
		file = f
		break
	}
	if file == nil {
		return os.ErrNotExist
	}

	var err error
	baseDir := path.Dir(name) + string(os.PathSeparator)
	target := filepath.Join(destination, strings.TrimPrefix(file.Name, baseDir))
	switch {
	case file.Mode().IsDir():
		if err = os.MkdirAll(target, fs.FileMode(file.Mode())); err != nil {
			return err
		}
		// Decompress all files inside the directory.
		for _, f := range r.zr.File {
			if f.Name == baseDir || !strings.HasPrefix(f.Name, baseDir) {
				continue
			}
			if err = r.Decompress(f.Name, destination); err != nil {
				return err
			}
		}
	case file.Mode().IsRegular():
		if err = os.MkdirAll(path.Dir(target), 0755); err != nil {
			return err
		}
		f, err := os.OpenFile(
			target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(file.Mode()))
		if err != nil {
			return fmt.Errorf("os.OpenFile: %w", err)
		}
		defer f.Close()
		src, err := file.Open()
		if err != nil {
			return fmt.Errorf("faled to open %q in zip: %w", file.Name, err)
		}
		defer src.Close()
		if _, err := io.Copy(f, src); err != nil {
			return fmt.Errorf("io.Copy: %w", err)
		}
	}
	logrus.Debugf("decompress: %v", target)

	return nil
}

func (r *Reader) DecompressTmp(name string) (string, error) {
	tmpDir, err := os.MkdirTemp(cacheDir, "*")
	if err != nil {
		return "", fmt.Errorf("failed to create tmp dir: %w", err)
	}
	err = r.Decompress(name, tmpDir)
	if err != nil {
		return "", err
	}
	return tmpDir, err
}

func (r *Reader) DecompressImageTmp(
	img *ImageSpec,
	imageSpecSet map[string]map[string]bool,
	blobDir string,
) (string, error) {
	if len(imageSpecSet["os"]) != 0 && !imageSpecSet["os"][img.OS] {
		return "", nil
	}
	if len(imageSpecSet["arch"]) != 0 && !imageSpecSet["arch"][img.Arch] {
		return "", nil
	}

	tmpDir, err := os.MkdirTemp(cacheDir, "*")
	if err != nil {
		return "", fmt.Errorf("failed to create tmp dir: %w", err)
	}
	// Decompress the OCI image folder.
	err = r.Decompress(img.Folder+string(os.PathSeparator), tmpDir)
	if err != nil {
		return "", fmt.Errorf("failed to decompress dir [%v]: %w",
			img.Folder, err)
	}
	return tmpDir, nil
}

func (r *Reader) Close() error {
	if r.zr != nil {
		r.zr = nil
	}
	if err := r.f.Close(); err != nil {
		return err
	}
	r.f = nil
	return nil
}

func (r *Reader) Ls() {
	for _, f := range r.zr.File {
		var t = " "
		switch {
		case f.Mode().IsRegular():
			t = "r"
		case f.Mode().IsDir():
			t = "d"
		}
		logrus.Infof(" %v %v", t, f.Name)
	}
}
