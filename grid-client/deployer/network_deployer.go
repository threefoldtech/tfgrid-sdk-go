package deployer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	client "github.com/threefoldtech/tfgrid-sdk-go/grid-client/node"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// NetworkDeployer struct
type NetworkDeployer struct {
	tfPluginClient *TFPluginClient
	deployer       MockDeployer
}

// NewNetworkDeployer generates a new network deployer
func NewNetworkDeployer(tfPluginClient *TFPluginClient) NetworkDeployer {
	deployer := NewDeployer(*tfPluginClient, true)
	return NetworkDeployer{
		tfPluginClient: tfPluginClient,
		deployer:       &deployer,
	}
}

// Validate validates a network deployer
func (d *NetworkDeployer) Validate(ctx context.Context, znet *workloads.ZNet) error {
	sub := d.tfPluginClient.SubstrateConn

	if err := validateAccountBalanceForExtrinsics(sub, d.tfPluginClient.Identity); err != nil {
		return err
	}

	if err := znet.Validate(); err != nil {
		return err
	}

	err := client.AreNodesUp(ctx, sub, znet.Nodes, d.tfPluginClient.NcPool)
	if err != nil {
		return err
	}

	return d.InvalidateBrokenAttributes(znet)
}

// GenerateVersionlessDeployments generates deployments for network deployer without versions.
// usedPorts can be used to exclude some ports from being assigned to networks
func (d *NetworkDeployer) GenerateVersionlessDeployments(ctx context.Context, znet *workloads.ZNet, usedPorts map[uint32][]uint16) (map[uint32]gridtypes.Deployment, error) {
	deployments := make(map[uint32]gridtypes.Deployment)

	log.Debug().Msgf("nodes: %v", znet.Nodes)
	sub := d.tfPluginClient.SubstrateConn

	endpoints := make(map[uint32]string)
	hiddenNodes := make([]uint32, 0)
	accessibleNodes := make([]uint32, 0)
	var ipv4Node uint32

	for _, nodeID := range znet.Nodes {
		nodeClient, err := d.tfPluginClient.NcPool.GetNodeClient(sub, nodeID)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get node %d client", nodeID)
		}

		endpoint, err := nodeClient.GetNodeEndpoint(ctx)
		if errors.Is(err, client.ErrNoAccessibleInterfaceFound) {
			hiddenNodes = append(hiddenNodes, nodeID)
		} else if err != nil {
			return nil, errors.Wrapf(err, "failed to get node %d endpoint", nodeID)
		} else if endpoint.To4() != nil {
			accessibleNodes = append(accessibleNodes, nodeID)
			ipv4Node = nodeID
			endpoints[nodeID] = endpoint.String()
		} else {
			accessibleNodes = append(accessibleNodes, nodeID)
			endpoints[nodeID] = fmt.Sprintf("[%s]", endpoint.String())
		}
	}

	needsIPv4Access := znet.AddWGAccess || (len(hiddenNodes) != 0 && len(hiddenNodes)+len(accessibleNodes) > 1)
	if needsIPv4Access {
		if znet.PublicNodeID != 0 { // it's set
			// if public node id is already set, it should be added to accessible nodes
			if !workloads.Contains(accessibleNodes, znet.PublicNodeID) {
				accessibleNodes = append(accessibleNodes, znet.PublicNodeID)
			}
		} else if ipv4Node != 0 { // there's one in the network original nodes
			znet.PublicNodeID = ipv4Node
		} else {
			publicNode, err := GetPublicNode(ctx, *d.tfPluginClient, []uint32{})
			if err != nil {
				return nil, errors.Wrap(err, "public node needed because you requested adding wg access or a hidden node is added to the network")
			}
			znet.PublicNodeID = publicNode
			accessibleNodes = append(accessibleNodes, publicNode)
		}

		if endpoints[znet.PublicNodeID] == "" { // old or new outsider
			cl, err := d.tfPluginClient.NcPool.GetNodeClient(sub, znet.PublicNodeID)
			if err != nil {
				return nil, errors.Wrapf(err, "could not get node %d client", znet.PublicNodeID)
			}
			endpoint, err := cl.GetNodeEndpoint(ctx)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get node %d endpoint", znet.PublicNodeID)
			}
			endpoints[znet.PublicNodeID] = endpoint.String()
		}
	}

	allNodes := append(hiddenNodes, accessibleNodes...)
	if err := znet.AssignNodesIPs(allNodes); err != nil {
		return nil, errors.Wrap(err, "could not assign node ips")
	}
	if err := znet.AssignNodesWGKey(allNodes); err != nil {
		return nil, errors.Wrap(err, "could not assign node wg keys")
	}
	if err := znet.AssignNodesWGPort(ctx, sub, d.tfPluginClient.NcPool, allNodes, usedPorts); err != nil {
		return nil, errors.Wrap(err, "could not assign node wg ports")
	}

	nonAccessibleIPRanges := []gridtypes.IPNet{}
	for _, nodeID := range hiddenNodes {
		r := znet.NodesIPRange[nodeID]
		nonAccessibleIPRanges = append(nonAccessibleIPRanges, r)
		nonAccessibleIPRanges = append(nonAccessibleIPRanges, workloads.WgIP(r))
	}
	if znet.AddWGAccess {
		r := znet.ExternalIP
		nonAccessibleIPRanges = append(nonAccessibleIPRanges, *r)
		nonAccessibleIPRanges = append(nonAccessibleIPRanges, workloads.WgIP(*r))
	}

	log.Debug().Msgf("hidden nodes: %v", hiddenNodes)
	log.Debug().Uint32("public node", znet.PublicNodeID)
	log.Debug().Msgf("accessible nodes: %v", accessibleNodes)
	log.Debug().Msgf("non accessible ip ranges: %v", nonAccessibleIPRanges)

	if znet.AddWGAccess {
		// if no wg private key, it should be generated
		if znet.ExternalSK.String() == workloads.ExternalSKZeroValue {
			wgSK, err := wgtypes.GeneratePrivateKey()
			if err != nil {
				return nil, errors.Wrapf(err, "failed to generate wireguard secret key for network: %s", znet.Name)
			}
			znet.ExternalSK = wgSK
		}

		znet.AccessWGConfig = workloads.GenerateWGConfig(
			workloads.WgIP(*znet.ExternalIP).IP.String(),
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
	metadata := workloads.NetworkMetaData{
		UserAccessIP: externalIP,
		PrivateKey:   znet.ExternalSK.String(),
		PublicNodeID: znet.PublicNodeID,
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
			allowedIPs := []gridtypes.IPNet{
				peerIPRange,
				workloads.WgIP(peerIPRange),
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
					AllowedIPs:  []gridtypes.IPNet{*znet.ExternalIP, workloads.WgIP(*znet.ExternalIP)},
				})
			}

			// hidden nodes
			for _, peerNodeID := range hiddenNodes {
				peerIPRange := znet.NodesIPRange[peerNodeID]
				peers = append(peers, zos.Peer{
					Subnet:      peerIPRange,
					WGPublicKey: znet.Keys[peerNodeID].PublicKey().String(),
					AllowedIPs: []gridtypes.IPNet{
						peerIPRange,
						workloads.WgIP(peerIPRange),
					},
				})
			}
		}

		workload := znet.ZosWorkload(znet.NodesIPRange[nodeID], znet.Keys[nodeID].String(), uint16(znet.WGPort[nodeID]), peers, string(metadataBytes))
		deployment := workloads.NewGridDeployment(d.tfPluginClient.TwinID, []gridtypes.Workload{workload})

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
				AllowedIPs: []gridtypes.IPNet{
					znet.IPRange,
					workloads.IPNet(100, 64, 0, 0, 16),
				},
				Endpoint: fmt.Sprintf("%s:%d", endpoints[znet.PublicNodeID], znet.WGPort[znet.PublicNodeID]),
			})
		}
		workload := znet.ZosWorkload(znet.NodesIPRange[nodeID], znet.Keys[nodeID].String(), uint16(znet.WGPort[nodeID]), peers, string(metadataBytes))
		deployment := workloads.NewGridDeployment(d.tfPluginClient.TwinID, []gridtypes.Workload{workload})

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

