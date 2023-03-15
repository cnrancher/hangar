module github.com/cnrancher/hangar

go 1.19

require (
	github.com/Masterminds/semver/v3 v3.2.0
	github.com/antonfisher/nested-logrus-formatter v1.3.1
	github.com/containers/image/v5 v5.23.1
	github.com/go-git/go-git/v5 v5.5.1
	github.com/klauspost/compress v1.15.13
	github.com/klauspost/pgzip v1.2.5
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.9.0
	github.com/spf13/cobra v1.6.1
	github.com/spf13/pflag v1.0.5
	golang.org/x/exp v0.0.0-20221126150942-6ab00d035af9
	golang.org/x/mod v0.6.0
	golang.org/x/term v0.5.0
	gopkg.in/yaml.v2 v2.4.0
)

replace (
	github.com/containerd/containerd => github.com/containerd/containerd v1.6.18
	golang.org/x/net => golang.org/x/net v0.7.0
)

require (
	github.com/Microsoft/go-winio v0.5.2 // indirect
	github.com/ProtonMail/go-crypto v0.0.0-20221026131551-cf6655e29de4 // indirect
	github.com/acomagu/bufpipe v1.0.3 // indirect
	github.com/cloudflare/circl v1.1.0 // indirect
	github.com/containers/libtrust v0.0.0-20200511145503-9c3a6c22cd9a // indirect
	github.com/containers/ocicrypt v1.1.5 // indirect
	github.com/docker/docker v20.10.22+incompatible // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/go-git/gcfg v1.5.0 // indirect
	github.com/go-git/go-billy/v5 v5.4.0 // indirect
	github.com/imdario/mergo v0.3.13 // indirect
	github.com/inconshreveable/mousetrap v1.0.1 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0-rc2 // indirect
	github.com/pjbgf/sha1cd v0.2.3 // indirect
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/skeema/knownhosts v1.1.0 // indirect
	github.com/stretchr/testify v1.8.1 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	golang.org/x/crypto v0.5.0 // indirect
	golang.org/x/net v0.5.0 // indirect
	golang.org/x/sys v0.5.0 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
)
