// part package implements a ReadWriteCloser to write data into multi parts
// and read data from different parts
package part

import (
	"errors"
	"fmt"
	"io"
	"os"

	"cnrancher.io/image-tools/utils"
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
	}
	return c
}

// Read implements io.Reader
func (c *PartHelper) Read(p []byte) (int, error) {
	var b byte
	var num int = 0
	for range p {
		if err := c.readOneByte(&b); err != nil {
			if errors.Is(err, io.EOF) {
				return num, err
			}
			return num, fmt.Errorf("Read: %w", err)
		}
		p[num] = b
		num++
	}

	return num, nil
}

// Write implements io.Writer
func (c *PartHelper) Write(p []byte) (int, error) {
	num := 0
	for _, v := range p {
		if err := c.writeOneByte(v); err != nil {
			return num, fmt.Errorf("Write: %w", err)
		}
		num++
	}
	return num, nil
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

	return nil
}

// writeOneByte will write one byte into file
// and switch file parts automatically.
func (c *PartHelper) writeOneByte(b byte) error {
	// initialize if the file is not open
	if c.file == nil {
		var err error
		c.file, err = os.Create(c.partname)
		if err != nil {
			return fmt.Errorf("writeOneByte: %w", err)
		}
	}
	// part will be disabled if c.Size is 0
	if c.Size == 0 {
		_, err := c.file.Write([]byte{b})
		return err
	}

	if c.writeBytes+1 > c.Size {
		if err := c.createNextPart(); err != nil {
			return fmt.Errorf("writeOneByte: %w", err)
		}
	}
	num, err := c.file.Write([]byte{b})
	if err != nil {
		return fmt.Errorf("writeOneByte: %w", err)
	}
	c.writeBytes += num
	return nil
}

// readOneByte will try to read one byte from file,
// this method will switch file part automatically.
// io.EOF will return when EOF.
func (c *PartHelper) readOneByte(b *byte) error {
	if b == nil {
		return utils.ErrNilPointer
	}
	// initialize if the file is not open
	if c.file == nil {
		var err error
		c.file, err = os.Open(c.partname)
		if err != nil {
			return fmt.Errorf("readOneByte: %w", err)
		}
	}

	// try to read one byte from current file part
	buff := make([]byte, 1)
	var err error
	_, err = c.file.Read(buff)
	if err == nil {
		// read succeed
		*b = buff[0]
		return nil
	} else if errors.Is(err, io.EOF) {
		// EOF, try to switch to next part
	} else {
		// other error occured
		return fmt.Errorf("readOneByte: %w", err)
	}

	err = c.openNextPart()
	if err == nil {
		// open next file part succeed
	} else if os.IsNotExist(err) {
		// next part does not exists
		return io.EOF
	} else {
		// other error occured
		return fmt.Errorf("readOneByte: %w", err)
	}

	_, err = c.file.Read(buff)
	if err != nil {
		return fmt.Errorf("readOneByte: %w", err)
	}
	*b = buff[0]
	return nil
}

// Part gets part num of PartHelper
func (c *PartHelper) Part() int {
	return c.part
}

// Close closes the current opening file
func (c *PartHelper) Close() error {
	if c.file == nil {
		return nil
	}
	return c.file.Close()
}
