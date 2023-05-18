package harbor_test

import (
	"os"
	"testing"

	"github.com/cnrancher/hangar/pkg/config"
	"github.com/cnrancher/hangar/pkg/credential"
	"github.com/cnrancher/hangar/pkg/harbor"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	config.Set("tls-verify", false)
}

func Test_ProjectExists(t *testing.T) {
	var url string = ""
	_ = url

	// EDIT THIS LINE MANUALLY
	// url = "https://h2.hxstarrys.me:30003/api/v2.0/projects"
	if url == "" {
		return
	}
	u, p, err := credential.GetRegistryCredential("h2.hxstarrys.me:30003")
	if err != nil {
		t.Error(err)
		return
	}

	if _, err := harbor.ProjectExists("priv", url, u, p); err != nil {
		t.Error(err)
	}
}

func Test_CreateProject(t *testing.T) {
	if os.Getenv("DRONE_COMMIT_SHA") != "" {
		t.Logf("SKIP THIS TEST RUNNING IN CI")
		return
	}

	// EDIT THIS LINE MANUALLY
	// url := "https://h2.hxstarrys.me:30003/api/v2.0/projects"
	url := ""

	if url == "" {
		return
	}
	u, p, err := credential.GetRegistryCredential("h2.hxstarrys.me:30003")
	if err != nil {
		t.Error(err)
		return
	}
	if err := harbor.CreateProject("name", url, u, p); err != nil {
		t.Error(err)
	}
}
