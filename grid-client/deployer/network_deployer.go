package deployer

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"slices"
	"sync"

	"github.com/hashicorp/go-multierror"
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
func (d *NetworkDeployer) Validate(ctx context.Context, znets []*workloads.ZNet) ([]*workloads.ZNet, error) {
	sub := d.tfPluginClient.SubstrateConn
	var multiErr error

	if err := validateAccountBalanceForExtrinsics(sub, d.tfPluginClient.Identity); err != nil {
		return nil, err
	}
	filteredZNets := make([]*workloads.ZNet, 0)
	for _, znet := range znets {
		if err := znet.Validate(); err != nil {
			multiErr = multierror.Append(multiErr, err)
			continue
		}

		if err := d.InvalidateBrokenAttributes(znet); err != nil {
			multiErr = multierror.Append(multiErr, err)
			continue
		}
		filteredZNets = append(filteredZNets, znet)
	}
	return filteredZNets, multiErr
}

// GenerateVersionlessDeployments generates deployments for network deployer without versions.
func (d *NetworkDeployer) GenerateVersionlessDeployments(ctx context.Context, znets []*workloads.ZNet) (map[uint32][]gridtypes.Deployment, error) {
	deployments := make(map[uint32][]gridtypes.Deployment)
	endpoints := make(map[uint32]net.IP)
	nodeUsedPorts := make(map[uint32][]uint16)
	allNodes := make(map[uint32]struct{})
	var multiErr error

	for _, znet := range znets {
		for _, node := range znet.Nodes {
			allNodes[node] = struct{}{}
		}
		if znet.PublicNodeID != 0 {
			allNodes[znet.PublicNodeID] = struct{}{}
		}
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for nodeID := range allNodes {
		wg.Add(1)
		go func(nodeID uint32) {
			defer wg.Done()
			endpoint, usedPorts, err := d.getNodeEndpointAndPorts(ctx, nodeID)
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

	var publicNode uint32
	var err error
	for nodeID := range allNodes {
		// we check first that there is a public node in all nodes in the networks.
		// that way we can save extra requests to get a new public node
		// and its used ports and endpoints. this public node also might not be used
		// as access node if there is no need for it.
		if endpoints[nodeID] != nil && endpoints[nodeID].To4() != nil {
			publicNode = nodeID
			break
		}
	}

	// if none of the nodes used are public (have ipv4 endpoint) we do
	// an extra check that at least one network needs a public node before
	// fetching a public node. we can use this one node as an access point
	// to all networks that need it.
	if publicNode == 0 && needPublicNode(znets) {
		publicNode, err = GetPublicNode(ctx, *d.tfPluginClient, nil)
		// we don't return immediately here as there might be some
		// networks that don't need a public node so they should continue
		// processing fine
		if err != nil {
			multiErr = multierror.Append(
				multiErr,
				fmt.Errorf("failed to get public node: %w", err),
			)
		}
	}
	// here we check that the we managed to get a public node
	// and that public node wasn't already processed. if we got
	// an error while getting public node, then node id will be 0
	// and we will skip getting its data.
	if _, ok := endpoints[publicNode]; !ok && publicNode != 0 {
		endpoint, usedPorts, err := d.getNodeEndpointAndPorts(ctx, publicNode)
		if err != nil {
			multiErr = multierror.Append(multiErr, err)
		} else {
			endpoints[publicNode] = endpoint
			nodeUsedPorts[publicNode] = usedPorts
		}
	}

	for _, znet := range znets {
		dls, err := d.generateDeployments(znet, endpoints, nodeUsedPorts, publicNode)
		if err != nil {
			multiErr = multierror.Append(multiErr, err)
			continue
		}
		for nodeID, dl := range dls {
			deployments[nodeID] = append(deployments[nodeID], dl)
		}
	}

	return deployments, multiErr
}

func (d *NetworkDeployer) getNodeEndpointAndPorts(ctx context.Context, nodeID uint32) (net.IP, []uint16, error) {
	nodeClient, err := d.tfPluginClient.NcPool.GetNodeClient(d.tfPluginClient.SubstrateConn, nodeID)
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

func needPublicNode(znets []*workloads.ZNet) bool {
	// entering here means all nodes for all networks are either hidden or ipv6 only
	// we need an extra public node in two cases:
	// - the user asked for WireGuard access
	// - there are multiple nodes in the network and none of them have ipv4.
	//   because networks must communicate through ipv4
	for _, znet := range znets {
		if znet.AddWGAccess {
			return true
		}
		if len(znet.Nodes) > 1 {
			return true
		}
	}
	return false
}

func (d *NetworkDeployer) generateDeployments(znet *workloads.ZNet, endpointIPs map[uint32]net.IP, usedPorts map[uint32][]uint16, publicNode uint32) (map[uint32]gridtypes.Deployment, error) {
	deployments := make(map[uint32]gridtypes.Deployment)

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
			if !workloads.Contains(accessibleNodes, znet.PublicNodeID) {
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
		p := uint16(rand.Intn(32768-1024) + 1024)
		for slices.Contains(nodeUsedPorts, p) {
			p = uint16(rand.Intn(32768-1024) + 1024)
		}
		nodeUsedPorts = append(nodeUsedPorts, p)
		usedPorts[nodeID] = nodeUsedPorts
		znet.WGPort[nodeID] = int(p)
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
		Version: workloads.Version,
		UserAccesses: []workloads.UserAccess{
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

		workload := znet.ZosWorkload(znet.NodesIPRange[nodeID], znet.Keys[nodeID].String(), uint16(znet.WGPort[nodeID]), peers, string(metadataBytes), znet.MyceliumKeys[nodeID])
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
		workload := znet.ZosWorkload(znet.NodesIPRange[nodeID], znet.Keys[nodeID].String(), uint16(znet.WGPort[nodeID]), peers, string(metadataBytes), znet.MyceliumKeys[nodeID])
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
	znets, err := d.Validate(ctx, []*workloads.ZNet{znet})
	if err != nil {
		return err
	}

	nodeDeployments, err := d.GenerateVersionlessDeployments(ctx, znets)
	if err != nil {
		return errors.Wrap(err, "could not generate deployments data")
	}
	newDeployments := make(map[uint32]gridtypes.Deployment)
	for node, deployments := range nodeDeployments {
		if len(deployments) != 1 {
			// this should never happen
			log.Debug().Uint32("node id", node).Msgf("got number of deployment %d, should be 1", len(deployments))
			continue
		}
		newDeployments[node] = deployments[0]
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
	var multiErr error
	filteredZNets, err := d.Validate(ctx, znets)
	if err != nil {
		multiErr = multierror.Append(multiErr, err)
	}
	if len(filteredZNets) == 0 {
		return multiErr
	}

	newDeployments, err := d.GenerateVersionlessDeployments(ctx, filteredZNets)
	if err != nil {
		multiErr = multierror.Append(multiErr, err)
	}
	if len(newDeployments) == 0 {
		return multiErr
	}

	newDls, err := d.deployer.BatchDeploy(ctx, newDeployments, make(map[uint32][]*uint64))
	if err != nil {
		multiErr = multierror.Append(multiErr, err)
	}

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

	return multiErr
}

// Cancel cancels all the deployments
func (d *NetworkDeployer) Cancel(ctx context.Context, znet *workloads.ZNet) error {
	err := validateAccountBalanceForExtrinsics(d.tfPluginClient.SubstrateConn, d.tfPluginClient.Identity)
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

// BatchCancel cancels all contracts for given networks. if one contracts failed all networks will not be canceled
// and state won't be updated.
func (d *NetworkDeployer) BatchCancel(ctx context.Context, znets []*workloads.ZNet) error {
	var contracts []uint64
	for _, znet := range znets {
		for _, contractID := range znet.NodeDeploymentID {
			if contractID != 0 {
				contracts = append(contracts, contractID)
			}
		}
	}
	err := d.tfPluginClient.BatchCancelContract(contracts)
	if err != nil {
		return fmt.Errorf("failed to cancel contracts: %w", err)
	}
	for _, znet := range znets {
		for nodeID, contractID := range znet.NodeDeploymentID {
			d.tfPluginClient.State.CurrentNodeDeployments[nodeID] = workloads.Delete(d.tfPluginClient.State.CurrentNodeDeployments[nodeID], contractID)
		}

		d.tfPluginClient.State.Networks.DeleteNetwork(znet.Name)
		znet.NodeDeploymentID = make(map[uint32]uint64)
		znet.Keys = make(map[uint32]wgtypes.Key)
		znet.WGPort = make(map[uint32]int)
		znet.NodesIPRange = make(map[uint32]gridtypes.IPNet)
		znet.AccessWGConfig = ""
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
			if dlData.Name == znet.Name && dl.ContractID != 0 {
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
