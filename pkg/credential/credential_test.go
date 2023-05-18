package credential_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cnrancher/hangar/pkg/credential"
	"github.com/stretchr/testify/assert"
)

func Test_GetRegistryCredential(t *testing.T) {
	if os.Getenv("DRONE_COMMIT_SHA") != "" {
		t.Logf("SKIP THIS TEST RUNNING IN CI")
		return
	}

	_, err := os.Stat(filepath.Join(os.Getenv("HOME"), ".docker/config.json"))
	if os.IsNotExist(err) {
		return
	}

	u, p, err := credential.GetRegistryCredential("")
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, u)
	assert.NotEmpty(t, p)
}
