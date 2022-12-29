package part

import (
	"errors"
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
	if num, err := c.Write(data); num != 8 || err != nil {
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

func Test_Write_Part(t *testing.T) {
	c := NewPartHelper(
		"test/test.txt",
		3, // split part each 3 bytes
	)
	if num, err := c.Write(data); num != 8 || err != nil {
		t.Error("Num is: ", num)
		t.Error(err)
	}
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
	if num != 8 {
		t.Error("Num is: ", num)
	}
	if !errors.Is(err, io.EOF) {
		t.Error(err)
	}
	t.Log(string(buff))
	if err := c.Close(); err != nil {
		t.Error(err)
	}
}
