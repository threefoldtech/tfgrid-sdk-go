// Package workloads includes workloads types (vm, zdb, QSFS, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// GatewayFQDNProxy for gateway FQDN proxy
type GatewayFQDNProxy struct {
	// required
	NodeID uint32
	// Backends are list of backend ips
	Backends []zos.Backend
	// FQDN deployed on the node
	FQDN string
	// Name is the workload name
	Name string

	// optional
	// Passthrough whether to pass tls traffic or not
	TLSPassthrough bool
	// Network name to join
	Network      string
	Description  string
	SolutionType string

	// computed
	ContractID       uint64
	NodeDeploymentID map[uint32]uint64
}

// NewGatewayFQDNProxyFromZosWorkload generates a gateway FQDN proxy from a zos workload
func NewGatewayFQDNProxyFromZosWorkload(wl gridtypes.Workload) (GatewayFQDNProxy, error) {
	dataI, err := wl.WorkloadData()
	if err != nil {
		return GatewayFQDNProxy{}, errors.Wrap(err, "failed to get workload data")
	}

	data, ok := dataI.(*zos.GatewayFQDNProxy)
	if !ok {
		return GatewayFQDNProxy{}, errors.Errorf("could not create gateway fqdn proxy workload from data %v", dataI)
	}
	network := ""
	if data.Network != nil {
		network = data.Network.String()
	}

	return GatewayFQDNProxy{
		Name:           wl.Name.String(),
		TLSPassthrough: data.TLSPassthrough,
		Backends:       data.Backends,
		FQDN:           data.FQDN,
		Network:        network,
		Description:    wl.Description,
	}, nil
}

// ZosWorkload generates a zos workload from GatewayFQDNProxy
func (g *GatewayFQDNProxy) ZosWorkload() gridtypes.Workload {
	network := (*gridtypes.Name)(&g.Network)
	if g.Network == "" {
		network = nil
	}
	return gridtypes.Workload{
		Version: 0,
		Type:    zos.GatewayFQDNProxyType,
		Name:    gridtypes.Name(g.Name),
		// REVISE: whether description should be set here
		Data: gridtypes.MustMarshal(zos.GatewayFQDNProxy{
			GatewayBase: zos.GatewayBase{
				TLSPassthrough: g.TLSPassthrough,
				Backends:       g.Backends,
				Network:        network,
			},
			FQDN: g.FQDN,
		}),
	}
}

// GenerateMetadata generates gateway deployment metadata
func (g *GatewayFQDNProxy) GenerateMetadata() (string, error) {
	if len(g.SolutionType) == 0 {
		g.SolutionType = "Gateway"
	}

	deploymentData := DeploymentData{
		Version:     Version,
		Name:        g.Name,
		Type:        "Gateway Fqdn",
		ProjectName: g.SolutionType,
	}

	deploymentDataBytes, err := json.Marshal(deploymentData)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse deployment data %v", deploymentData)
	}

	return string(deploymentDataBytes), nil
}

// NewZosBackends generates new zos backends for the given string backends
func NewZosBackends(bks []string) (backends []zos.Backend) {
	for _, b := range bks {
		backends = append(backends, zos.Backend(b))
	}
	return backends
}
