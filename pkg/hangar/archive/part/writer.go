package part

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

const (
	BYTE int64 = 1
	KB   int64 = 1024 * BYTE
	MB   int64 = 1024 * KB
	GB   int64 = 1024 * MB

	MAX_PART_NUM  int64 = 0xFFFF
	MIN_PART_SIZE int64 = 1 * MB
	MAX_PART_SIZE int64 = 100 * GB
)

var (
	ErrToManyParts      = errors.New("too many parts, should smaller than 65535")
	ErrPartSizeTooSmall = errors.New("part size too small, mim part size allowed is 1MB")
	ErrPartSizeTooLarge = errors.New("part size too large, max part size allowed is 10TB")
)

type PartWriter interface {
	io.WriteCloser

	FileName() string
	PartName() string
	Size() int64
	Part() int
}

type writer struct {
	filename   string
	partname   string
	file       *os.File
	size       int64
	part       int
	writeBytes int64
}

func NewPartWriter(fname string, size int64) (PartWriter, error) {
	if size > MAX_PART_SIZE {
		return nil, ErrPartSizeTooLarge
	}
	if size != 0 && size < MIN_PART_SIZE {
		return nil, ErrPartSizeTooSmall
	}

	w := &writer{
		filename:   fname,
		partname:   "",
		file:       nil,
		size:       size,
		part:       0,
		writeBytes: 0,
	}
	if size == 0 {
		w.partname = fname
	} else {
		w.partname = fmt.Sprintf("%s.part%d", fname, 0)
	}
	var err error
	w.file, err = os.OpenFile(w.partname, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	logrus.Infof("create %q", w.partname)

	return w, nil
}

func (w *writer) FileName() string {
	return w.filename
}

func (w *writer) PartName() string {
	return w.partname
}

func (w *writer) Size() int64 {
	return w.size
}

func (w *writer) Part() int {
	return w.part
}

func (w *writer) Write(p []byte) (int, error) {
	if w.file == nil {
		return 0, fmt.Errorf("file closed")
	}
	if w.size == 0 {
		// disable part
		return w.file.Write(p)
	}

	var (
		start int64
		end   = int64(len(p))
	)
	for int64(end-start) > w.size-w.writeBytes {
		end = start + w.size - w.writeBytes
		_, err := w.file.Write(p[start:end])
		if err != nil {
			return int(end), err
		}
		if err := w.createNextPart(); err != nil {
			return int(end), err
		}
		start = end
		end = int64(len(p))
	}
	if end > start {
		num, err := w.file.Write(p[start:end])
		if err != nil {
			return int(end), err
		}
		w.writeBytes += int64(num)
	}
	return int(end), nil
}

func (w *writer) Close() error {
	if w.file == nil {
		return nil
	}
	if err := w.file.Close(); err != nil {
		return err
	}
	w.file = nil
	return nil
}

func (w *writer) createNextPart() error {
	partname := fmt.Sprintf("%s.part%d", w.filename, w.part+1)
	if err := w.file.Close(); err != nil {
		return err
	}

	var err error
	w.file, err = os.Create(partname)
	if err != nil {
		return fmt.Errorf("createNextPart: %w", err)
	}
	w.part++
	w.partname = partname
	w.writeBytes = 0
	logrus.Infof("create %q", partname)
	return nil
}
