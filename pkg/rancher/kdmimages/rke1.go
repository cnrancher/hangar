package kdmimages

import (
	"context"
	"fmt"
	"strings"

	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/rancher/rke/types"
	"github.com/sirupsen/logrus"
)

// rkeGetter is the object to get RKE images and versions.
type rkeGetter struct {
	rancherVersion    string
	rkeSysImages      map[string]types.RKESystemImages
	linuxSvcOptions   map[string]types.KubernetesServicesOptions
	windowsSvcOptions map[string]types.KubernetesServicesOptions
	rancherVersions   map[string]types.K8sVersionInfo

	linuxInfo   *versionInfo
	windowsInfo *versionInfo

	// map[image][source]bool
	linuxImageSet   map[string]map[string]bool
	windowsImageSet map[string]map[string]bool
	// RKE versions set
	versionSet map[string]bool
}

func newRKEGetter(o *GetterOptions) (*rkeGetter, error) {
	if _, err := utils.EnsureSemverValid(o.RancherVersion); err != nil {
		return nil, err
	}

	return &rkeGetter{
		rancherVersion:    o.RancherVersion,
		rkeSysImages:      o.KDMData.K8sVersionRKESystemImages,
		linuxSvcOptions:   o.KDMData.K8sVersionServiceOptions,
		windowsSvcOptions: o.KDMData.K8sVersionWindowsServiceOptions,
		rancherVersions:   o.KDMData.K8sVersionInfo,
	}, nil
}

func (g *rkeGetter) Get(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if g.linuxImageSet == nil {
		g.linuxImageSet = make(map[string]map[string]bool)
	}
	if g.windowsImageSet == nil {
		g.windowsImageSet = make(map[string]map[string]bool)
	}
	if g.versionSet == nil {
		g.versionSet = make(map[string]bool)
	}

	logrus.Infof("Generating RKE images.")

	// RKE1 images already removed the deprecated k8s patch release so the
	// ignoreDeprecated option is not needed here.
	if err := g.getK8sVersionInfo(); err != nil {
		return err
	}
	if err := fetchImages(g.linuxInfo, g.linuxImageSet, "linux"); err != nil {
		return err
	}
	if err := fetchImages(g.windowsInfo, g.windowsImageSet, "windows"); err != nil {
		return err
	}
	// Remove images begins with noiro
	for image := range g.linuxImageSet {
		if discardImage(image) {
			logrus.Debugf("Discard %q rke system image", image)
			delete(g.linuxImageSet, image)
		}
	}
	for image := range g.windowsImageSet {
		if discardImage(image) {
			logrus.Debugf("Discard %q rke system image", image)
			delete(g.windowsImageSet, image)
		}
	}
	return nil
}

func discardImage(image string) bool {
	project := utils.GetProjectName(image)
	switch project {
	case "rancher", "cnrancher", "library":
		return false
	}
	return true
}

func fetchImages(
	versionInfo *versionInfo,
	imageSet map[string]map[string]bool,
	os string,
) error {
	if versionInfo == nil || len(versionInfo.RKESystemImages) <= 0 {
		return nil
	}
	collectionImagesList := []any{versionInfo.RKESystemImages}
	images, err := flatImagesFromCollections(collectionImagesList...)
	if err != nil {
		return fmt.Errorf("fetchImages: %w", err)
	}
	for _, image := range images {
		if imageSet[image] == nil {
			imageSet[image] = make(map[string]bool)
		}
		imageSet[image]["rke-system-"+os] = true
	}
	return nil
}

func flatImagesFromCollections(
	cols ...any,
) (images []string, err error) {
	for _, col := range cols {
		colObj := map[string]any{}
		if err := utils.ToObj(col, &colObj); err != nil {
			return []string{}, err
		}
		images = append(images, fetchImagesFromCollection(colObj)...)
	}
	return images, nil
}

func fetchImagesFromCollection(obj map[string]any) (images []string) {
	for _, v := range obj {
		switch t := v.(type) {
		case string:
			images = append(images, t)
		case map[string]any:
			images = append(images, fetchImagesFromCollection(t)...)
		}
	}
	return images
}

