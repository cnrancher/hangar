package oci

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/containers/image/v5/signature"
)

var policy = &signature.Policy{
	Default: []signature.PolicyRequirement{
		signature.NewPRInsecureAcceptAnything(),
	},
	Transports: make(map[string]signature.PolicyTransportScopes),
}

var opts = []*ChartOptions{
	// Repository URL
	{
		URL:                "https://charts.rancher.cn/2.10-prime/latest/",
		Name:               "rancher",
		Version:            "2.10.3-ent",
		InsecureSkipVerify: false,
		SystemContext:      utils.CopySystemContext(nil),
		Policy:             policy,
	},
	// Tarball URL
	{
		URL:                "https://charts.rancher.cn/2.10-prime/latest/rancher-2.10.1-ent.tgz",
		Name:               "",
		Version:            "",
		InsecureSkipVerify: false,
		SystemContext:      utils.CopySystemContext(nil),
		Policy:             policy,
	},
	// OCI repository
	{
		URL:                "oci://ghcr.io/nginx/charts",
		Name:               "nginx-ingress",
		Version:            "2.0.1",
		InsecureSkipVerify: false,
		SystemContext:      utils.CopySystemContext(nil),
		Policy:             policy,
	},
	// Directory (skip if not exists)
	{
		URL:                "./test/charts",
		Name:               "rancher-eks-operator",
		Version:            "",
		InsecureSkipVerify: false,
		SystemContext:      utils.CopySystemContext(nil),
		Policy:             policy,
	},
}

func Test_Chart(t *testing.T) {
	for _, o := range opts {
		if strings.HasPrefix(o.URL, "./") {
			if _, err := os.Stat(o.URL); err != nil {
				t.Logf("skip test %q", o.URL)
				continue
			}
		}
		c := NewChart(o)
		defer c.Cleanup()
		if err := c.Fetch(context.TODO()); err != nil {
			t.Errorf("faied to fetch %q: %v", c.url, err)
			return
		}
		t.Logf("chart %q cache dir: %q\n", c.url, c.cacheDir)

		img, err := c.image()
		if err != nil {
			t.Error(err)
			return
		}
		t.Logf("%q:\n%v\n", c.cacheDir, utils.ToJSON(img))
		t.Logf("------------------------------\n")
	}
}
