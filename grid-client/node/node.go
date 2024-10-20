// Package client provides a simple RMB interface to work with the node.
//
// # Requirements
//
// 1. A msg bus instance must be running on the node. this client uses RMB (message bus)
// to send messages to nodes, and get the responses.
// 2. A valid ed25519 key pair. this key is used to sign deployments and MUST be the same
// key used to configure the local twin on substrate.
//
// # Simple deployment
//
// create an instance from the default rmb client.
// ```
// cl, err := rmb.Default()
//
//	if err != nil {
//		panic(err)
//	}
//
// ```
// then create an instance of the node client
// ```
// node := client.NewNodeClient(NodeTwinID, cl)
// ```
// define your deployment object
// ```
//
//	dl := gridtypes.Deployment{
//		Version: Version,
//		twinID:  Twin, //LocalTwin,
//		// this contract id must match the one on substrate
//		Workloads: []gridtypes.Workload{
//			network(), // network workload definition
//			zmount(), // zmount workload definition
//			publicip(), // public ip definition
//			zmachine(), // zmachine definition
//		},
//		SignatureRequirement: gridtypes.SignatureRequirement{
//			WeightRequired: 1,
//			Requests: []gridtypes.SignatureRequest{
//				{
//					twinID: Twin,
//					Weight: 1,
//				},
//			},
//		},
//	}
//
// ```
// compute hash
// ```
// hash, err := dl.ChallengeHash()
//
//	if err != nil {
//		panic("failed to create hash")
//	}
//
// ```
// create the contract and ge the contract id
// then
// “
// dl.ContractID = 11 // from substrate
// ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
// defer cancel()
// err = node.DeploymentDeploy(ctx, dl)
//
//	if err != nil {
//		panic(err)
//	}
//
// ```
package client