func (g *rkeGetter) getK8sVersionInfo() error {
	linuxInfo := newVersionInfo()
	windowsInfo := newVersionInfo()
	g.linuxInfo = linuxInfo
	g.windowsInfo = windowsInfo

	maxVersionForMajorK8sVersion := map[string]string{}
	for k8sVersion := range g.rkeSysImages {
		rancherVersionInfo, ok := g.rancherVersions[k8sVersion]
		if ok && toIgnoreForAllK8s(rancherVersionInfo, g.rancherVersion) {
			continue
		}
		majorVersion := getTagMajorVersion(k8sVersion)
		majorVersionInfo, ok := g.rancherVersions[majorVersion]
		if ok && toIgnoreForK8sCurrent(majorVersionInfo, g.rancherVersion) {
			continue
		}
		curr, ok := maxVersionForMajorK8sVersion[majorVersion]
		res, err := utils.SemverCompare(k8sVersion, curr)
		if err != nil || !ok || res > 0 {
			maxVersionForMajorK8sVersion[majorVersion] = k8sVersion
		}
	}
	for majorVersion, k8sVersion := range maxVersionForMajorK8sVersion {
		sysImgs, exist := g.rkeSysImages[k8sVersion]
		if !exist {
			continue
		}
		// windows has been supported since v1.14,
		// the following logic would not find `< v1.14` service options
		if svcOptions, exist := g.windowsSvcOptions[majorVersion]; exist {
			// only keep the related images for windows
			windowsSysImgs := types.RKESystemImages{
				NginxProxy:                sysImgs.NginxProxy,
				CertDownloader:            sysImgs.CertDownloader,
				KubernetesServicesSidecar: sysImgs.KubernetesServicesSidecar,
				Kubernetes:                sysImgs.Kubernetes,
				WindowsPodInfraContainer:  sysImgs.WindowsPodInfraContainer,
			}

			windowsInfo.RKESystemImages[k8sVersion] = windowsSysImgs
			windowsInfo.KubernetesServicesOptions[k8sVersion] = svcOptions
			g.versionSet[k8sVersion] = true
		}
		if svcOptions, exist := g.linuxSvcOptions[majorVersion]; exist {
			// clean the unrelated images for linux
			sysImgs.WindowsPodInfraContainer = ""

			linuxInfo.RKESystemImages[k8sVersion] = sysImgs
			linuxInfo.KubernetesServicesOptions[k8sVersion] = svcOptions
			g.versionSet[k8sVersion] = true
		}
	}

	return nil
}

func getTagMajorVersion(tag string) string {
	splitTag := strings.Split(tag, ".")
	if len(splitTag) < 2 {
		return ""
	}
	return strings.Join(splitTag[:2], ".")
}

type versionInfo struct {
	RKESystemImages           map[string]types.RKESystemImages
	KubernetesServicesOptions map[string]types.KubernetesServicesOptions
}

func newVersionInfo() *versionInfo {
	return &versionInfo{
		RKESystemImages:           map[string]types.RKESystemImages{},
		KubernetesServicesOptions: map[string]types.KubernetesServicesOptions{},
	}
}

func toIgnoreForAllK8s(
	rancherVersionInfo types.K8sVersionInfo,
	rancherVersion string,
) bool {
	if rancherVersionInfo.DeprecateRancherVersion != "" {
		res, err := utils.SemverCompare(
			rancherVersion, rancherVersionInfo.DeprecateRancherVersion)
		if err != nil {
			logrus.Warn(err)
		} else if res >= 0 {
			return true
		}
	}
	if rancherVersionInfo.MinRancherVersion != "" {
		res, err := utils.SemverCompare(
			rancherVersion, rancherVersionInfo.MinRancherVersion)
		if err != nil {
			logrus.Warn(err)
		} else if res < 0 {
			// only respect min versions, even if max is present
			// we need to support upgraded clusters
			return true
		}
	}
	return false
}

func toIgnoreForK8sCurrent(
	majorVersionInfo types.K8sVersionInfo,
	rancherVersion string,
) bool {
	if majorVersionInfo.MaxRancherVersion != "" {
		res, err := utils.SemverCompare(
			rancherVersion, majorVersionInfo.MaxRancherVersion)
		if err != nil {
			logrus.Warn(err)
		} else if res > 0 {
			// include in K8sVersionCurrent only if less then max version
			return true
		}
	}
	return false
}

func (g *rkeGetter) LinuxImageSet() map[string]map[string]bool {
	return g.linuxImageSet
}

func (g *rkeGetter) WindowsImageSet() map[string]map[string]bool {
	return g.windowsImageSet
}

func (g *rkeGetter) VersionSet() map[string]bool {
	return g.versionSet
}

func (g *rkeGetter) Source() ClusterType {
	return RKE
}
