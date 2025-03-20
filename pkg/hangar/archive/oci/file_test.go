package oci

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/cnrancher/hangar/pkg/utils"
)

var file_opts = []*FileOptions{
	// Remote URL
	{
		CommonOpts: CommonOpts{
			InsecureSkipVerify: false,
			SystemContext:      utils.CopySystemContext(nil),
			Policy:             policy,
		},
		URL: "https://charts.rancher.cn/2.10-prime/latest/rancher-2.10.1-ent.tgz",
	},
	// Local File (skip if not exists)
	{
		CommonOpts: CommonOpts{
			InsecureSkipVerify: false,
			SystemContext:      utils.CopySystemContext(nil),
			Policy:             policy,
		},
		URL: "./test/.gitignore",
	},
}

func Test_File(t *testing.T) {
	for _, o := range file_opts {
		if strings.HasPrefix(o.URL, "./") {
			if _, err := os.Stat(o.URL); err != nil {
				t.Logf("skip test %q", o.URL)
				continue
			}
		}
		f := NewFile(o)
		defer f.Cleanup()
		if err := f.Fetch(context.TODO()); err != nil {
			t.Errorf("faied to fetch %q: %v", f.url, err)
			return
		}
		t.Logf("file %q cache dir: %q\n", f.url, f.cacheDir)

		img, err := f.image()
		if err != nil {
			t.Error(err)
			return
		}
		t.Logf("%q:\n%v\n", f.cacheDir, utils.ToJSON(img))
		t.Logf("------------------------------\n")
	}
}
