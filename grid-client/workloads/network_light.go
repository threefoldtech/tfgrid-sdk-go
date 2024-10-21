// Package workloads includes workloads types (vm, zdb, QSFS, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"slices"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	client "github.com/threefoldtech/tfgrid-sdk-go/grid-client/node"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/subi"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// ZNetLight is zos network light workload
type ZNetLight struct {
	Name         string
	Description  string
	SolutionType string
	Nodes        []uint32
	IPRange      zos.IPNet
	MyceliumKeys map[uint32][]byte

	// computed
	PublicNodeID     uint32
	NodesIPRange     map[uint32]zos.IPNet
	NodeDeploymentID map[uint32]uint64
}

// NewNetworkFromWorkload generates a new znet from a workload
func NewNetworkLightFromWorkload(wl zos.Workload, nodeID uint32) (ZNetLight, error) {
	data, err := wl.NetworkLightWorkload()
	if err != nil {
		return ZNetLight{}, errors.Errorf("could not create network light workload from data")
	}

	metadata := NetworkMetaData{}
	if err := json.Unmarshal([]byte(wl.Metadata), &metadata); err != nil {
		return ZNetLight{}, errors.Wrapf(err, "failed to parse network light metadata from workload %s", wl.Name)
	}

	var publicNodeID uint32
	if len(metadata.UserAccesses) > 0 {
		publicNodeID = metadata.UserAccesses[0].NodeID
	}

	myceliumKeys := make(map[uint32][]byte)
	if data.Mycelium.Key != nil {
		myceliumKeys[nodeID] = data.Mycelium.Key
	}

	return ZNetLight{
		Name:        wl.Name,
		Description: wl.Description,
		Nodes:       []uint32{nodeID},
		// IPRange:      data.NetworkIPRange,
		NodesIPRange: map[uint32]zos.IPNet{nodeID: zos.IPNet(data.Subnet)},
		PublicNodeID: publicNodeID,
		MyceliumKeys: myceliumKeys,
	}, nil
}

func (znet *ZNetLight) GetVersion() Version {
	return 4
}

func (znet *ZNetLight) GetNodes() []uint32 {
	return znet.Nodes
}

func (znet *ZNetLight) GetIPRange() zos.IPNet {
	return znet.IPRange
}

func (znet *ZNetLight) GetAccessWGConfig() string {
	return ""
}

func (znet *ZNetLight) GetExternalIP() *zos.IPNet {
	return nil
}

func (znet *ZNetLight) GetExternalSK() wgtypes.Key {
	return wgtypes.Key{}
}

func (znet *ZNetLight) GetName() string {
	return znet.Name
}

func (znet *ZNetLight) GetDescription() string {
	return znet.Description
}

func (znet *ZNetLight) GetSolutionType() string {
	return znet.SolutionType
}

func (znet *ZNetLight) GetNodesIPRange() map[uint32]zos.IPNet {
	return znet.NodesIPRange
}

func (znet *ZNetLight) GetNodeDeploymentID() map[uint32]uint64 {
	return znet.NodeDeploymentID
}

func (znet *ZNetLight) GetAddWGAccess() bool {
	return false
}

func (znet *ZNetLight) SetNodeDeploymentID(nodeDeploymentsIDs map[uint32]uint64) {
	znet.NodeDeploymentID = nodeDeploymentsIDs
}

func (znet *ZNetLight) SetNodes(nodes []uint32) {
	znet.Nodes = nodes
}

func (znet *ZNetLight) GetMyceliumKeys() map[uint32][]byte {
	return znet.MyceliumKeys
}

func (znet *ZNetLight) SetMyceliumKeys(keys map[uint32][]byte) {
	znet.MyceliumKeys = keys
}

func (znet *ZNetLight) GetPublicNodeID() uint32 {
	return znet.PublicNodeID
}

func (znet *ZNetLight) SetNodesIPRange(nodesIPRange map[uint32]zos.IPNet) {
	znet.NodesIPRange = nodesIPRange
}

func (znet *ZNetLight) SetKeys(keys map[uint32]wgtypes.Key) {
}

func (znet *ZNetLight) SetWGPort(wgPort map[uint32]int) {
}

func (znet *ZNetLight) SetAccessWGConfig(accessWGConfig string) {
}

func (znet *ZNetLight) SetExternalIP(ip *zos.IPNet) {
}

func (znet *ZNetLight) SetExternalSK(key wgtypes.Key) {
}

func (znet *ZNetLight) SetPublicNodeID(node uint32) {
	znet.PublicNodeID = node
}

// Validate validates a network light data
func (znet *ZNetLight) Validate() error {
	if err := validateName(znet.Name); err != nil {
		return errors.Wrap(err, "network name is invalid")
	}

	if len(znet.Nodes) == 0 {
		return fmt.Errorf("number of nodes in znet: %s, should be nonzero positive number", znet.Name)
	}

	mask := znet.IPRange.Mask
	if ones, _ := mask.Size(); ones != 16 {
		return errors.Errorf("subnet in ip range %s should be 16", znet.IPRange.String())
	}

	for node, key := range znet.MyceliumKeys {
		if len(key) != zos.MyceliumKeyLen && len(key) != 0 {
			return fmt.Errorf("invalid mycelium key length %d must be %d or empty", len(key), zos.MyceliumKeyLen)
		}

		if !slices.Contains(znet.Nodes, node) {
			return fmt.Errorf("invalid node %d for mycelium key, must be included in the network nodes %v", node, znet.Nodes)
		}
	}

	return nil
}

