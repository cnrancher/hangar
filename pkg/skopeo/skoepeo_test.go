package skopeo_test

import (
	"os"
	"testing"

	"github.com/cnrancher/hangar/pkg/skopeo"
)

func Test_Installed(t *testing.T) {
	if os.Getenv("DRONE_COMMIT_SHA") != "" {
		t.Logf("SKIP THIS TEST RUNNING IN CI")
		return
	}

	err := skopeo.Installed()
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Error(err)
	}
}
