package registry

import "testing"

func Test_CreateHarborProject(t *testing.T) {
	var url string
	// EDIT THIS LINE MANUALLY
	// url = "https://harbor2.private.io/api/v2.0/projects"

	if url == "" {
		return
	}
	if err := CreateHarborProject("name", url); err != nil {
		t.Error(err)
	}
}