// InvalidateBrokenAttributes removes outdated attrs and deleted contracts
func (znet *ZNetLight) InvalidateBrokenAttributes(subConn subi.SubstrateExt, ncPool client.NodeClientGetter) error {
	for node, contractID := range znet.NodeDeploymentID {
		contract, err := subConn.GetContract(contractID)
		if (err == nil && !contract.IsCreated()) || errors.Is(err, substrate.ErrNotFound) {
			delete(znet.NodeDeploymentID, node)
			delete(znet.NodesIPRange, node)
		} else if err != nil {
			return errors.Wrapf(err, "could not get node %d contract %d", node, contractID)
		}
	}

	for node, ip := range znet.NodesIPRange {
		if !znet.IPRange.Contains(ip.IP) {
			delete(znet.NodesIPRange, node)
		}
	}

	if znet.PublicNodeID != 0 {
		// TODO: add a check that the node is still public
		cl, err := ncPool.GetNodeClient(subConn, znet.PublicNodeID)
		if err != nil {
			// whatever the error, delete it and it will get reassigned later
			znet.PublicNodeID = 0
		}
		if err := cl.IsNodeUp(context.Background()); err != nil {
			znet.PublicNodeID = 0
		}
	}

	return nil
}

// ZosWorkload generates a zos workload from a network
func (znet *ZNetLight) ZosWorkload(subnet zos.IPNet, _ string, _ uint16, _ []zos.Peer, metadata string, myceliumKey []byte) zos.Workload {
	return zos.Workload{
		Version:     0,
		Type:        zos.NetworkLightType,
		Description: znet.Description,
		Name:        znet.Name,
		Data: zos.MustMarshal(zos.NetworkLight{
			Subnet: subnet,
			Mycelium: zos.Mycelium{
				Key: myceliumKey,
			},
		}),
		Metadata: metadata,
	}
}

// GenerateMetadata generates deployment metadata
func (znet *ZNetLight) GenerateMetadata() (string, error) {
	if len(znet.SolutionType) == 0 {
		znet.SolutionType = "Network"
	}

	deploymentData := DeploymentData{
		Version:     int(Version4),
		Name:        znet.Name,
		Type:        "network-light",
		ProjectName: znet.SolutionType,
	}

	deploymentDataBytes, err := json.Marshal(deploymentData)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse deployment data %v", deploymentData)
	}

	return string(deploymentDataBytes), nil
}

// AssignNodesIPs assign network nodes ips
func (znet *ZNetLight) AssignNodesIPs(nodes []uint32) error {
	ips := make(map[uint32]zos.IPNet)
	l := len(znet.IPRange.IP)
	usedIPs := make([]byte, 0) // the third octet
	for node, ip := range znet.NodesIPRange {
		if Contains(nodes, node) {
			usedIPs = append(usedIPs, ip.IP[l-2])
			ips[node] = ip
		}
	}
	var cur byte = 2
	for _, nodeID := range nodes {
		if _, ok := ips[nodeID]; !ok {
			err := nextFreeIP(usedIPs, &cur)
			if err != nil {
				return err
			}
			usedIPs = append(usedIPs, cur)
			ips[nodeID] = IPNet(znet.IPRange.IP[l-4], znet.IPRange.IP[l-3], cur, znet.IPRange.IP[l-2], 24)
		}
	}
	znet.NodesIPRange = ips
	return nil
}

// GenerateVersionlessDeployments generates deployments for network without versions.
func (znet *ZNetLight) GenerateVersionlessDeployments(
	ctx context.Context,
	ncPool client.NodeClientGetter,
	subConn subi.SubstrateExt,
	twinID, publicNode uint32,
	allNodes map[uint32]struct{},
	_ map[uint32]net.IP,
	_ map[uint32][]uint16,
) (map[uint32]zos.Deployment, error) {
	return znet.generateDeployments(nil, nil, publicNode, twinID)
}

func (znet *ZNetLight) generateDeployments(_ map[uint32]net.IP, _ map[uint32][]uint16, publicNode, twinID uint32) (map[uint32]zos.Deployment, error) {
	deployments := make(map[uint32]zos.Deployment)

	log.Debug().Msgf("nodes: %v", znet.Nodes)

	if err := znet.AssignNodesIPs(znet.Nodes); err != nil {
		return nil, errors.Wrap(err, "could not assign node ips")
	}

	metadata := NetworkMetaData{
		Version: int(Version4),
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal network metadata")
	}

	for _, nodeID := range znet.Nodes {
		workload := znet.ZosWorkload(znet.NodesIPRange[nodeID], "", 0, nil, string(metadataBytes), znet.MyceliumKeys[nodeID])
		deployment := zos.NewGridDeployment(twinID, []zos.Workload{workload})

		// add metadata
		deployment.Metadata, err = znet.GenerateMetadata()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate deployment %s metadata", znet.Name)
		}

		deployments[nodeID] = deployment
	}

	return deployments, nil
}

// ReadNodesConfig reads the configuration of a network
func (znet *ZNetLight) ReadNodesConfig(ctx context.Context, nodeDeployments map[uint32]zos.Deployment) error {
	nodesIPRange := make(map[uint32]zos.IPNet)
	log.Debug().Msg("reading node config")
	for node, dl := range nodeDeployments {
		for _, wl := range dl.Workloads {
			if wl.Type != zos.NetworkLightType {
				continue
			}

			d, err := wl.NetworkLightWorkload()
			if err != nil {
				return errors.Wrap(err, "could not parse workload data")
			}

			nodesIPRange[node] = zos.IPNet(d.Subnet)
		}
	}

	znet.NodesIPRange = nodesIPRange
	return nil
}
