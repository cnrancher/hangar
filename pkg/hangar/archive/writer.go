package archive

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type WriteMode int

const (
	// CreateTrunc creates the new archive file.
	// It will truncate if the file already exists.
	CreateTrunc WriteMode = iota

	// CreateAppend appends new data to existing tarball,
	// Append mode only supports TAR format, the compressed format is not
	// supported.
	CreateAppend
)

type Writer struct {
	fname  string
	f      *os.File
	tw     *tar.Writer
	gw     *gzip.Writer
	format Format
	mode   WriteMode
}

// NewWriter constructs a new Writer object.
func NewWriter(fname string, format Format, mode WriteMode) (*Writer, error) {
	if format.String() == "" || format.String() == "zstd" {
		return nil, fmt.Errorf("unsupported format: %v", format.String())
	}
	if format != TAR && mode == CreateAppend {
		return nil, fmt.Errorf(
			"append mode is not supported for %q compressed format", format.String())
	}

	writer := &Writer{
		fname: fname,
	}
	var flag = os.O_RDWR | os.O_CREATE
	switch mode {
	case CreateAppend:
		// Reason of not using O_APPEND flag is:
		// https://stackoverflow.com/questions/18323995/golang-append-file-to-an-existing-tar-archive
		// flag |= os.O_APPEND
	default:
		mode = CreateTrunc
		flag |= os.O_TRUNC
	}

	f, err := os.OpenFile(fname, flag, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q: %w", fname, err)
	}
	if mode == CreateAppend {
		// Ignore the error here as the file may have just been created
		// and its length is 0.
		end, _ := f.Seek(-1024, io.SeekEnd)
		logrus.Debugf("Seek file to end offset: %v", end)
	}

	writer.f = f
	writer.mode = mode
	if err := writer.init(format); err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to init hangar archive writer: %w", err)
	}
	writer.format = format

	return writer, nil
}

func (w *Writer) init(format Format) error {
	var (
		gw *gzip.Writer
		tw *tar.Writer
	)

	switch format {
	case GZIP:
		gw = gzip.NewWriter(w.f)
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
	return w.tw.Flush()
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

	return w.tw.Flush()
}

func (w *Writer) RemoveIndex() error {
	offset, err := w.getIndexOffset()
	if err != nil {
		return err
	}
	_, err = w.f.Seek(offset, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to remove index: %w", err)
	}
	return nil
}

// WriteIndex writes the index json file into the end of the tar archive.
func (w *Writer) WriteIndex(index *Index) error {
	var err error
	data, err := json.Marshal(index)
	if err != nil {
		return fmt.Errorf("writeIndex: %w", err)
	}
	err = w.tw.WriteHeader(&tar.Header{
		Typeflag: 0,
		Name:     IndexFileName,
		Linkname: "",
		Size:     int64(len(data)),
		Mode:     0644,
		Uid:      1000,
		Gid:      1000,
		ModTime:  time.Now(),
		Format:   0,
	})
	if err != nil {
		return fmt.Errorf("writeIndex: tar write header failed: %w", err)
	}
	_, err = w.tw.Write(data)
	if err != nil {
		return fmt.Errorf("writeIndex: tar write failed: %w", err)
	}
	logrus.Infof("Write index file %q, size %.2fK",
		IndexFileName, float32(len(data))/1024)
	return nil
}

func (w *Writer) getIndexOffset() (int64, error) {
	if w.mode != CreateAppend {
		return 0, fmt.Errorf("getIndexOffset: writer is not in CreateAppend mode")
	}
	// Backup current file seek offset.
	currentSeek, err := w.f.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, fmt.Errorf("getIndexOffset: failed to get current seek: %w", err)
	}
	defer func() error {
		_, err := w.f.Seek(currentSeek, io.SeekStart)
		if err != nil {
			return fmt.Errorf("getIndexOffset: failed to restore seek: %w", err)
		}
		return nil
	}()

	_, err = w.f.Seek(0, io.SeekStart)
	if err != nil {
		return 0, fmt.Errorf("getIndexOffset: seek failed: %w", err)
	}
	tr := tar.NewReader(w.f)
	reader := &Reader{
		tr: tr,
	}
	reader.buildCache()
	h, err := reader.getIndexHeader()
	if err != nil {
		return 0, err
	}
	return h.offset, nil
}

func (w *Writer) Close() error {
	var err error
	if w.tw != nil {
		err = w.tw.Close()
		if err != nil {
			return err
		}
		w.tw = nil
	}
	if w.gw != nil {
		err = w.gw.Close()
		if err != nil {
			return err
		}
		w.gw = nil
	}
	return nil
}
