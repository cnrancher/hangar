package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_CompareBuildManifest(t *testing.T) {
	var src BuildManifestListParam
	var dst BuildManifestListParam
	if CompareBuildManifests(nil, nil) {
		t.Error("CompareBuildxManifest 1 failed")
	}
	src = BuildManifestListParam{
		Digest: "abcabc",
		Platform: BuildManifestListPlatform{
			Architecture: "amd64",
			OS:           "linux",
			OsVersion:    "1.0.0",
			Variant:      "",
		},
	}
	dst = BuildManifestListParam{
		Digest: "abcabc",
		Platform: BuildManifestListPlatform{
			Architecture: "amd64",
			OS:           "linux",
			OsVersion:    "1.0.0",
			Variant:      "",
		},
	}
	if !compareBuildManifest(&src, &dst) {
		t.Error("CompareBuildxManifest 2 failed")
	}
	dst = BuildManifestListParam{
		Digest: "ffffff",
		Platform: BuildManifestListPlatform{
			Architecture: "arm64",
			OS:           "Windows",
			OsVersion:    "2.0.0",
			Variant:      "v8",
		},
	}
	if compareBuildManifest(&src, &dst) {
		t.Error("CompareBuildxManifest 3 failed")
	}
}

func Test_BuildManifestExists(t *testing.T) {
	assert.False(t, BuildManifestExists(nil, BuildManifestListParam{}))
	assert.False(t, BuildManifestExists(
		[]BuildManifestListParam{}, BuildManifestListParam{}))
	params := []BuildManifestListParam{
		{
			Digest: "sha256:aaa",
			Platform: BuildManifestListPlatform{
				Architecture: "amd64",
			},
		},
	}
	assert.False(t, BuildManifestExists(
		params, BuildManifestListParam{}))
	assert.False(t, BuildManifestExists(
		params, BuildManifestListParam{
			Digest: "sha256:bbb",
			Platform: BuildManifestListPlatform{
				Architecture: "amd64",
			},
		}))
	assert.True(t, BuildManifestExists(
		params, BuildManifestListParam{
			Digest: "sha256:aaa",
			Platform: BuildManifestListPlatform{
				Architecture: "amd64",
			},
		}))
}

func Test_PushManifest(t *testing.T) {
	// if os.Getenv("DRONE_COMMIT_SHA") != "" {
	// 	t.Logf("SKIP THIS TEST RUNNING IN CI")
	// 	return
	// }

	// // EDIT REGISTRY SERVER & DIGEST IN PARAM MANUALLY
	// // THIS TEST NEED TO RUN MANUALLY
	// if true {
	// 	return
	// }

	// // u, p, err := credential.GetRegistryCredential("h2.hxstarrys.me:30003")
	// // if err != nil {
	// // 	return
	// // }
	// f, err := os.Open("test/manifest.json")
	// if err != nil {
	// 	if os.IsNotExist(err) {
	// 		return
	// 	}
	// 	t.Error(err)
	// 	return
	// }
	// b, _ := io.ReadAll(f)
	// err = PushManifest(
	// 	"docker://h2.hxstarrys.me:30003/library/nginx:1.22",
	// 	u, p, b)
	// if err != nil {
	// 	t.Error(err)
	// 	return
	// }
}

func Test_BuildManifestList(t *testing.T) {
	if os.Getenv("DRONE_COMMIT_SHA") != "" {
		t.Logf("SKIP THIS TEST RUNNING IN CI")
		return
	}

	// EDIT REGISTRY SERVER & DIGEST IN PARAM MANUALLY
	// THIS TEST NEED TO RUN MANUALLY
	if true {
		return
	}

	param := []BuildManifestListParam{
		{
			Digest: "sha256:9081064712674ffcff7b7bdf874c75bcb8e5fb933b65527026090dacda36ea8b",
			Platform: BuildManifestListPlatform{
				Architecture: "amd64",
				OS:           "linux",
				OsVersion:    "",
				Variant:      "",
			},
		},
		{
			Digest: "sha256:cf4ffe24f08a167176c84f2779c9fc35c2f7ce417b411978e384cbe63525b420",
			Platform: BuildManifestListPlatform{
				Architecture: "arm64",
				OS:           "linux",
				OsVersion:    "",
				Variant:      "",
			},
		},
	}
	s2, err := BuildManifestList(
		"h2.hxstarrys.me:30003/library/nginx",
		"admin", "Harbor12345",
		param)
	if err != nil {
		t.Error(err)
		return
	}

	d, _ := json.MarshalIndent(s2, "", "  ")
	fmt.Printf("%s\n", string(d))
}
