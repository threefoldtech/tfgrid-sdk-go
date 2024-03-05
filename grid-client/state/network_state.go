// Package state for grid state
package state

import "github.com/threefoldtech/zos/pkg/gridtypes"

// NetworkState is a map of of names and their networks
type NetworkState map[string]Network

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
func (nm NetworkState) GetNetwork(networkName string) Network {
	if _, ok := nm[networkName]; !ok {
		nm[networkName] = NewNetwork()
	}
	net := nm[networkName]
	return net
}

// UpdateNetworkSubnets updates a network subnets given its name
func (nm NetworkState) UpdateNetworkSubnets(networkName string, ipRange map[uint32]gridtypes.IPNet) {
	network := nm.GetNetwork(networkName)
	network.Subnets = map[uint32]string{}
	for nodeID, subnet := range ipRange {
		network.SetNodeSubnet(nodeID, subnet.String())
	}
	nm[networkName] = network
}

// DeleteNetwork deletes a Network using its name
func (nm NetworkState) DeleteNetwork(networkName string) {
	delete(nm, networkName)
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
