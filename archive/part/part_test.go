package part

import (
	"io"
	"os"
	"testing"
)

var _ io.ReadWriteCloser = &PartHelper{}

var data = []byte{'0', '1', '2', '3', '4', '5', '6', '7'}

func Test_Write(t *testing.T) {
	c := NewPartHelper(
		"test/test.txt",
		0, // disable part mode, write all data into 'test/test.txt.part0'
	)
	if num, err := c.Write(data); num != len(data) || err != nil {
		t.Error("Num is: ", num)
		t.Error(err)
	}
	if err := c.Close(); err != nil {
		t.Error(err)
	}

	// check data in 'test/test.txt.part0'
	f, err := os.Open("test/test.txt.part0")
	if err != nil {
		t.Error(err)
		return
	}
	buff := make([]byte, 16)
	num, err := f.Read(buff)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(string(buff))
	if num != len(data) {
		t.Errorf("Test_Write failed: num: %d", num)
		return
	}
	for i := 0; i < num; i++ {
		if buff[i] != data[i] {
			t.Errorf("Test_Write failed %d", i)
		}
	}
}

func Test_Write_Byte(t *testing.T) {
	c := NewPartHelper(
		"test/test.txt",
		3, // split part
	)
	// write 2 bytes firstly
	t.Log("Write 2 bytes")
	if num, err := c.Write([]byte{'1', '2'}); num != 2 || err != nil {
		t.Error("Num is: ", num)
		t.Error(err)
	}
	if c.writeBytes != 2 {
		t.Error("writeBytes is: ", c.writeBytes)
	}
	if c.partname != "test/test.txt.part0" {
		t.Error("partname is: ", c.partname)
	}
	if c.part != 0 {
		t.Error("part is: ", c.part)
	}
	// write one byte
	t.Log("Write 1 byte")
	if num, err := c.Write([]byte{'3'}); num != 1 || err != nil {
		t.Error("Num is: ", num)
		t.Error(err)
	}
	if c.writeBytes != 3 {
		t.Error("writeBytes is: ", c.writeBytes)
	}
	if c.partname != "test/test.txt.part0" {
		t.Error("partname is: ", c.partname)
	}
	if c.part != 0 {
		t.Error("part is: ", c.part)
	}
	// write one byte
	t.Log("Write 1 byte")
	if num, err := c.Write([]byte{'4'}); num != 1 || err != nil {
		t.Error("Num is: ", num)
		t.Error(err)
	}
	if c.writeBytes != 1 {
		t.Error("writeBytes is: ", c.writeBytes)
	}
	if c.partname != "test/test.txt.part1" {
		t.Error("partname is: ", c.partname)
	}
	if c.part != 1 {
		t.Error("part is: ", c.part)
	}
	if err := c.Close(); err != nil {
		t.Error(err)
	}
}

func Test_Write_Part(t *testing.T) {
	c := NewPartHelper(
		"test/test.txt",
		3, // split part
	)
	if num, err := c.Write(data); num != len(data) || err != nil {
		t.Error("Num is: ", num)
		t.Error(err)
	}
	if err := c.Close(); err != nil {
		t.Error(err)
	}
}

func Test_Read_Byte(t *testing.T) {
	c := NewPartHelper(
		"test/test.txt",
		0, // size will be ignored when read
	)
	// read 1 byte ('0')
	buff := make([]byte, 1)
	num, err := c.Read(buff)
	if num != 1 {
		t.Error("Num is: ", num)
	}
	if err != nil {
		t.Error(err)
	}
	t.Log(string(buff))
	// read 1 byte ('1')
	num, err = c.Read(buff)
	if num != 1 {
		t.Error("Num is: ", num)
	}
	if err != nil {
		t.Error(err)
	}
	t.Log(string(buff))
	if c.readBytes != 2 {
		t.Error("readBytes should be 2 but got: ", c.readBytes)
	}
	// read 1 byte ('2')
	num, err = c.Read(buff)
	if num != 1 {
		t.Error("Num is: ", num)
	}
	if err != nil {
		t.Error(err)
	}
	if c.readBytes != 3 {
		t.Error("readBytes should be 3 but got: ", c.readBytes)
	}
	t.Log(string(buff))
	// read 1 byte ('3')
	num, err = c.Read(buff)
	if num != 1 {
		t.Error("Num is: ", num)
	}
	if err != nil {
		t.Error(err)
	}
	if c.readBytes != 1 {
		t.Error("readBytes should be 1 but got: ", c.readBytes)
	}
	t.Log(string(buff))

	if err := c.Close(); err != nil {
		t.Error(err)
	}
}

func Test_Read_Part(t *testing.T) {
	c := NewPartHelper(
		"test/test.txt",
		0, // size will be ignored when read
	)
	buff := make([]byte, 16)
	num, err := c.Read(buff)
	if num != len(data) {
		t.Error("Num is: ", num)
	}
	if err != nil {
		t.Error(err)
	}
	t.Log(string(buff))
	if err := c.Close(); err != nil {
		t.Error(err)
	}
}
