package mirror

import (
	"io"
	"testing"

	"github.com/cnrancher/hangar/pkg/skopeo"
)

func Test_StartSave(t *testing.T) {
	var mirror *Mirror = nil
	var err error

	if err = mirror.StartSave(); err == nil {
		t.Error("StartSave failed")
	}

	mirror = NewMirror(&MirrorOptions{
		Mode: 0,
	})
	if err = mirror.StartSave(); err == nil {
		t.Error("StartSave failed")
	}
	mirror = NewMirror(&MirrorOptions{
		Source:      "",
		Destination: "",
		Tag:         "",
		ArchList:    []string{"amd64", "arm64"},
		OsList:      []string{"windows", "linux"},
		Directory:   "",
		Mode:        MODE_SAVE,
	})

	// fake skopeo inspect / skopeo copy function
	fake := func(a string, in io.Reader, out io.Writer, p ...string) error {
		o, err := testFs.ReadFile(TestS2V2ListFileName)
		if err != nil {
			return err
		}
		if out != nil {
			out.Write(o)
		}
		return nil
	}
	skopeo.RunCommandFunc = fake
	if err = mirror.StartSave(); err != nil {
		t.Error(err)
	}

	if mirror.Saved() != 2 {
		t.Error("Saved failed")
	}
}