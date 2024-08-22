package types

import "github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/app/dependency"

// DeploymentData is a helper struct to hold the deployment data to ease the deployment process.
type DeploymentData struct {
	ServicesGraph  *dependency.DRGraph
	NetworkNodeMap map[string]*NetworkData
}

// NetworkData is a helper struct to hold the network data to ease the deployment process.
// It holds the node id and the services that are part of a network.
type NetworkData struct {
	NodeID   uint32
	Services map[string]*Service
}