import (
	"context"
	"math/rand"
	"net"
	"slices"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/subi"
	zosTypes "github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
	"github.com/threefoldtech/zos/pkg/capacity/dmi"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// ErrNoAccessibleInterfaceFound no accessible interface found
var ErrNoAccessibleInterfaceFound = errors.Errorf("could not find a publicly accessible ipv4 or ipv6")

// IfaceType define the different public interface supported
type IfaceType string

// PublicConfig is the configuration of the interface
// that is connected to the public internet
type PublicConfig struct {
	// Type define if we need to use
	// the Vlan field or the MacVlan
	Type IfaceType `json:"type"`
	// Vlan int16     `json:"vlan"`
	// Macvlan net.HardwareAddr

	IPv4 gridtypes.IPNet `json:"ipv4"`
	IPv6 gridtypes.IPNet `json:"ipv6"`

	GW4 net.IP `json:"gw4"`
	GW6 net.IP `json:"gw6"`

	// Domain is the node domain name like gent01.devnet.grid.tf
	// or similar
	Domain string `json:"domain"`
}

// ExitDevice stores the dual nic setup of a node.
type ExitDevice struct {
	// IsSingle is set to true if br-pub
	// is connected to zos bridge
	IsSingle bool `json:"is_single"`
	// IsDual is set to true if br-pub is
	// connected to a physical nic
	IsDual bool `json:"is_dual"`
	// AsDualInterface is set to the physical
	// interface name if IsDual is true
	AsDualInterface string `json:"dual_interface"`
}

// PoolMetrics stores storage pool metrics
type PoolMetrics struct {
	Name string         `json:"name"`
	Type zos.DeviceType `json:"type"`
	Size gridtypes.Unit `json:"size"`
	Used gridtypes.Unit `json:"used"`
}

// Interface stores physical network interface information
type Interface struct {
	IPs []string `json:"ips"`
	Mac string   `json:"mac"`
}

// NodeClient struct
type NodeClient struct {
	nodeTwin uint32
	bus      rmb.Client
	timeout  time.Duration
}

// rmbCmdArgs is a map of command line arguments
type rmbCmdArgs map[string]interface{}

// NewNodeClient creates a new node RMB client. This client then can be used to
// communicate with the node over RMB.
func NewNodeClient(nodeTwin uint32, bus rmb.Client, timeout time.Duration) *NodeClient {
	return &NodeClient{
		nodeTwin: nodeTwin,
		bus:      bus,
		timeout:  timeout,
	}
}

// SystemGetNodeFeatures gets the supported nodes features.
func (n *NodeClient) SystemGetNodeFeatures(ctx context.Context) (feat []string, err error) {
	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	const cmd = "zos.system.node_features_get"

	err = n.bus.Call(ctx, n.nodeTwin, cmd, nil, &feat)
	return
}

// DeploymentDeploy sends the deployment to the node for processing.
func (n *NodeClient) DeploymentDeploy(ctx context.Context, dl zosTypes.Deployment) error {
	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	const cmd = "zos.deployment.deploy"
	return n.bus.Call(ctx, n.nodeTwin, cmd, dl, nil)
}

// DeploymentUpdate update the given deployment. deployment must be a valid update for
// a deployment that has been already created via DeploymentDeploy
func (n *NodeClient) DeploymentUpdate(ctx context.Context, dl zosTypes.Deployment) error {
	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	const cmd = "zos.deployment.update"
	return n.bus.Call(ctx, n.nodeTwin, cmd, dl, nil)
}

// DeploymentGet gets a deployment via contract ID
func (n *NodeClient) DeploymentGet(ctx context.Context, contractID uint64) (dl zosTypes.Deployment, err error) {
	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	const cmd = "zos.deployment.get"
	in := rmbCmdArgs{
		"contract_id": contractID,
	}

	if err = n.bus.Call(ctx, n.nodeTwin, cmd, in, &dl); err != nil {
		return dl, err
	}

	return dl, nil
}

// DeploymentDelete deletes a deployment, the node will make sure to decomission all deployments
// and set all workloads to deleted. A call to Get after delete is valid
func (n *NodeClient) DeploymentDelete(ctx context.Context, contractID uint64) error {
	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	const cmd = "zos.deployment.delete"
	in := rmbCmdArgs{
		"contract_id": contractID,
	}

	return n.bus.Call(ctx, n.nodeTwin, cmd, in, nil)
}

// DeploymentList gets all deployments for a twin
func (n *NodeClient) DeploymentList(ctx context.Context) (dls []zosTypes.Deployment, err error) {
	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	const cmd = "zos.deployment.list"

	err = n.bus.Call(ctx, n.nodeTwin, cmd, nil, &dls)
	return
}

// Statistics returns some node statistics. Including total and available cpu, memory, storage, etc...
func (n *NodeClient) Statistics(ctx context.Context) (total gridtypes.Capacity, used gridtypes.Capacity, err error) {
	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	const cmd = "zos.statistics.get"
	result := struct {
		// Total system capacity
		Total gridtypes.Capacity `json:"total"`
		// Used capacity this include user + system resources
		Used gridtypes.Capacity `json:"used"`
		// System resource reserved by zos
		System gridtypes.Capacity `json:"system"`
		// Users statistics by zos
		Users struct {
			// Total deployments count
			Deployments int `json:"deployments"`
			// Total workloads count
			Workloads int `json:"workloads"`
		} `json:"users"`
	}{}

	if err = n.bus.Call(ctx, n.nodeTwin, cmd, nil, &result); err != nil {
		return
	}

	return result.Total, result.Used, nil
}

// NetworkListPrivateIPs list private ips reserved for a network
func (n *NodeClient) NetworkListPrivateIPs(ctx context.Context, networkName string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	const cmd = "zos.network.list_private_ips"
	var result []string
	in := rmbCmdArgs{
		"network_name": networkName,
	}

	if err := n.bus.Call(ctx, n.nodeTwin, cmd, in, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// NetworkListWGPorts return a list of all "taken" ports on the node. A new deployment
// should be careful to use a free port for its network setup.
func (n *NodeClient) NetworkListWGPorts(ctx context.Context) ([]uint16, error) {
	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	const cmd = "zos.network.list_wg_ports"
	var result []uint16

	if err := n.bus.Call(ctx, n.nodeTwin, cmd, nil, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// NetworkListInterfaces return a map of all interfaces and their ips
func (n *NodeClient) NetworkListInterfaces(ctx context.Context) (map[string][]net.IP, error) {
	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	const cmd = "zos.network.interfaces"
	var result map[string][]net.IP

	if err := n.bus.Call(ctx, n.nodeTwin, cmd, nil, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// DeploymentChanges return changes of a deployment via contract ID
func (n *NodeClient) DeploymentChanges(ctx context.Context, contractID uint64) (changes []zosTypes.Workload, err error) {
	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	const cmd = "zos.deployment.changes"
	in := rmbCmdArgs{
		"contract_id": contractID,
	}

	if err = n.bus.Call(ctx, n.nodeTwin, cmd, in, &changes); err != nil {
		return changes, err
	}

	return changes, nil
}

// NetworkListIPs list taken public IPs on the node
func (n *NodeClient) NetworkListIPs(ctx context.Context) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	const cmd = "zos.network.list_public_ips"
	var result []string

	if err := n.bus.Call(ctx, n.nodeTwin, cmd, nil, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// NetworkGetPublicConfig returns the current public node network configuration. A node with a
// public config can be used as an access node for wireguard.
func (n *NodeClient) NetworkGetPublicConfig(ctx context.Context) (cfg PublicConfig, err error) {
	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	const cmd = "zos.network.public_config_get"

	if err = n.bus.Call(ctx, n.nodeTwin, cmd, nil, &cfg); err != nil {
		return
	}

	return
}

// NetworkSetPublicConfig sets the current public node network configuration. A node with a
// public config can be used as an access node for wireguard.
func (n *NodeClient) NetworkSetPublicConfig(ctx context.Context, cfg PublicConfig) error {
	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	const cmd = "zos.network.public_config_set"
	return n.bus.Call(ctx, n.nodeTwin, cmd, cfg, nil)
}

// SystemDMI executes dmidecode to get dmidecode output
func (n *NodeClient) SystemDMI(ctx context.Context) (result dmi.DMI, err error) {
	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	const cmd = "zos.system.dmi"

	if err = n.bus.Call(ctx, n.nodeTwin, cmd, nil, &result); err != nil {
		return
	}

	return
}

// SystemHypervisor executes hypervisor cmd
func (n *NodeClient) SystemHypervisor(ctx context.Context) (result string, err error) {
	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	const cmd = "zos.system.hypervisor"

	if err = n.bus.Call(ctx, n.nodeTwin, cmd, nil, &result); err != nil {
		return
	}

	return
}

// Version is ZOS version
type Version struct {
	ZOS   string `json:"zos"`
	ZInit string `json:"zinit"`
}

// SystemVersion executes system version cmd
func (n *NodeClient) SystemVersion(ctx context.Context) (ver Version, err error) {
	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	const cmd = "zos.system.version"

	if err = n.bus.Call(ctx, n.nodeTwin, cmd, nil, &ver); err != nil {
		return
	}

	return
}

// TaskResult holds the perf test result
type TaskResult struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Timestamp   uint64      `json:"timestamp"`
	Result      interface{} `json:"result"`
}

// GetPerfTestsResults get all perf tests results
func (n *NodeClient) GetPerfTestResults(ctx context.Context) (result []TaskResult, err error) {
	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	const cmd = "zos.perf.get_all"
	err = n.bus.Call(ctx, n.nodeTwin, cmd, nil, &result)

	return
}

// GetPerfTestResult get a single perf test result
func (n *NodeClient) GetPerfTestResult(ctx context.Context, testName string) (result TaskResult, err error) {
	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	payload := struct {
		Name string
	}{
		Name: testName,
	}

	const cmd = "zos.perf.get"
	err = n.bus.Call(ctx, n.nodeTwin, cmd, payload, &result)

	return
}

// IsNodeUp checks if the node is up
func (n *NodeClient) IsNodeUp(ctx context.Context) error {
	_, err := n.SystemVersion(ctx)
	return err
}

// AreNodesUp checks if nodes are up
func AreNodesUp(ctx context.Context, sub subi.SubstrateExt, nodes []uint32, nc NodeClientGetter) error {
	for _, node := range nodes {
		cl, err := nc.GetNodeClient(sub, node)
		if err != nil {
			return errors.Wrapf(err, "could not get node %d client", node)
		}
		if err := cl.IsNodeUp(ctx); err != nil {
			return errors.Wrapf(err, "could not reach node %d", node)
		}
	}
	return nil
}

// GetNodeFreeWGPort returns node free wireguard port
func (n *NodeClient) GetNodeFreeWGPort(ctx context.Context, nodeID uint32, usedPorts []uint16) (int, error) {
	nodeUsedPorts, err := n.NetworkListWGPorts(ctx)
	if err != nil {
		return 0, errors.Wrap(err, "failed to list wg ports")
	}
	log.Debug().Msgf("reserved ports for node %d: %v", nodeID, nodeUsedPorts)
	// from 1024 to 32767 (the lower limit for ephemeral ports)
	p := uint(rand.Intn(32768-1024) + 1024)

	for contains(nodeUsedPorts, uint16(p)) || slices.Contains(usedPorts, uint16(p)) {
		p = uint(rand.Intn(32768-1024) + 1024)
	}
	log.Debug().Msgf("Selected port for node %d is %d", nodeID, p)
	return int(p), nil
}

// GetNodeEndpoint gets node end point network ip
func (n *NodeClient) GetNodeEndpoint(ctx context.Context) (net.IP, error) {
	publicConfig, err := n.NetworkGetPublicConfig(ctx)
	if err == nil && publicConfig.IPv4.IP != nil {

		ip := publicConfig.IPv4.IP
		log.Debug().Msgf("ip: %s, global unicast: %t, privateIP: %t", ip.String(), ip.IsGlobalUnicast(), ip.IsPrivate())
		if ip.IsGlobalUnicast() && !ip.IsPrivate() {
			return ip, nil
		}
	} else if err == nil && publicConfig.IPv6.IP != nil {
		ip := publicConfig.IPv6.IP
		log.Debug().Msgf("ip: %s, global unicast: %t, privateIP: %t", ip.String(), ip.IsGlobalUnicast(), ip.IsPrivate())
		if ip.IsGlobalUnicast() && !ip.IsPrivate() {
			return ip, nil
		}
	}

	ifs, err := n.NetworkListInterfaces(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not list node interfaces")
	}
	log.Debug().Msgf("interface: %v", ifs)

	zosIf, ok := ifs["zos"]
	if !ok {
		return nil, errors.Wrap(ErrNoAccessibleInterfaceFound, "no zos interface")
	}
	for _, ip := range zosIf {
		log.Debug().Msgf("ip: %s, global unicast: %t, privateIP: %t", ip.String(), ip.IsGlobalUnicast(), ip.IsPrivate())
		if !ip.IsGlobalUnicast() || ip.IsPrivate() {
			continue
		}

		return ip, nil
	}
	return nil, errors.Wrap(ErrNoAccessibleInterfaceFound, "no public ipv4 or ipv6 on zos interface found")
}

// Pools returns statistics of separate pools
func (n *NodeClient) Pools(ctx context.Context) (pools []PoolMetrics, err error) {
	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	const cmd = "zos.storage.pools"
	err = n.bus.Call(ctx, n.nodeTwin, cmd, nil, &pools)
	return
}

type GPU struct {
	ID       string `json:"id"`
	Vendor   string `json:"vendor"`
	Device   string `json:"device"`
	Contract uint64 `json:"contract"`
}

// GPUs returns a list of gpus
func (n *NodeClient) GPUs(ctx context.Context) (gpus []GPU, err error) {
	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	const cmd = "zos.gpu.list"
	err = n.bus.Call(ctx, n.nodeTwin, cmd, nil, &gpus)
	return
}

// HasPublicIPv6 returns true if the node has a public ip6 configuration
func (n *NodeClient) HasPublicIPv6(ctx context.Context) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	const cmd = "zos.network.has_ipv6"
	var result bool

	if err := n.bus.Call(ctx, n.nodeTwin, cmd, nil, &result); err != nil {
		return false, err
	}

	return result, nil
}

// NetworkListAllInterfaces return all physical devices on a node
func (n *NodeClient) NetworkListAllInterfaces(ctx context.Context) (result map[string]Interface, err error) {
	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	const cmd = "zos.network.admin.interfaces"
	err = n.bus.Call(ctx, n.nodeTwin, cmd, nil, &result)

	return
}

// NetworkSetPublicExitDevice select which physical interface to use as an exit device
// setting `iface` to `zos` will then make node run in a single nic setup.
func (n *NodeClient) NetworkSetPublicExitDevice(ctx context.Context, iface string) error {
	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	const cmd = "zos.network.admin.set_public_nic"
	return n.bus.Call(ctx, n.nodeTwin, cmd, iface, nil)
}

// NetworkGetPublicExitDevice gets the current dual nic setup of the node.
func (n *NodeClient) NetworkGetPublicExitDevice(ctx context.Context) (exit ExitDevice, err error) {
	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	const cmd = "zos.network.admin.get_public_nic"
	err = n.bus.Call(ctx, n.nodeTwin, cmd, nil, &exit)
	return
}

func contains[T comparable](elements []T, element T) bool {
	for _, e := range elements {
		if element == e {
			return true
		}
	}
	return false
}
