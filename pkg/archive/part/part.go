// part package implements a ReadWriteCloser to write data into multi parts
// and read data from different parts
package part

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

// PartHelper implements io.ReadWriteCloser to write file
// into different part (.part*), and read from it.
type PartHelper struct {
	// Filename is the file name of the compressed file
	Filename string

	// partname is the actual file name of each part
	partname string

	// file is the current file of
	file *os.File

	// Size is the maximum size of each part of tarball,
	// multi-part archive will be ignored if size is 0
	Size int

	// part is the part number of the tarball
	part int

	// writeBytes for record how many bytes were written
	// into the part of the archive
	writeBytes int

	// readBytes for record how many bytes were read
	// from the part of the archive
	readBytes int

	readEOF bool
}

// NewPartHelper create a new PartHelper
//
//	file is the filename of the compressed file (without .part* extension)
//	size is the maximum size of each part to write, 0 to disable
func NewPartHelper(file string, size int) *PartHelper {
	if size < 0 {
		size = 0
	}

	c := &PartHelper{
		Filename:   file,
		partname:   fmt.Sprintf("%s.part%d", file, 0),
		file:       nil,
		Size:       size,
		part:       0,
		writeBytes: 0,
		readBytes:  0,
		readEOF:    false,
	}
	return c
}

// Read implements io.Reader
func (c *PartHelper) Read(p []byte) (int, error) {
	if c.file == nil {
		if err := c.initRead(); err != nil {
			return 0, err
		}
	}
	if c.readEOF {
		return 0, io.EOF
	}

	start := 0
	num := 0
	var err error
	for start < len(p) {
		num, err = c.file.Read(p[start:])
		start += num
		c.readBytes += num
		if err == nil {
			// no error occurred
			continue
		} else if !errors.Is(err, io.EOF) {
			// other error occurred
			return start, fmt.Errorf("Read: %w", err)
		}

		// switch to next part
		err := c.openNextPart()
		if err != nil && errors.Is(err, os.ErrNotExist) {
			// next part does not exist, return
			c.readEOF = true
			return start, nil
		} else if err != nil {
			// other error occurred
			return start, fmt.Errorf("Read: %w", err)
		}
	}
	return start, nil
}

// Write implements io.Writer
func (c *PartHelper) Write(p []byte) (int, error) {
	if c.file == nil {
		if err := c.initWrite(); err != nil {
			return 0, err
		}
	}
	if c.Size == 0 {
		// disable part
		return c.file.Write(p)
	}

	start := 0
	end := len(p)
	for end-start > c.Size-c.writeBytes {
		end = start + c.Size - c.writeBytes
		// logrus.Infof("Write from %d to %d: %v", start, end, p[start:end])
		_, err := c.file.Write(p[start:end])
		if err != nil {
			return end, fmt.Errorf("Write: %w", err)
		}
		if err := c.createNextPart(); err != nil {
			return end, fmt.Errorf("Write: %w", err)
		}
		start = end
		end = len(p)
	}
	if end > start {
		// logrus.Infof("Write from %d to %d: %v", start, end, p[start:end])
		num, err := c.file.Write(p[start:end])
		if err != nil {
			return end, fmt.Errorf("Write: %w", err)
		}
		c.writeBytes += num
	}
	return end, nil
}

func (c *PartHelper) initRead() error {
	if c.file == nil {
		// first init
		var err error
		c.file, err = os.Open(c.partname)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("initRead: %w", err)
		} else if errors.Is(err, os.ErrNotExist) {
			// failed to open .part*
			c.file, err = os.Open(c.Filename)
			if err != nil {
				return fmt.Errorf("initRead: %w", err)
			}
			logrus.Infof("read %q", c.Filename)
		} else {
			logrus.Infof("read %q", c.partname)
		}
	}

	return nil
}

func (c *PartHelper) initWrite() error {
	if c.file == nil {
		// first init
		var err error
		c.file, err = os.Create(c.partname)
		if err != nil {
			return fmt.Errorf("initWrite: %w", err)
		}
		logrus.Infof("create %q", c.partname)
	}
	return nil
}

// nextPartExists determines whether the next part exists or not.
func (c *PartHelper) nextPartExists() bool {
	partname := fmt.Sprintf("%s.part%d", c.Filename, c.part+1)
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

// createNextPart will create a new part file,
// this method is used by Write method
// this method will reset writeBytes to 0.
func (c *PartHelper) createNextPart() error {
	partname := fmt.Sprintf("%s.part%d", c.Filename, c.part+1)
	c.file.Close() // ignore close error

	var err error
	c.file, err = os.Create(partname)
	if err != nil {
		return fmt.Errorf("createNextPart: %w", err)
	}
	c.part++
	c.partname = partname
	c.writeBytes = 0
	logrus.Infof("create %q", partname)
	return nil
}

// openNextPart will try to open next part file,
// if next part does not exists, an os.ErrNotExist error will return.
// this method will reset readBytes to 0 if open succeed.
func (c *PartHelper) openNextPart() error {
	if !c.nextPartExists() {
		return os.ErrNotExist
	}
	partname := fmt.Sprintf("%s.part%d", c.Filename, c.part+1)
	c.file.Close() // ignore close error
	var err error
	c.file, err = os.Open(partname)
	if err != nil {
		return fmt.Errorf("openNextPart: %w", err)
	}
	c.part++
	c.partname = partname
	c.readBytes = 0
	logrus.Infof("read %q", partname)

	return nil
}

// Part gets part num of PartHelper
func (c *PartHelper) Part() int {
	return c.part
}

// Close closes the current opening file and reset the status of PartHelper
func (c *PartHelper) Close() error {
	c.readBytes = 0
	c.writeBytes = 0
	c.part = 0
	c.readEOF = false

	c.partname = fmt.Sprintf("%spart%d", c.Filename, c.part)
	if c.file == nil {
		return nil
	}
	return c.file.Close()
}
