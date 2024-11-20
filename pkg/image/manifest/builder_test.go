package manifest

import (
	"fmt"
	"testing"

	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
)

func Test_Builder_Add(t *testing.T) {
	assert := assert.New(t)
	builder := &Builder{
		name:          "registry.io/library/example:latest",
		reference:     nil,
		images:        nil,
		systemContext: nil,
		retryOpts:     nil,
	}

	a := &Image{
		Size:        0,
		Digest:      digest.Digest("sha256:" + utils.Sha256Sum("abcabc")),
		MediaType:   "",
		Annotations: nil,
		platform: manifestPlatform{
			arch:       "amd64",
			os:         "linux",
			variant:    "",
			osVersion:  "",
			osFeatures: nil,
		},
	}
	builder.Add(a)
	assert.Equal(1, builder.Images())
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("abcabc")), builder.images[0].Digest)

	// Add the image with the same digest & platform
	builder.Add(a)
	assert.Equal(1, builder.Images())
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("abcabc")), builder.images[0].Digest)

	// Add the image with digest changed but platform same
	b := a.DeepCopy()
	b.Digest = digest.Digest("sha256:" + utils.Sha256Sum("defdef"))
	builder.Add(b)
	assert.Equal(1, builder.Images())
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("defdef")), builder.images[0].Digest)

	// Add the image with digest same but platform changed
	// This is for image supports multi platforms and their digest could be same in manifest index
	c := b.DeepCopy()
	c.platform = manifestPlatform{
		arch:       "arm64",
		os:         "linux",
		variant:    "v8",
		osVersion:  "",
		osFeatures: nil,
	}
	builder.Add(c)
	assert.Equal(2, builder.Images())
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("defdef")), builder.images[0].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("defdef")), builder.images[1].Digest)

	// Add the image with digest changed
	c.Digest = digest.Digest("sha256:" + utils.Sha256Sum("123123"))
	builder.Add(c)
	assert.Equal(2, builder.Images())
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("defdef")), builder.images[0].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("123123")), builder.images[1].Digest)

	// Ensure arm64v8 and arm64 are the same platform
	c.platform.variant = ""
	builder.Add(c)
	assert.Equal(2, builder.Images())
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("defdef")), builder.images[0].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("123123")), builder.images[1].Digest)
	assert.Equal("arm64", builder.images[1].platform.arch)
	assert.Equal("", builder.images[1].platform.variant)

	// Add windows image
	d := a.DeepCopy()
	d.Digest = digest.Digest("sha256:" + utils.Sha256Sum("xyzxyz"))
	d.platform = manifestPlatform{
		arch:       "amd64",
		os:         "windows",
		variant:    "",
		osVersion:  "10.abc.def.111",
		osFeatures: nil,
	}
	builder.Add(d)
	assert.Equal(3, builder.Images())
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("defdef")), builder.images[0].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("123123")), builder.images[1].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("xyzxyz")), builder.images[2].Digest)
	assert.Equal("amd64", builder.images[2].platform.arch)
	assert.Equal("windows", builder.images[2].platform.os)
	assert.Equal("10.abc.def.111", builder.images[2].platform.osVersion)

	// Add another windows image with same os & arch but version changed
	d.platform.osVersion = "20.abc.def.222"
	builder.Add(d)
	assert.Equal(4, builder.Images())
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("defdef")), builder.images[0].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("123123")), builder.images[1].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("xyzxyz")), builder.images[2].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("xyzxyz")), builder.images[3].Digest)
	assert.Equal("amd64", builder.images[3].platform.arch)
	assert.Equal("windows", builder.images[3].platform.os)
	assert.Equal("20.abc.def.222", builder.images[3].platform.osVersion)

	// Add another windows image with another arch but platform same
	d.platform.arch = "arm64"
	d.platform.variant = "v8"
	builder.Add(d)
	assert.Equal(5, builder.Images())
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("defdef")), builder.images[0].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("123123")), builder.images[1].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("xyzxyz")), builder.images[2].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("xyzxyz")), builder.images[3].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("xyzxyz")), builder.images[4].Digest)
	assert.Equal("arm64", builder.images[4].platform.arch)
	assert.Equal("v8", builder.images[4].platform.variant)
	assert.Equal("windows", builder.images[4].platform.os)
	assert.Equal("20.abc.def.222", builder.images[4].platform.osVersion)

	// Upgrade windows image (digest changed)
	d.Digest = digest.Digest("sha256:" + utils.Sha256Sum("ijkijk"))
	builder.Add(d)
	assert.Equal(5, builder.Images())
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("defdef")), builder.images[0].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("123123")), builder.images[1].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("xyzxyz")), builder.images[2].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("xyzxyz")), builder.images[3].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("ijkijk")), builder.images[4].Digest)
	assert.Equal("arm64", builder.images[4].platform.arch)
	assert.Equal("v8", builder.images[4].platform.variant)
	assert.Equal("windows", builder.images[4].platform.os)
	assert.Equal("20.abc.def.222", builder.images[4].platform.osVersion)

	// Upgrade linux amd64 image (digest changed)
	a.Digest = digest.Digest("sha256:" + utils.Sha256Sum("lmnlmn"))
	builder.Add(a)
	assert.Equal(5, builder.Images())
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("123123")), builder.images[0].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("xyzxyz")), builder.images[1].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("xyzxyz")), builder.images[2].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("ijkijk")), builder.images[3].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("lmnlmn")), builder.images[4].Digest)
	assert.Equal("arm64", builder.images[3].platform.arch)
	assert.Equal("v8", builder.images[3].platform.variant)
	assert.Equal("windows", builder.images[3].platform.os)
	assert.Equal("20.abc.def.222", builder.images[3].platform.osVersion)

	// Add SLSA provenance for amd64 linux image
	e := a.DeepCopy()
	e.Digest = digest.Digest("sha256:" + utils.Sha256Sum("000000"))
	e.Annotations = map[string]string{
		"vnd.docker.reference.digest": "sha256:" + utils.Sha256Sum("lmnlmn"),
		"vnd.docker.reference.type":   "attestation-manifest",
	}
	e.platform = manifestPlatform{
		arch: "unknown",
		os:   "unknown",
	}
	builder.Add(e)
	assert.Equal(6, builder.Images())
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("123123")), builder.images[0].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("xyzxyz")), builder.images[1].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("xyzxyz")), builder.images[2].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("ijkijk")), builder.images[3].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("lmnlmn")), builder.images[4].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("000000")), builder.images[5].Digest)
	assert.Equal("unknown", builder.images[5].platform.arch)
	assert.Equal("unknown", builder.images[5].platform.os)

	// Add SLSA provenance for arm64 linux image
	f := e.DeepCopy()
	f.Digest = digest.Digest("sha256:" + utils.Sha256Sum("111111"))
	f.Annotations["vnd.docker.reference.digest"] = "sha256:" + utils.Sha256Sum("123123")
	builder.Add(f)
	assert.Equal(7, builder.Images())
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("123123")), builder.images[0].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("xyzxyz")), builder.images[1].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("xyzxyz")), builder.images[2].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("ijkijk")), builder.images[3].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("lmnlmn")), builder.images[4].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("000000")), builder.images[5].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("111111")), builder.images[6].Digest)
	assert.Equal("unknown", builder.images[6].platform.arch)
	assert.Equal("unknown", builder.images[6].platform.os)

	// SLSA Provenance update (digest changed)
	f.Digest = digest.Digest("sha256:" + utils.Sha256Sum("222222"))
	builder.Add(f)
	assert.Equal(7, builder.Images())
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("222222")), builder.images[6].Digest)
	assert.Equal("unknown", builder.images[6].platform.arch)
	assert.Equal("unknown", builder.images[6].platform.os)

	// Add SLSA Provenance with same digest but for different images
	f.Annotations["vnd.docker.reference.digest"] = "sha256:" + utils.Sha256Sum("xyzxyz")
	builder.Add(f)
	assert.Equal(8, builder.Images())
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("222222")), builder.images[6].Digest)
	assert.Equal(digest.Digest("sha256:"+utils.Sha256Sum("222222")), builder.images[7].Digest)
	assert.Equal("sha256:"+utils.Sha256Sum("123123"), builder.images[6].Annotations["vnd.docker.reference.digest"])
	assert.Equal("sha256:"+utils.Sha256Sum("xyzxyz"), builder.images[7].Annotations["vnd.docker.reference.digest"])
	assert.Equal("unknown", builder.images[6].platform.arch)
	assert.Equal("unknown", builder.images[6].platform.os)

	s, _ := builder.String()
	fmt.Printf("%v\n", s)
}

func Test_manifestPlatform_equal(t *testing.T) {
	assert := assert.New(t)
	a := &manifestPlatform{
		arch:       "arm64",
		os:         "linux",
		variant:    "",
		osVersion:  "",
		osFeatures: nil,
	}

	b := &manifestPlatform{
		arch:       "arm64",
		os:         "linux",
		variant:    "v8",
		osVersion:  "",
		osFeatures: nil,
	}
	assert.True(a.equal(b))
}