// Deploy deploys the network deployments using the deployer
func (d *NetworkDeployer) Deploy(ctx context.Context, znet *workloads.ZNet) error {
	err := d.Validate(ctx, znet)
	if err != nil {
		return err
	}

	newDeployments, err := d.GenerateVersionlessDeployments(ctx, znet, nil)
	if err != nil {
		return errors.Wrap(err, "could not generate deployments data")
	}

	newDeploymentsSolutionProvider := make(map[uint32]*uint64)
	for _, nodeID := range znet.Nodes {
		// solution providers
		newDeploymentsSolutionProvider[nodeID] = nil
	}

	znet.NodeDeploymentID, err = d.deployer.Deploy(ctx, znet.NodeDeploymentID, newDeployments, newDeploymentsSolutionProvider)

	// update deployment and plugin state
	// error is not returned immediately before updating state because of untracked failed deployments
	for _, nodeID := range znet.Nodes {
		if contractID, ok := znet.NodeDeploymentID[nodeID]; ok && contractID != 0 {
			d.tfPluginClient.State.Networks.UpdateNetworkSubnets(znet.Name, znet.NodesIPRange)
			if !workloads.Contains(d.tfPluginClient.State.CurrentNodeDeployments[nodeID], znet.NodeDeploymentID[nodeID]) {
				d.tfPluginClient.State.CurrentNodeDeployments[nodeID] = append(d.tfPluginClient.State.CurrentNodeDeployments[nodeID], znet.NodeDeploymentID[nodeID])
			}
		}
	}

	if err != nil {
		return errors.Wrapf(err, "could not deploy network %s", znet.Name)
	}

	if err := d.ReadNodesConfig(ctx, znet); err != nil {
		return errors.Wrap(err, "could not read node's data")
	}

	return nil
}

