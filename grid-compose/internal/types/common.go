package types

type DeploymentData struct {
	ServicesGraph  *DRGraph
	NetworkNodeMap map[string]*NetworkData
}

type NetworkData struct {
	NodeID   uint32
	Services map[string]*Service
}
