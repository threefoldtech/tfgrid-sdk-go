package workloads

import (
	"context"
	"net"

	client "github.com/threefoldtech/tfgrid-sdk-go/grid-client/node"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/subi"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type Version int

const (
	Version3 Version = 3
	Version4 Version = 4
)

type Network interface {
	Validate() error
	GetVersion() Version
	ZosWorkload(subnet zos.IPNet, wgPrivateKey string, wgListenPort uint16, peers []zos.Peer, metadata string, myceliumKey []byte) zos.Workload
	GenerateMetadata() (string, error)
	InvalidateBrokenAttributes(subConn subi.SubstrateExt, ncPool client.NodeClientGetter) error
	GenerateVersionlessDeployments(
		ctx context.Context,
		ncPool client.NodeClientGetter,
		subConn subi.SubstrateExt,
		twinID, publicNode uint32,
		allNodes map[uint32]struct{},
		endpoints map[uint32]net.IP,
		nodeUsedPorts map[uint32][]uint16,
	) (map[uint32]zos.Deployment, error)
	ReadNodesConfig(ctx context.Context, nodeDeployments map[uint32]zos.Deployment) error

	GetNodes() []uint32
	GetNodeDeploymentID() map[uint32]uint64
	GetPublicNodeID() uint32
	GetNodesIPRange() map[uint32]zos.IPNet
	GetName() string
	GetSolutionType() string
	GetDescription() string
	GetAddWGAccess() bool
	GetMyceliumKeys() map[uint32][]byte
	GetIPRange() zos.IPNet
	GetAccessWGConfig() string
	GetExternalIP() *zos.IPNet
	GetExternalSK() wgtypes.Key

	SetNodeDeploymentID(nodeDeploymentsIDs map[uint32]uint64)
	SetNodes(nodes []uint32)
	SetMyceliumKeys(keys map[uint32][]byte)
	SetKeys(keys map[uint32]wgtypes.Key)
	SetNodesIPRange(nodesIPRange map[uint32]zos.IPNet)
	SetWGPort(wgPort map[uint32]int)
	SetAccessWGConfig(accessWGConfig string)
	SetExternalIP(ip *zos.IPNet)
	SetExternalSK(key wgtypes.Key)
	SetPublicNodeID(node uint32)
}
