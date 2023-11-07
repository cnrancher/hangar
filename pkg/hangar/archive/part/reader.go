package part

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

type PartReader interface {
	io.ReadCloser

	// FileName returns the name of the reading file
	// (without .partN extention)
	FileName() string

	// PartName returns the name of the actual reading file.
	// (with .partN extention if the file is splitted)
	PartName() string

	// Part returns the index number of the current reading file parts.
	// (begins with 0)
	Part() int

	// IsPart returns whether the reading file is splitted to parts.
	IsPart() bool

	// ReadEOF returns whether all of the files were being read.
	ReadEOF() bool
}

type reader struct {
	filename  string
	partname  string
	file      *os.File
	part      int
	readBytes int64
	isPart    bool
	readEOF   bool
}

func NewPartReader(fname string) (PartReader, error) {
	r := &reader{
		filename:  fname,
		partname:  fmt.Sprintf("%s.part%d", fname, 0),
		file:      nil,
		part:      0,
		readBytes: 0,
		isPart:    false,
		readEOF:   false,
	}
	if fi, err := os.Stat(r.partname); err == nil {
		if !fi.Mode().IsRegular() {
			return nil, fmt.Errorf("%v is not a regular file", r.partname)
		}
		r.file, err = os.Open(r.partname)
		if err != nil {
			return nil, err
		}
		r.isPart = true
	} else if fi, err := os.Stat(r.filename); err == nil {
		if !fi.Mode().IsRegular() {
			return nil, fmt.Errorf("%v is not a regular file", r.filename)
		}
		r.file, err = os.Open(r.filename)
		if err != nil {
			return nil, err
		}
		r.partname = r.filename
		r.isPart = false
	}

	return r, nil
}

func (r *reader) Read(b []byte) (int, error) {
	if r.file == nil {
		return 0, fmt.Errorf("file closed")
	}
	if !r.isPart {
		return r.file.Read(b)
	}

	if r.readEOF {
		return 0, io.EOF
	}

	var (
		start int
		num   int
		err   error
	)
	for start < len(b) {
		num, err = r.file.Read(b[start:])
		start += num
		r.readBytes += int64(num)
		if err == nil {
			continue
		} else if !errors.Is(err, io.EOF) {
			return start, err
		}

		// switch to next part
		err := r.openNextPart()
		if err != nil && errors.Is(err, os.ErrNotExist) {
			// next part does not exist, return
			r.readEOF = true
			return start, nil
		} else if err != nil {
			// other error occurred
			return start, err
		}
	}
	return start, nil
}

// nextPartExists determines whether the next part exists or not.
func (r *reader) nextPartExists() bool {
	partname := fmt.Sprintf("%s.part%d", r.filename, r.part+1)
	info, err := os.Stat(partname)
	if err == nil {
		// next part exists
	} else if os.IsNotExist(err) {
		// next part does not exist
		return false
	} else {
		// failed to check file status
		return false
	}
	if info.IsDir() {
		// next part is a invalid file
		return false
	}
	return true
}

// openNextPart will try to open next part file,
// if next part does not exists, an os.ErrNotExist error will return.
// this method will reset readBytes to 0 if open succeed.
func (r *reader) openNextPart() error {
	if !r.nextPartExists() {
		return os.ErrNotExist
	}
	var err error
	partname := fmt.Sprintf("%s.part%d", r.filename, r.part+1)
	if err = r.file.Close(); err != nil {
		return err
	}
	r.file = nil
	r.file, err = os.Open(partname)
	if err != nil {
		return fmt.Errorf("openNextPart: %w", err)
	}
	r.part++
	r.partname = partname
	r.readBytes = 0
	logrus.Infof("Reading %q", partname)

	return nil
}

// Close closes the current opening file and reset the status of reader.
func (r *reader) Close() error {
	if r.file == nil {
		return nil
	}
	if err := r.file.Close(); err != nil {
		return err
	}
	r.readBytes = 0
	r.part = 0
	r.readEOF = false
	r.partname = ""

	return nil
}

func (r *reader) FileName() string {
	return r.filename
}

func (r *reader) PartName() string {
	return r.partname
}

func (r *reader) Part() int {
	return r.part
}

func (r *reader) IsPart() bool {
	return r.isPart
}

func (r *reader) ReadEOF() bool {
	return r.readEOF
}
