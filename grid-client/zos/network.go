package zos

import (
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// Network is the description of a part of a network local to a specific node.
// A network workload defines a wireguard network that is usually spans multiple nodes. One of the nodes must work as an access node
// in other words, it must be reachable from other nodes, hence it needs to have a `PublicConfig`.
// Since the user library creates all deployments upfront then all wireguard keys, and ports must be pre-deterministic and must be
// also created upfront.
// A network structure basically must consist of
// - The network information (IP range) must be an ipv4 /16 range
// - The local (node) peer definition (subnet of the network ip range, wireguard secure key, wireguard port if any)
// - List of other peers that are part of the same network with their own config
// - For each PC or a laptop (for each wireguard peer) there must be a peer in the peer list (on all nodes)
// This is why this can get complicated.
type Network struct {
	// IP range of the network, must be an IPv4 /16
	// for example a 10.1.0.0/16
	NetworkIPRange IPNet `json:"ip_range"`

	// IPV4 subnet for this network resource
	// this must be a valid subnet of the entire network ip range.
	// for example 10.1.1.0/24
	Subnet IPNet `json:"subnet"`

	// The private wg key of this node (this peer) which is installing this
	// network workload right now.
	// This has to be filled in by the user (and not generated for example)
	// because other peers need to be installed as well (with this peer public key)
	// hence it's easier to configure everything one time at the user side and then
	// apply everything on all nodes at once
	WGPrivateKey string `json:"wireguard_private_key"`
	// WGListenPort is the wireguard listen port on this node. this has
	// to be filled in by the user for same reason as private key (other nodes need to know about it)
	// To find a free port you have to ask the node first by a call over RMB about which ports are possible
	// to use.
	WGListenPort uint16 `json:"wireguard_listen_port"`

	// Peers is a list of other peers in this network
	Peers []Peer `json:"peers"`

	// Optional mycelium configuration. If provided
	// VMs in this network can use the mycelium feature.
	// if no mycelium configuration is provided, vms can't
	// get mycelium IPs.
	Mycelium *Mycelium `json:"mycelium,omitempty"`
}

// Peer is the description of a peer of a NetResource
type Peer struct {
	// IPV4 subnet of the network resource of the peer
	Subnet IPNet `json:"subnet"`
	// WGPublicKey of the peer (driven from its private key)
	WGPublicKey string `json:"wireguard_public_key"`
	// Allowed Ips is related to his subnet.
	// todo: remove and derive from subnet
	AllowedIPs []IPNet `json:"allowed_ips"`
	// Entrypoint of the peer
	Endpoint string `json:"endpoint"`
}

func (wl *Workload) NetworkWorkload() (*zos.Network, error) {
	dataI, err := wl.Workload3().WorkloadData()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workload data")
	}

	data, ok := dataI.(*zos.Network)
	if !ok {
		return nil, errors.Errorf("could not create network workload from data %v", dataI)
	}

	return data, nil
}
