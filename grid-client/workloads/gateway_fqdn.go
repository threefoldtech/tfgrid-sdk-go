// Package workloads includes workloads types (vm, zdb, QSFS, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

var fqdnRegex = regexp.MustCompile(`^([a-zA-Z0-9-_]+\.)+[a-zA-Z0-9-_]{2,}$`)

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

// Validate validates gateway data
func (g *GatewayFQDNProxy) Validate() error {
	if err := validateName(g.Name); err != nil {
		return errors.Wrap(err, "gateway name is invalid")
	}

	if g.NodeID == 0 {
		return fmt.Errorf("node ID should be a positive integer not zero")
	}

	if len(strings.TrimSpace(g.Network)) != 0 {
		if err := validateName(g.Network); err != nil {
			return errors.Wrap(err, "gateway network is invalid")
		}
	}

	if !fqdnRegex.MatchString(g.FQDN) {
		return fmt.Errorf("fqdn %s is invalid", g.FQDN)
	}

	return validateBackend(g.Backends, g.TLSPassthrough)
}

func validateBackend(backends []zos.Backend, tlsPassthrough bool) error {
	if len(backends) == 0 {
		return fmt.Errorf("backends list can not be empty")
	}

	if len(backends) != 1 {
		return fmt.Errorf("only one backend is supported")
	}

	for _, backend := range backends {
		if err := backend.Valid(tlsPassthrough); err != nil {
			return errors.Wrapf(err, "failed to validate backend '%s'", backend)
		}
	}

	return nil
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
		g.SolutionType = g.Name
	}

	deploymentData := DeploymentData{
		Version:     int(Version3),
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
