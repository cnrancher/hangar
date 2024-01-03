package chartimages

// Some latest tag chart images does not exists.
var IgnoreChartImages = map[string]bool{
	"rancher/mirrored-sig-storage-csi-attacher:latest":                   true,
	"rancher/mirrored-sig-storage-csi-node-driver-registrar:latest":      true,
	"rancher/mirrored-sig-storage-csi-provisioner:latest":                true,
	"rancher/mirrored-sig-storage-csi-resizer:latest":                    true,
	"rancher/mirrored-sig-storage-livenessprobe:latest":                  true,
	"rancher/mirrored-cloud-provider-vsphere-csi-release-syncer:latest":  true,
	"rancher/mirrored-cloud-provider-vsphere-csi-release-driver:latest":  true,
	"rancher/mirrored-cloud-provider-vsphere-cpi-release-manager:latest": true,
}
