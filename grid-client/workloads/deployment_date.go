// Package workloads includes workloads types (vm, zdb, QSFS, public IP, gateway name, gateway fqdn, disk)
package workloads

var (
	// VMType for deployment date of vms
	VMType = "vm"
	// GatewayNameType for deployment date of name gateway
	GatewayNameType = "Gateway Name"
	// GatewayFQDNType for deployment date of fqdn gateway
	GatewayFQDNType = "Gateway Fqdn"
	// K8sType for deployment date of k8s
	K8sType = "kubernetes"
	// NetworkType for deployment date of network
	NetworkType = "network"
)

// DeploymentData for deployments meta data
type DeploymentData struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	ProjectName string `json:"projectName"`
}
