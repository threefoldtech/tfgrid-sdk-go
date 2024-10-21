package zos

import (
	"encoding/hex"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos4/pkg/gridtypes/zos"
)

// Bytes value that is represented as hex when serialized to json
type Bytes []byte

// NetworkLight is the description of a part of a network local to a specific node.
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
type NetworkLight struct {
	// IPV4 subnet for this network resource
	// this must be a valid subnet of the entire network ip range.
	// for example 10.1.1.0/24
	Subnet IPNet `json:"subnet"`

	// Optional mycelium configuration. If provided
	// VMs in this network can use the mycelium feature.
	// if no mycelium configuration is provided, vms can't
	// get mycelium IPs.
	Mycelium Mycelium `json:"mycelium,omitempty"`
}

type Mycelium struct {
	// Key is the key of the mycelium peer in the mycelium node
	// associated with this network.
	// It's provided by the user so it can be later moved to other nodes
	// without losing the key.
	Key Bytes `json:"hex_key"`
	// An optional mycelium peer list to be used with this node, otherwise
	// the default peer list is used.
	Peers []string `json:"peers"`
}

func (h *Bytes) UnmarshalText(text []byte) error {
	data, err := hex.DecodeString(string(text))
	if err != nil {
		return err
	}

	*h = data
	return nil
}

func (h Bytes) MarshalText() (text []byte, err error) {
	return []byte(hex.EncodeToString(h)), nil
}

func (wl *Workload) NetworkLightWorkload() (*zos.NetworkLight, error) {
	dataI, err := wl.Workload4().WorkloadData()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workload data")
	}

	data, ok := dataI.(*zos.NetworkLight)
	if !ok {
		return nil, errors.Errorf("could not create network workload from data %v", dataI)
	}

	return data, nil
}
