// Package workloads includes workloads types (vm, zdb, QSFS, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"github.com/pkg/errors"
	client "github.com/threefoldtech/tfgrid-sdk-go/grid-client/node"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/subi"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// ExternalSKZeroValue as its not empty when it is zero
var ExternalSKZeroValue = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

// UserAccess struct
type UserAccess struct {
	UserAddress        string
	UserSecretKey      string
	PublicNodePK       string
	AllowedIPs         []string
	PublicNodeEndpoint string
}

// NetworkMetaData is added to network workloads to help rebuilding networks when retrieved from the grid
type NetworkMetaData struct {
	UserAccessIP string `json:"ip"`
	PrivateKey   string `json:"priv_key"`
	PublicNodeID uint32 `json:"node_id"`
}

// ZNet is zos network workload
type ZNet struct {
	Name        string
	Description string
	Nodes       []uint32
	IPRange     gridtypes.IPNet
	AddWGAccess bool

	// computed
	SolutionType     string
	AccessWGConfig   string
	ExternalIP       *gridtypes.IPNet
	ExternalSK       wgtypes.Key
	PublicNodeID     uint32
	NodesIPRange     map[uint32]gridtypes.IPNet
	NodeDeploymentID map[uint32]uint64

	WGPort map[uint32]int
	Keys   map[uint32]wgtypes.Key
}

// NewNetworkFromWorkload generates a new znet from a workload
func NewNetworkFromWorkload(wl gridtypes.Workload, nodeID uint32) (ZNet, error) {
	dataI, err := wl.WorkloadData()
	if err != nil {
		return ZNet{}, errors.Wrap(err, "failed to get workload data")
	}

	data, ok := dataI.(*zos.Network)
	if !ok {
		return ZNet{}, errors.Errorf("could not create network workload from data %v", dataI)
	}

	keys := map[uint32]wgtypes.Key{}
	if data.WGPrivateKey != "" {
		wgKey, err := wgtypes.ParseKey(data.WGPrivateKey)
		if err != nil {
			return ZNet{}, errors.Errorf("could not parse wg private key: %s", data.WGPrivateKey)
		}
		keys[nodeID] = wgKey
	}

	wgPort := map[uint32]int{}
	if data.WGListenPort != 0 {
		wgPort[nodeID] = int(data.WGListenPort)
	}

	metadata := NetworkMetaData{}
	if err := json.Unmarshal([]byte(wl.Metadata), &metadata); err != nil {
		return ZNet{}, errors.Wrapf(err, "failed to parse network metadata from workload %s", wl.Name)
	}

	var externalIP *gridtypes.IPNet
	if metadata.UserAccessIP != "" {

		ipNet, err := gridtypes.ParseIPNet(metadata.UserAccessIP)
		if err != nil {
			return ZNet{}, err
		}

		externalIP = &ipNet
	}

	var externalSK wgtypes.Key
	if metadata.PrivateKey != "" {
		key, err := wgtypes.ParseKey(metadata.PrivateKey)
		if err != nil {
			return ZNet{}, errors.Wrap(err, "failed to parse user access private key")
		}
		externalSK = key
	}

	return ZNet{
		Name:         wl.Name.String(),
		Description:  wl.Description,
		Nodes:        []uint32{nodeID},
		IPRange:      data.NetworkIPRange,
		NodesIPRange: map[uint32]gridtypes.IPNet{nodeID: data.Subnet},
		WGPort:       wgPort,
		Keys:         keys,
		AddWGAccess:  externalIP != nil,
		PublicNodeID: metadata.PublicNodeID,
		ExternalIP:   externalIP,
		ExternalSK:   externalSK,
	}, nil
}

// NewIPRange generates a new IPRange from the given network IP
func NewIPRange(n net.IPNet) gridtypes.IPNet {
	return gridtypes.NewIPNet(n)
}

// Validate validates a network mask to be 16
func (znet *ZNet) Validate() error {
	mask := znet.IPRange.Mask
	if ones, _ := mask.Size(); ones != 16 {
		return errors.Errorf("subnet in ip range %s should be 16", znet.IPRange.String())
	}

	return nil
}

// ZosWorkload generates a zos workload from a network
func (znet *ZNet) ZosWorkload(subnet gridtypes.IPNet, wgPrivateKey string, wgListenPort uint16, peers []zos.Peer, metadata string) gridtypes.Workload {
	return gridtypes.Workload{
		Version:     0,
		Type:        zos.NetworkType,
		Description: znet.Description,
		Name:        gridtypes.Name(znet.Name),
		Data: gridtypes.MustMarshal(zos.Network{
			NetworkIPRange: gridtypes.MustParseIPNet(znet.IPRange.String()),
			Subnet:         subnet,
			WGPrivateKey:   wgPrivateKey,
			WGListenPort:   wgListenPort,
			Peers:          peers,
		}),
		Metadata: metadata,
	}
}

