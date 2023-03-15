package kdm

type KubernetesServicesOptions struct {
	// Additional options passed to Etcd
	Etcd map[string]string `json:"etcd"`
	// Additional options passed to KubeAPI
	KubeAPI map[string]string `json:"kubeapi"`
	// Additional options passed to Kubelet
	Kubelet map[string]string `json:"kubelet"`
	// Additional options passed to Kubeproxy
	Kubeproxy map[string]string `json:"kubeproxy"`
	// Additional options passed to KubeController
	KubeController map[string]string `json:"kubeController"`
	// Additional options passed to Scheduler
	Scheduler map[string]string `json:"scheduler"`
}

type RKESystemImages struct {
	// etcd image
	Etcd string `yaml:"etcd" json:"etcd,omitempty"`
	// Alpine image
	Alpine string `yaml:"alpine" json:"alpine,omitempty"`
	// rke-nginx-proxy image
	NginxProxy string `yaml:"nginx_proxy" json:"nginxProxy,omitempty"`
	// rke-cert-deployer image
	CertDownloader string `yaml:"cert_downloader" json:"certDownloader,omitempty"`
	// rke-service-sidekick image
	KubernetesServicesSidecar string `yaml:"kubernetes_services_sidecar" json:"kubernetesServicesSidecar,omitempty"`
	// KubeDNS image
	KubeDNS string `yaml:"kubedns" json:"kubedns,omitempty"`
	// DNSMasq image
	DNSmasq string `yaml:"dnsmasq" json:"dnsmasq,omitempty"`
	// KubeDNS side car image
	KubeDNSSidecar string `yaml:"kubedns_sidecar" json:"kubednsSidecar,omitempty"`
	// KubeDNS autoscaler image
	KubeDNSAutoscaler string `yaml:"kubedns_autoscaler" json:"kubednsAutoscaler,omitempty"`
	// CoreDNS image
	CoreDNS string `yaml:"coredns" json:"coredns,omitempty"`
	// CoreDNS autoscaler image
	CoreDNSAutoscaler string `yaml:"coredns_autoscaler" json:"corednsAutoscaler,omitempty"`
	// Nodelocal image
	Nodelocal string `yaml:"nodelocal" json:"nodelocal,omitempty"`
	// Kubernetes image
	Kubernetes string `yaml:"kubernetes" json:"kubernetes,omitempty"`
	// Flannel image
	Flannel string `yaml:"flannel" json:"flannel,omitempty"`
	// Flannel CNI image
	FlannelCNI string `yaml:"flannel_cni" json:"flannelCni,omitempty"`
	// Calico Node image
	CalicoNode string `yaml:"calico_node" json:"calicoNode,omitempty"`
	// Calico CNI image
	CalicoCNI string `yaml:"calico_cni" json:"calicoCni,omitempty"`
	// Calico Controllers image
	CalicoControllers string `yaml:"calico_controllers" json:"calicoControllers,omitempty"`
	// Calicoctl image
	CalicoCtl string `yaml:"calico_ctl" json:"calicoCtl,omitempty"`
	//CalicoFlexVol image
	CalicoFlexVol string `yaml:"calico_flexvol" json:"calicoFlexVol,omitempty"`
	// Canal Node Image
	CanalNode string `yaml:"canal_node" json:"canalNode,omitempty"`
	// Canal CNI image
	CanalCNI string `yaml:"canal_cni" json:"canalCni,omitempty"`
	// Canal Controllers Image needed for Calico/Canal v3.14.0+
	CanalControllers string `yaml:"canal_controllers" json:"canalControllers,omitempty"`
	//CanalFlannel image
	CanalFlannel string `yaml:"canal_flannel" json:"canalFlannel,omitempty"`
	//CanalFlexVol image
	CanalFlexVol string `yaml:"canal_flexvol" json:"canalFlexVol,omitempty"`
	//Weave Node image
	WeaveNode string `yaml:"weave_node" json:"weaveNode,omitempty"`
	// Weave CNI image
	WeaveCNI string `yaml:"weave_cni" json:"weaveCni,omitempty"`
	// Pod infra container image
	PodInfraContainer string `yaml:"pod_infra_container" json:"podInfraContainer,omitempty"`
	// Ingress Controller image
	Ingress string `yaml:"ingress" json:"ingress,omitempty"`
	// Ingress Controller Backend image
	IngressBackend string `yaml:"ingress_backend" json:"ingressBackend,omitempty"`
	// Ingress Webhook image
	IngressWebhook string `yaml:"ingress_webhook" json:"ingressWebhook,omitempty"`
	// Metrics Server image
	MetricsServer string `yaml:"metrics_server" json:"metricsServer,omitempty"`
	// Pod infra container image for Windows
	WindowsPodInfraContainer string `yaml:"windows_pod_infra_container" json:"windowsPodInfraContainer,omitempty"`
	// Cni deployer container image for Cisco ACI
	AciCniDeployContainer string `yaml:"aci_cni_deploy_container" json:"aciCniDeployContainer,omitempty"`
	// host container image for Cisco ACI
	AciHostContainer string `yaml:"aci_host_container" json:"aciHostContainer,omitempty"`
	// opflex agent container image for Cisco ACI
	AciOpflexContainer string `yaml:"aci_opflex_container" json:"aciOpflexContainer,omitempty"`
	// mcast daemon container image for Cisco ACI
	AciMcastContainer string `yaml:"aci_mcast_container" json:"aciMcastContainer,omitempty"`
	// OpenvSwitch container image for Cisco ACI
	AciOpenvSwitchContainer string `yaml:"aci_ovs_container" json:"aciOvsContainer,omitempty"`
	// Controller container image for Cisco ACI
	AciControllerContainer string `yaml:"aci_controller_container" json:"aciControllerContainer,omitempty"`
	// GBP Server container image for Cisco ACI
	AciGbpServerContainer string `yaml:"aci_gbp_server_container" json:"aciGbpServerContainer,omitempty"`
	// Opflex Server container image for Cisco ACI
	AciOpflexServerContainer string `yaml:"aci_opflex_server_container" json:"aciOpflexServerContainer,omitempty"`
}

type K8sVersionInfo struct {
	MinRKEVersion       string `yaml:"min_rke_version" json:"minRKEVersion,omitempty"`
	MaxRKEVersion       string `yaml:"max_rke_version" json:"maxRKEVersion,omitempty"`
	DeprecateRKEVersion string `yaml:"deprecate_rke_version" json:"deprecateRKEVersion,omitempty"`

	MinRancherVersion       string `yaml:"min_rancher_version" json:"minRancherVersion,omitempty"`
	MaxRancherVersion       string `yaml:"max_rancher_version" json:"maxRancherVersion,omitempty"`
	DeprecateRancherVersion string `yaml:"deprecate_rancher_version" json:"deprecateRancherVersion,omitempty"`
}

type CisBenchmarkVersionInfo struct {
	Managed              bool              `yaml:"managed" json:"managed"`
	MinKubernetesVersion string            `yaml:"min_kubernetes_version" json:"minKubernetesVersion"`
	SkippedChecks        map[string]string `yaml:"skipped_checks" json:"skippedChecks"`
	NotApplicableChecks  map[string]string `yaml:"not_applicable_checks" json:"notApplicableChecks"`
}

type CisConfigParams struct {
	BenchmarkVersion string `yaml:"benchmark_version" json:"benchmarkVersion"`
}
