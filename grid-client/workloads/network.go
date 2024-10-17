// Package workloads includes workloads types (vm, zdb, QSFS, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	r "math/rand"
	"net"
	"slices"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	client "github.com/threefoldtech/tfgrid-sdk-go/grid-client/node"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/subi"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// ExternalSKZeroValue as its not empty when it is zero
var ExternalSKZeroValue = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

// UserAccess struct
type UserAccess struct {
	Subnet     string `json:"subnet"`
	PrivateKey string `json:"private_key"`
	NodeID     uint32 `json:"node_id"`
}

// NetworkMetaData is added to network workloads to help rebuilding networks when retrieved from the grid
type NetworkMetaData struct {
	Version      int          `json:"version"`
	UserAccesses []UserAccess `json:"user_accesses"`
}

func (m *NetworkMetaData) UnmarshalJSON(data []byte) error {
	var deprecated struct {
		Version      int          `json:"version"`
		UserAccesses []UserAccess `json:"user_accesses"`
		// deprecated fields

		UserAccessIP string `json:"ip"`
		PrivateKey   string `json:"priv_key"`
		PublicNodeID uint32 `json:"node_id"`
	}
	if err := json.Unmarshal(data, &deprecated); err != nil {
		return err
	}
	m.Version = deprecated.Version
	m.UserAccesses = deprecated.UserAccesses
	if deprecated.UserAccessIP != "" || deprecated.PrivateKey != "" || deprecated.PublicNodeID != 0 {
		// it must be deprecated format
		m.UserAccesses = []UserAccess{{
			Subnet:     deprecated.UserAccessIP,
			PrivateKey: deprecated.PrivateKey,
			NodeID:     deprecated.PublicNodeID,
		}}
	}
	return nil
}

// ZNet is zos network workload
type ZNet struct {
	Name         string
	Description  string
	Nodes        []uint32
	IPRange      zos.IPNet
	AddWGAccess  bool
	MyceliumKeys map[uint32][]byte
	SolutionType string

	// computed
	AccessWGConfig   string
	ExternalIP       *zos.IPNet
	ExternalSK       wgtypes.Key
	PublicNodeID     uint32
	NodesIPRange     map[uint32]zos.IPNet
	NodeDeploymentID map[uint32]uint64

	WGPort map[uint32]int
	Keys   map[uint32]wgtypes.Key
}