// BatchDeploy deploys multiple network deployments using the deployer
func (d *NetworkDeployer) BatchDeploy(ctx context.Context, znets []*workloads.ZNet, updateMetadata ...bool) error {
	newDeployments := make(map[uint32][]gridtypes.Deployment)
	newDeploymentsSolutionProvider := make(map[uint32][]*uint64)
	nodePorts := make(map[uint32][]uint16)
	for _, znet := range znets {
		err := d.Validate(ctx, znet)
		if err != nil {
			return err
		}

		dls, err := d.GenerateVersionlessDeployments(ctx, znet, nodePorts)
		if err != nil {
			return errors.Wrap(err, "could not generate deployments data")
		}

		for nodeID, dl := range dls {
			// solution providers
			newDeploymentsSolutionProvider[nodeID] = nil

			if _, ok := newDeployments[nodeID]; !ok {
				newDeployments[nodeID] = []gridtypes.Deployment{dl}
				continue
			}
			newDeployments[nodeID] = append(newDeployments[nodeID], dl)
		}
	}

	newDls, err := d.deployer.BatchDeploy(ctx, newDeployments, newDeploymentsSolutionProvider)

	update := true
	if len(updateMetadata) != 0 {
		update = updateMetadata[0]
	}

	// update deployment and plugin state
	// error is not returned immediately before updating state because of untracked failed deployments
	for _, znet := range znets {
		if err := d.updateStateFromDeployments(ctx, znet, newDls, update); err != nil {
			return errors.Wrapf(err, "failed to update network '%s' state", znet.Name)
		}
	}

	return err
}

