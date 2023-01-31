package extraimages

// ExtraImagesLinux are the images hard-coded and cannot read from KDM/Chart.
// Keep this map data updated when new version released.
var ExtraImagesLinux = map[string]map[string]map[string]bool{
	"2.5.0": {},
	"2.6.0": {}, // TODO:
	"2.7.0": {
		// pkg/apis/management.cattle.io/v3/tools_system_images.go
		"rancher/kube-api-auth:v0.1.8": {"system": true},
		// pkg/image/resolve.go
		"rancher/shell:v0.1.18": {"core": true},
		// pkg/image/resolve.go
		"rancher/machine:v0.15.0-rancher95": {"core": true},
		// pkg/image/resolve.go
		"rancher/mirrored-bci-busybox:15.4.11.2": {"core": true},
		// pkg/image/resolve.go
		"rancher/mirrored-bci-micro:15.4.14.3": {"core": true},
	},
	// "2.7.1": {
	// 	// pkg/apis/management.cattle.io/v3/tools_system_images.go
	// 	"rancher/kube-api-auth:v0.1.8": {"system": true},
	// 	// pkg/image/resolve.go
	// 	"rancher/shell:v0.1.18": {"core": true},
	// 	// pkg/image/resolve.go
	// 	"rancher/machine:v0.15.0-rancher95": {"core": true},
	// 	// pkg/image/resolve.go
	// 	"rancher/mirrored-bci-busybox:15.4.11.2": {"core": true},
	// 	// pkg/image/resolve.go
	// 	"rancher/mirrored-bci-micro:15.4.14.3": {"core": true},
	// },
}