// GenerateMetadata generates deployment metadata
func (znet *ZNet) GenerateMetadata() (string, error) {
	if len(znet.SolutionType) == 0 {
		znet.SolutionType = "Network"
	}

	deploymentData := DeploymentData{
		Name:        znet.Name,
		Type:        "network",
		ProjectName: znet.SolutionType,
	}

	deploymentDataBytes, err := json.Marshal(deploymentData)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse deployment data %v", deploymentData)
	}

	return string(deploymentDataBytes), nil
}

// AssignNodesIPs assign network nodes ips
func (znet *ZNet) AssignNodesIPs(nodes []uint32) error {
	ips := make(map[uint32]gridtypes.IPNet)
	l := len(znet.IPRange.IP)
	usedIPs := make([]byte, 0) // the third octet
	for node, ip := range znet.NodesIPRange {
		if Contains(nodes, node) {
			usedIPs = append(usedIPs, ip.IP[l-2])
			ips[node] = ip
		}
	}
	var cur byte = 2
	if znet.AddWGAccess {
		if znet.ExternalIP != nil {
			usedIPs = append(usedIPs, znet.ExternalIP.IP[l-2])
		} else {
			err := nextFreeIP(usedIPs, &cur)
			if err != nil {
				return err
			}
			usedIPs = append(usedIPs, cur)
			ip := IPNet(znet.IPRange.IP[l-4], znet.IPRange.IP[l-3], cur, znet.IPRange.IP[l-1], 24)
			znet.ExternalIP = &ip
		}
	}
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

// AssignNodesWGPort assign network nodes wireguard port
func (znet *ZNet) AssignNodesWGPort(ctx context.Context, sub subi.SubstrateExt, ncPool client.NodeClientGetter, nodes []uint32, usedPorts map[uint32][]uint16) error {
	if usedPorts == nil {
		usedPorts = make(map[uint32][]uint16)
	}
	for _, nodeID := range nodes {
		if _, ok := znet.WGPort[nodeID]; ok {
			continue
		}
		cl, err := ncPool.GetNodeClient(sub, nodeID)
		if err != nil {
			return errors.Wrap(err, "could not get node client")
		}
		port, err := cl.GetNodeFreeWGPort(ctx, nodeID, usedPorts[nodeID])
		if err != nil {
			return errors.Wrapf(err, "failed to get node %d free wg ports", nodeID)
		}
		usedPorts[nodeID] = append(usedPorts[nodeID], uint16(port))

		if len(znet.WGPort) == 0 {
			znet.WGPort = map[uint32]int{nodeID: port}
			continue
		}
		znet.WGPort[nodeID] = port
	}

	return nil
}

// AssignNodesWGKey assign network nodes wireguard key
func (znet *ZNet) AssignNodesWGKey(nodes []uint32) error {
	for _, nodeID := range nodes {
		if _, ok := znet.Keys[nodeID]; !ok {

			key, err := wgtypes.GenerateKey()
			if err != nil {
				return errors.Wrap(err, "failed to generate wg private key")
			}

			if len(znet.Keys) == 0 {
				znet.Keys = map[uint32]wgtypes.Key{nodeID: key}
				continue
			}
			znet.Keys[nodeID] = key
		}
	}

	return nil
}

// IPNet returns an IP net type
func IPNet(a, b, c, d, msk byte) gridtypes.IPNet {
	return gridtypes.NewIPNet(net.IPNet{
		IP:   net.IPv4(a, b, c, d),
		Mask: net.CIDRMask(int(msk), 32),
	})
}

// WgIP return wireguard IP network
func WgIP(ip gridtypes.IPNet) gridtypes.IPNet {
	a := ip.IP[len(ip.IP)-3]
	b := ip.IP[len(ip.IP)-2]

	return gridtypes.NewIPNet(net.IPNet{
		IP:   net.IPv4(100, 64, a, b),
		Mask: net.CIDRMask(32, 32),
	})

}

// GenerateWGConfig generates wireguard configs
func GenerateWGConfig(Address string, AccessPrivatekey string, NodePublicKey string, NodeEndpoint string, NetworkIPRange string) string {

	return fmt.Sprintf(`
[Interface]
Address = %s
PrivateKey = %s
[Peer]
PublicKey = %s
AllowedIPs = %s, 100.64.0.0/16
PersistentKeepalive = 25
Endpoint = %s
	`, Address, AccessPrivatekey, NodePublicKey, NetworkIPRange, NodeEndpoint)
}

// nextFreeIP finds a free ip for a node
func nextFreeIP(used []byte, start *byte) error {
	for Contains(used, *start) && *start <= 254 {
		*start++
	}
	if *start == 255 {
		return errors.New("could not find a free ip to add node")
	}
	return nil
}