// Cancel cancels all the deployments
func (d *NetworkDeployer) Cancel(ctx context.Context, znet *workloads.ZNet) error {
	err := d.Validate(ctx, znet)
	if err != nil {
		return err
	}

	for nodeID, contractID := range znet.NodeDeploymentID {
		err = d.deployer.Cancel(ctx, contractID)
		if err != nil {
			return errors.Wrapf(err, "could not cancel network %s, contract %d", znet.Name, contractID)
		}
		delete(znet.NodeDeploymentID, nodeID)
		d.tfPluginClient.State.CurrentNodeDeployments[nodeID] = workloads.Delete(d.tfPluginClient.State.CurrentNodeDeployments[nodeID], contractID)
	}

	// delete network from state if all contracts was deleted
	d.tfPluginClient.State.Networks.DeleteNetwork(znet.Name)

	if err := d.ReadNodesConfig(ctx, znet); err != nil {
		return errors.Wrap(err, "could not read node's data")
	}

	return nil
}

func (d *NetworkDeployer) updateStateFromDeployments(ctx context.Context, znet *workloads.ZNet, dls map[uint32][]gridtypes.Deployment, updateMetadata bool) error {
	znet.NodeDeploymentID = map[uint32]uint64{}

	for _, nodeID := range znet.Nodes {
		// assign NodeDeploymentIDs
		for _, dl := range dls[nodeID] {
			dlData, err := workloads.ParseDeploymentData(dl.Metadata)
			if err != nil {
				return errors.Wrapf(err, "could not get deployment %d data", dl.ContractID)
			}

			if dlData.Name == znet.Name {
				znet.NodeDeploymentID[nodeID] = dl.ContractID
			}
		}

		if contractID, ok := znet.NodeDeploymentID[nodeID]; ok && contractID != 0 {
			d.tfPluginClient.State.Networks.UpdateNetworkSubnets(znet.Name, znet.NodesIPRange)
			if !workloads.Contains(d.tfPluginClient.State.CurrentNodeDeployments[nodeID], znet.NodeDeploymentID[nodeID]) {
				d.tfPluginClient.State.CurrentNodeDeployments[nodeID] = append(d.tfPluginClient.State.CurrentNodeDeployments[nodeID], znet.NodeDeploymentID[nodeID])
			}
		}
	}

	if !updateMetadata {
		return nil
	}
	if err := d.ReadNodesConfig(ctx, znet); err != nil {
		return errors.Wrapf(err, "could not read node's data for network %s", znet.Name)
	}

	return nil
}

// InvalidateBrokenAttributes removes outdated attrs and deleted contracts
func (d *NetworkDeployer) InvalidateBrokenAttributes(znet *workloads.ZNet) error {
	for node, contractID := range znet.NodeDeploymentID {
		contract, err := d.tfPluginClient.SubstrateConn.GetContract(contractID)
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
		cl, err := d.tfPluginClient.NcPool.GetNodeClient(d.tfPluginClient.SubstrateConn, znet.PublicNodeID)
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

// ReadNodesConfig reads the configuration of a network
func (d *NetworkDeployer) ReadNodesConfig(ctx context.Context, znet *workloads.ZNet) error {
	keys := make(map[uint32]wgtypes.Key)
	WGPort := make(map[uint32]int)
	nodesIPRange := make(map[uint32]gridtypes.IPNet)
	log.Debug().Msg("reading node config")
	nodeDeployments, err := d.deployer.GetDeployments(ctx, znet.NodeDeploymentID)
	if err != nil {
		return errors.Wrap(err, "failed to get deployment objects")
	}

	WGAccess := false
	for node, dl := range nodeDeployments {
		for _, wl := range dl.Workloads {
			if wl.Type != zos.NetworkType {
				continue
			}
			data, err := wl.WorkloadData()
			if err != nil {
				return errors.Wrap(err, "could not parse workload data")
			}

			d := data.(*zos.Network)
			WGPort[node] = int(d.WGListenPort)
			keys[node], err = wgtypes.ParseKey(d.WGPrivateKey)
			if err != nil {
				return errors.Wrap(err, "could not parse wg private key from workload object")
			}
			nodesIPRange[node] = d.Subnet
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