// NewNetworkFromWorkload generates a new znet from a workload
func NewNetworkFromWorkload(wl zos.Workload, nodeID uint32) (ZNet, error) {
	data, err := wl.NetworkWorkload()
	if err != nil {
		return ZNet{}, errors.Errorf("could not create network workload from data")
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

	var externalIP *zos.IPNet
	if len(metadata.UserAccesses) > 0 && metadata.UserAccesses[0].Subnet != "" {

		ipNet, err := zos.ParseIPNet(metadata.UserAccesses[0].Subnet)
		if err != nil {
			return ZNet{}, err
		}

		externalIP = &ipNet
	}

	var externalSK wgtypes.Key
	if len(metadata.UserAccesses) > 0 && metadata.UserAccesses[0].PrivateKey != "" {
		key, err := wgtypes.ParseKey(metadata.UserAccesses[0].PrivateKey)
		if err != nil {
			return ZNet{}, errors.Wrap(err, "failed to parse user access private key")
		}
		externalSK = key
	}
	var publicNodeID uint32
	if len(metadata.UserAccesses) > 0 {
		publicNodeID = metadata.UserAccesses[0].NodeID
	}
	myceliumKeys := make(map[uint32][]byte)
	if data.Mycelium != nil {
		myceliumKeys[nodeID] = data.Mycelium.Key
	}

	return ZNet{
		Name:         wl.Name,
		Description:  wl.Description,
		Nodes:        []uint32{nodeID},
		IPRange:      zos.IPNet(data.NetworkIPRange),
		NodesIPRange: map[uint32]zos.IPNet{nodeID: zos.IPNet(data.Subnet)},
		WGPort:       wgPort,
		Keys:         keys,
		AddWGAccess:  externalIP != nil,
		PublicNodeID: publicNodeID,
		ExternalIP:   externalIP,
		ExternalSK:   externalSK,
		MyceliumKeys: myceliumKeys,
	}, nil
}

// NewIPRange generates a new IPRange from the given network IP
func NewIPRange(n net.IPNet) zos.IPNet {
	return zos.IPNet{IPNet: n}
}

// Validate validates a network data
func (znet *ZNet) Validate() error {
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
func (znet *ZNet) InvalidateBrokenAttributes(subConn subi.SubstrateExt, ncPool client.NodeClientGetter) error {
	for node, contractID := range znet.NodeDeploymentID {
		contract, err := subConn.GetContract(contractID)
		if (err == nil && !contract.IsCreated()) || errors.Is(err, substrate.ErrNotFound) {
			delete(znet.NodeDeploymentID, node)
			delete(znet.NodesIPRange, node)
			delete(znet.Keys, node)
			delete(znet.WGPort, node)
		} else if err != nil {
			return errors.Wrapf(err, "could not get node %d contract %d", node, contractID)
		}
	}
	if znet.ExternalIP != nil && !znet.IPRange.Contains(znet.ExternalIP.IP) {
		znet.ExternalIP = nil
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

	if !znet.AddWGAccess {
		znet.ExternalIP = nil
	}
	return nil
}

// ZosWorkload generates a zos workload from a network
func (znet *ZNet) ZosWorkload(subnet zos.IPNet, wgPrivateKey string, wgListenPort uint16, peers []zos.Peer, metadata string, myceliumKey []byte) zos.Workload {
	var mycelium *zos.Mycelium
	if len(myceliumKey) != 0 {
		mycelium = &zos.Mycelium{Key: myceliumKey}
	}
	return zos.Workload{
		Version:     0,
		Type:        zos.NetworkType,
		Description: znet.Description,
		Name:        znet.Name,
		Data: zos.MustMarshal(zos.Network{
			NetworkIPRange: zos.MustParseIPNet(znet.IPRange.String()),
			Subnet:         subnet,
			WGPrivateKey:   wgPrivateKey,
			WGListenPort:   wgListenPort,
			Peers:          peers,
			Mycelium:       mycelium,
		}),
		Metadata: metadata,
	}
}

func (znet *ZNet) GetVersion() Version {
	return 3
}

func (znet *ZNet) GetNodes() []uint32 {
	return znet.Nodes
}

func (znet *ZNet) GetIPRange() zos.IPNet {
	return znet.IPRange
}

func (znet *ZNet) GetAccessWGConfig() string {
	return znet.AccessWGConfig
}

func (znet *ZNet) GetExternalIP() *zos.IPNet {
	return znet.ExternalIP
}

func (znet *ZNet) GetExternalSK() wgtypes.Key {
	return znet.ExternalSK
}

func (znet *ZNet) SetNodes(nodes []uint32) {
	znet.Nodes = nodes
}

func (znet *ZNet) GetMyceliumKeys() map[uint32][]byte {
	return znet.MyceliumKeys
}

func (znet *ZNet) SetMyceliumKeys(keys map[uint32][]byte) {
	znet.MyceliumKeys = keys
}

func (znet *ZNet) SetKeys(keys map[uint32]wgtypes.Key) {
	znet.Keys = keys
}

func (znet *ZNet) SetExternalIP(ip *zos.IPNet) {
	znet.ExternalIP = ip
}

func (znet *ZNet) SetExternalSK(key wgtypes.Key) {
	znet.ExternalSK = key
}

func (znet *ZNet) SetPublicNodeID(node uint32) {
	znet.PublicNodeID = node
}

func (znet *ZNet) SetNodesIPRange(nodesIPRange map[uint32]zos.IPNet) {
	znet.NodesIPRange = nodesIPRange
}

func (znet *ZNet) SetWGPort(wgPort map[uint32]int) {
	znet.WGPort = wgPort
}

func (znet *ZNet) SetAccessWGConfig(accessWGConfig string) {
	znet.AccessWGConfig = accessWGConfig
}

func (znet *ZNet) GetName() string {
	return znet.Name
}

func (znet *ZNet) GetDescription() string {
	return znet.Description
}

func (znet *ZNet) GetSolutionType() string {
	return znet.SolutionType
}

func (znet *ZNet) GetNodesIPRange() map[uint32]zos.IPNet {
	return znet.NodesIPRange
}

func (znet *ZNet) GetAddWGAccess() bool {
	return znet.AddWGAccess
}

func (znet *ZNet) GetNodeDeploymentID() map[uint32]uint64 {
	return znet.NodeDeploymentID
}

func (znet *ZNet) SetNodeDeploymentID(nodeDeploymentsIDs map[uint32]uint64) {
	znet.NodeDeploymentID = nodeDeploymentsIDs
}

func (znet *ZNet) GetPublicNodeID() uint32 {
	return znet.PublicNodeID
}

// GenerateMetadata generates deployment metadata
func (znet *ZNet) GenerateMetadata() (string, error) {
	if len(znet.SolutionType) == 0 {
		znet.SolutionType = "Network"
	}

	deploymentData := DeploymentData{
		Version:     int(Version3),
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
func IPNet(a, b, c, d, msk byte) zos.IPNet {
	return zos.IPNet{IPNet: net.IPNet{
		IP:   net.IPv4(a, b, c, d),
		Mask: net.CIDRMask(int(msk), 32),
	}}
}

// WgIP return wireguard IP network
func WgIP(ip zos.IPNet) zos.IPNet {
	a := ip.IP[len(ip.IP)-3]
	b := ip.IP[len(ip.IP)-2]

	return zos.IPNet{IPNet: net.IPNet{
		IP:   net.IPv4(100, 64, a, b),
		Mask: net.CIDRMask(32, 32),
	}}
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

func RandomMyceliumKey() ([]byte, error) {
	key := make([]byte, zos.MyceliumKeyLen)
	_, err := rand.Read(key)
	return key, err
}

// GenerateVersionlessDeployments generates deployments for network without versions.
func (znet *ZNet) GenerateVersionlessDeployments(
	ctx context.Context,
	ncPool client.NodeClientGetter,
	subConn subi.SubstrateExt,
	twinID, publicNode uint32,
	allNodes map[uint32]struct{},
	endpoints map[uint32]net.IP,
	nodeUsedPorts map[uint32][]uint16,
) (map[uint32]zos.Deployment, error) {
	var multiErr error

	var wg sync.WaitGroup
	var mu sync.Mutex

	for nodeID := range allNodes {
		wg.Add(1)
		go func(nodeID uint32) {
			defer wg.Done()
			endpoint, usedPorts, err := getNodeEndpointAndPorts(ctx, ncPool, subConn, nodeID)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				multiErr = multierror.Append(multiErr, err)
				return
			}
			endpoints[nodeID] = endpoint
			nodeUsedPorts[nodeID] = usedPorts
		}(nodeID)

	}
	wg.Wait()

	// here we check that the we managed to get a public node
	// and that public node wasn't already processed. if we got
	// an error while getting public node, then node id will be 0
	// and we will skip getting its data.
	if _, ok := endpoints[publicNode]; !ok && publicNode != 0 {
		endpoint, usedPorts, err := getNodeEndpointAndPorts(ctx, ncPool, subConn, publicNode)
		if err != nil {
			multiErr = multierror.Append(multiErr, err)
		} else {
			endpoints[publicNode] = endpoint
			nodeUsedPorts[publicNode] = usedPorts
		}
	}

	dls, err := znet.generateDeployments(endpoints, nodeUsedPorts, publicNode, twinID)
	if err != nil {
		multiErr = multierror.Append(multiErr, err)
	}

	return dls, multiErr
}

func (znet *ZNet) generateDeployments(endpointIPs map[uint32]net.IP, usedPorts map[uint32][]uint16, publicNode, twinID uint32) (map[uint32]zos.Deployment, error) {
	deployments := make(map[uint32]zos.Deployment)

	log.Debug().Msgf("nodes: %v", znet.Nodes)

	endpoints := make(map[uint32]string)
	hiddenNodes := make([]uint32, 0)
	accessibleNodes := make([]uint32, 0)
	var ipv4Node uint32

	for _, nodeID := range znet.Nodes {
		if _, ok := endpointIPs[nodeID]; !ok {
			// means the network has a failing node
			// we will skip generating deployments for it.
			return nil, fmt.Errorf("failed to process network %s", znet.Name)
		}
		if endpointIPs[nodeID] == nil {
			hiddenNodes = append(hiddenNodes, nodeID)
		} else if endpointIPs[nodeID].To4() != nil {
			accessibleNodes = append(accessibleNodes, nodeID)
			ipv4Node = nodeID
			endpoints[nodeID] = endpointIPs[nodeID].String()
		} else {
			accessibleNodes = append(accessibleNodes, nodeID)
			endpoints[nodeID] = fmt.Sprintf("[%s]", endpointIPs[nodeID].String())
		}
	}

	needsIPv4Access := znet.AddWGAccess || (len(hiddenNodes) != 0 && len(hiddenNodes)+len(accessibleNodes) > 1)
	if needsIPv4Access {
		if znet.PublicNodeID != 0 { // it's set
			// if public node id is already set, it should be added to accessible nodes
			if !Contains(accessibleNodes, znet.PublicNodeID) {
				accessibleNodes = append(accessibleNodes, znet.PublicNodeID)
			}
		} else if ipv4Node != 0 { // there's one in the network original nodes
			znet.PublicNodeID = ipv4Node
		} else if _, ok := endpointIPs[publicNode]; !ok || publicNode == 0 {
			// that means either we didn't find a public node
			// or we failed to get its endpoint, so can't continue with the network
			return nil, fmt.Errorf("failed to get public node for %s", znet.Name)
		} else {
			znet.PublicNodeID = publicNode
			accessibleNodes = append(accessibleNodes, publicNode)
			endpoints[publicNode] = endpointIPs[publicNode].String()
		}
	}

	allNodes := append(hiddenNodes, accessibleNodes...)
	if err := znet.AssignNodesIPs(allNodes); err != nil {
		return nil, errors.Wrap(err, "could not assign node ips")
	}
	if err := znet.AssignNodesWGKey(allNodes); err != nil {
		return nil, errors.Wrap(err, "could not assign node wg keys")
	}
	if znet.WGPort == nil {
		znet.WGPort = make(map[uint32]int)
	}

	// assign WireGuard ports
	for _, nodeID := range allNodes {
		nodeUsedPorts := usedPorts[nodeID]
		p := uint16(r.Intn(32768-1024) + 1024)
		for slices.Contains(nodeUsedPorts, p) {
			p = uint16(r.Intn(32768-1024) + 1024)
		}
		nodeUsedPorts = append(nodeUsedPorts, p)
		usedPorts[nodeID] = nodeUsedPorts
		znet.WGPort[nodeID] = int(p)
	}

	nonAccessibleIPRanges := []zos.IPNet{}
	for _, nodeID := range hiddenNodes {
		r := znet.NodesIPRange[nodeID]
		nonAccessibleIPRanges = append(nonAccessibleIPRanges, r)
		nonAccessibleIPRanges = append(nonAccessibleIPRanges, WgIP(r))
	}
	if znet.AddWGAccess {
		r := znet.ExternalIP
		nonAccessibleIPRanges = append(nonAccessibleIPRanges, *r)
		nonAccessibleIPRanges = append(nonAccessibleIPRanges, WgIP(*r))
	}

	log.Debug().Msgf("hidden nodes: %v", hiddenNodes)
	log.Debug().Uint32("public node", znet.PublicNodeID)
	log.Debug().Msgf("accessible nodes: %v", accessibleNodes)
	log.Debug().Msgf("non accessible ip ranges: %v", nonAccessibleIPRanges)

	if znet.AddWGAccess {
		// if no wg private key, it should be generated
		if znet.ExternalSK.String() == ExternalSKZeroValue {
			wgSK, err := wgtypes.GeneratePrivateKey()
			if err != nil {
				return nil, errors.Wrapf(err, "failed to generate wireguard secret key for network: %s", znet.Name)
			}
			znet.ExternalSK = wgSK
		}

		znet.AccessWGConfig = GenerateWGConfig(
			WgIP(*znet.ExternalIP).IP.String(),
			znet.ExternalSK.String(),
			znet.Keys[znet.PublicNodeID].PublicKey().String(),
			fmt.Sprintf("%s:%d", endpoints[znet.PublicNodeID], znet.WGPort[znet.PublicNodeID]),
			znet.IPRange.String(),
		)
	}

	externalIP := ""
	if znet.ExternalIP != nil {
		externalIP = znet.ExternalIP.String()
	}
	metadata := NetworkMetaData{
		Version: int(Version3),
		UserAccesses: []UserAccess{
			{
				Subnet:     externalIP,
				PrivateKey: znet.ExternalSK.String(),
				NodeID:     znet.PublicNodeID,
			},
		},
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal network metadata")
	}

	// accessible nodes deployments
	for _, nodeID := range accessibleNodes {
		peers := make([]zos.Peer, 0, len(znet.Nodes))
		for _, peerNodeID := range accessibleNodes {
			if peerNodeID == nodeID {
				continue
			}

			peerIPRange := znet.NodesIPRange[peerNodeID]
			allowedIPs := []zos.IPNet{
				peerIPRange,
				WgIP(peerIPRange),
			}

			if peerNodeID == znet.PublicNodeID {
				allowedIPs = append(allowedIPs, nonAccessibleIPRanges...)
			}

			peers = append(peers, zos.Peer{
				Subnet:      znet.NodesIPRange[peerNodeID],
				WGPublicKey: znet.Keys[peerNodeID].PublicKey().String(),
				Endpoint:    fmt.Sprintf("%s:%d", endpoints[peerNodeID], znet.WGPort[peerNodeID]),
				AllowedIPs:  allowedIPs,
			})
		}

		if nodeID == znet.PublicNodeID {
			// external node
			if znet.AddWGAccess {
				peers = append(peers, zos.Peer{
					Subnet:      *znet.ExternalIP,
					WGPublicKey: znet.ExternalSK.PublicKey().String(),
					AllowedIPs:  []zos.IPNet{*znet.ExternalIP, WgIP(*znet.ExternalIP)},
				})
			}

			// hidden nodes
			for _, peerNodeID := range hiddenNodes {
				peerIPRange := znet.NodesIPRange[peerNodeID]
				peers = append(peers, zos.Peer{
					Subnet:      peerIPRange,
					WGPublicKey: znet.Keys[peerNodeID].PublicKey().String(),
					AllowedIPs: []zos.IPNet{
						peerIPRange,
						WgIP(peerIPRange),
					},
				})
			}
		}

		workload := znet.ZosWorkload(znet.NodesIPRange[nodeID], znet.Keys[nodeID].String(), uint16(znet.WGPort[nodeID]), peers, string(metadataBytes), znet.MyceliumKeys[nodeID])
		deployment := zos.NewGridDeployment(twinID, []zos.Workload{workload})

		// add metadata
		deployment.Metadata, err = znet.GenerateMetadata()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate deployment %s metadata", znet.Name)
		}

		deployments[nodeID] = deployment
	}

	// hidden nodes deployments
	for _, nodeID := range hiddenNodes {
		peers := make([]zos.Peer, 0)
		if znet.PublicNodeID != 0 {
			peers = append(peers, zos.Peer{
				WGPublicKey: znet.Keys[znet.PublicNodeID].PublicKey().String(),
				Subnet:      znet.NodesIPRange[nodeID],
				AllowedIPs: []zos.IPNet{
					znet.IPRange,
					IPNet(100, 64, 0, 0, 16),
				},
				Endpoint: fmt.Sprintf("%s:%d", endpoints[znet.PublicNodeID], znet.WGPort[znet.PublicNodeID]),
			})
		}
		workload := znet.ZosWorkload(znet.NodesIPRange[nodeID], znet.Keys[nodeID].String(), uint16(znet.WGPort[nodeID]), peers, string(metadataBytes), znet.MyceliumKeys[nodeID])
		deployment := zos.NewGridDeployment(twinID, []zos.Workload{workload})

		// add metadata
		var err error
		deployment.Metadata, err = znet.GenerateMetadata()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate deployment %s metadata", znet.Name)
		}

		deployments[nodeID] = deployment
	}
	return deployments, nil
}

func getNodeEndpointAndPorts(ctx context.Context, ncPool client.NodeClientGetter, subConn subi.SubstrateExt, nodeID uint32) (net.IP, []uint16, error) {
	nodeClient, err := ncPool.GetNodeClient(subConn, nodeID)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get node %d client: %w", nodeID, err)
	}

	endpoint, err := nodeClient.GetNodeEndpoint(ctx)
	if err != nil && !errors.Is(err, client.ErrNoAccessibleInterfaceFound) {
		return nil, nil, fmt.Errorf("failed to get node %d endpoint: %w", nodeID, err)
	}

	usedPorts, err := nodeClient.NetworkListWGPorts(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get node %d used ports: %w", nodeID, err)
	}
	return endpoint, usedPorts, nil
}

// ReadNodesConfig reads the configuration of a network
func (znet *ZNet) ReadNodesConfig(ctx context.Context, nodeDeployments map[uint32]zos.Deployment) error {
	keys := make(map[uint32]wgtypes.Key)
	WGPort := make(map[uint32]int)
	nodesIPRange := make(map[uint32]zos.IPNet)

	log.Debug().Msg("reading node config")
	WGAccess := false
	for node, dl := range nodeDeployments {
		for _, wl := range dl.Workloads {
			if wl.Type != zos.NetworkType {
				continue
			}

			d, err := wl.NetworkWorkload()
			if err != nil {
				return errors.Wrap(err, "could not parse workload data")
			}

			WGPort[node] = int(d.WGListenPort)
			keys[node], err = wgtypes.ParseKey(d.WGPrivateKey)
			if err != nil {
				return errors.Wrap(err, "could not parse wg private key from workload object")
			}
			nodesIPRange[node] = zos.IPNet(d.Subnet)
			// this will fail when hidden node is supported
			for _, peer := range d.Peers {
				if peer.Endpoint == "" {
					WGAccess = true
				}
			}
		}
	}
	znet.Keys = keys
	znet.WGPort = WGPort
	znet.NodesIPRange = nodesIPRange
	znet.AddWGAccess = WGAccess
	if !WGAccess {
		znet.AccessWGConfig = ""
	}
	return nil
}
