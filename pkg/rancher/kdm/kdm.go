// Types defined in this package were copied from
// https://github.com/rancher/rke/tree/release/v1.4/types
package kdm

import "encoding/json"

type Data struct {
	// K8sVersionServiceOptions - service options per k8s version
	K8sVersionServiceOptions  map[string]KubernetesServicesOptions
	K8sVersionRKESystemImages map[string]RKESystemImages

	// Addon Templates per K8s version ("default" where nothing changes for k8s version)
	K8sVersionedTemplates map[string]map[string]string

	// K8sVersionInfo - min/max RKE+Rancher versions per k8s version
	K8sVersionInfo map[string]K8sVersionInfo

	//Default K8s version for every rancher version
	RancherDefaultK8sVersions map[string]string

	//Default K8s version for every rke version
	RKEDefaultK8sVersions map[string]string

	K8sVersionDockerInfo map[string][]string

	// K8sVersionWindowsServiceOptions - service options per windows k8s version
	K8sVersionWindowsServiceOptions map[string]KubernetesServicesOptions

	CisConfigParams         map[string]CisConfigParams
	CisBenchmarkVersionInfo map[string]CisBenchmarkVersionInfo

	// K3S specific data, opaque and defined by the config file in kdm
	K3S map[string]interface{} `json:"k3s,omitempty"`
	// Rke2 specific data, defined by the config file in kdm
	RKE2 map[string]interface{} `json:"rke2,omitempty"`
}

func FromData(b []byte) (Data, error) {
	d := &Data{}

	if err := json.Unmarshal(b, d); err != nil {
		return Data{}, err
	}
	return *d, nil
}
