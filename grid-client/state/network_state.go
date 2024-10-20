// Package state for grid state
package state

import (
	"sync"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
)

// NetworkState is a struct of networks names and their networks and mutex to protect the state
type NetworkState struct {
	State     map[string]Network
	stateLock sync.Mutex
}

// Network struct includes Subnets and node IPs
type Network struct {
	Subnets map[uint32]string
}

// NewNetwork creates a new Network
func NewNetwork() Network {
	return Network{
		Subnets: map[uint32]string{},
	}
}

// GetNetwork get a Network using its name
func (nm *NetworkState) GetNetwork(networkName string) Network {
	nm.stateLock.Lock()
	defer nm.stateLock.Unlock()

	if _, ok := nm.State[networkName]; !ok {
		nm.State[networkName] = NewNetwork()
	}
	net := nm.State[networkName]
	return net
}

// UpdateNetworkSubnets updates a network subnets given its name
func (nm *NetworkState) UpdateNetworkSubnets(networkName string, ipRange map[uint32]zos.IPNet) {
	network := nm.GetNetwork(networkName)
	network.Subnets = map[uint32]string{}
	for nodeID, subnet := range ipRange {
		network.SetNodeSubnet(nodeID, subnet.String())
	}

	nm.stateLock.Lock()
	defer nm.stateLock.Unlock()
	nm.State[networkName] = network
}

// DeleteNetwork deletes a Network using its name
func (nm *NetworkState) DeleteNetwork(networkName string) {
	nm.stateLock.Lock()
	defer nm.stateLock.Unlock()
	delete(nm.State, networkName)
}

// GetNodeSubnet gets a node subnet using its ID
func (n *Network) GetNodeSubnet(nodeID uint32) string {
	return n.Subnets[nodeID]
}

// SetNodeSubnet sets a node subnet with its ID and subnet
func (n *Network) SetNodeSubnet(nodeID uint32, subnet string) {
	n.Subnets[nodeID] = subnet
}

// DeleteNodeSubnet deletes a node subnet using its ID
func (n *Network) deleteNodeSubnet(nodeID uint32) {
	delete(n.Subnets, nodeID)
}
