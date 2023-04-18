// Package deployer is grid deployer
package deployer

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	client "github.com/threefoldtech/tfgrid-sdk-go/grid-client/node"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// FilterNodes filters nodes using proxy
func FilterNodes(ctx context.Context, tfPlugin TFPluginClient, options types.NodeFilter) ([]types.Node, error) {
	nodes, _, err := tfPlugin.GridProxyClient.Nodes(options, types.Limit{})
	if err != nil {
		return []types.Node{}, errors.Wrap(err, "could not fetch nodes from the rmb proxy")
	}

	if len(nodes) == 0 {
		return []types.Node{}, errors.Errorf("could not find any node with options: %+v", serializeOptions(options))
	}

	// if no sru needed
	if options.FreeSRU == nil {
		return nodes, nil
	}

	// check pools
	var nodePools []types.Node
	for _, node := range nodes {
		client, err := tfPlugin.NcPool.GetNodeClient(tfPlugin.SubstrateConn, uint32(node.NodeID))
		if err != nil {
			return []types.Node{}, errors.Wrapf(err, "failed to get node '%d' client", node.NodeID)
		}
		pools, err := client.Pools(ctx)
		if err != nil {
			return []types.Node{}, errors.Wrapf(err, "failed to get node '%d' pools", node.NodeID)
		}
		if hasEnoughStorage(pools, *options.FreeSRU) {
			nodePools = append(nodePools, node)
		}
	}

	if len(nodePools) == 0 {
		return []types.Node{}, errors.Errorf("could not find any node with free ssd pools: %d GB", convertBytesToGB(*options.FreeSRU))
	}

	return nodePools, nil
}

var (
	trueVal  = true
	statusUp = "up"
)

// GetPublicNode return public node ID
func GetPublicNode(ctx context.Context, tfPlugin TFPluginClient, preferredNodes []uint32) (uint32, error) {
	preferredNodesSet := make(map[int]struct{})
	for _, node := range preferredNodes {
		preferredNodesSet[int(node)] = struct{}{}
	}

	nodes, err := FilterNodes(ctx, tfPlugin, types.NodeFilter{
		IPv4:   &trueVal,
		Status: &statusUp,
	})
	if err != nil {
		return 0, err
	}

	// force add preferred nodes
	nodeMap := make(map[int]struct{})
	for _, node := range nodes {
		nodeMap[node.NodeID] = struct{}{}
	}

	for _, node := range preferredNodes {
		if _, ok := nodeMap[int(node)]; ok {
			continue
		}
		nodeInfo, err := tfPlugin.GridProxyClient.Node(node)
		if err != nil {
			log.Error().Msgf("failed to get node %d from the grid proxy", node)
			continue
		}
		if nodeInfo.PublicConfig.Ipv4 == "" {
			continue
		}
		if nodeInfo.Status != "up" {
			continue
		}
		nodes = append(nodes, types.Node{
			PublicConfig: nodeInfo.PublicConfig,
		})
	}

	lastPreferred := 0
	for i := range nodes {
		if _, ok := preferredNodesSet[nodes[i].NodeID]; ok {
			nodes[i], nodes[lastPreferred] = nodes[lastPreferred], nodes[i]
			lastPreferred++
		}
	}

	for _, node := range nodes {
		log.Printf("found a node with ipv4 public config: %d %s\n", node.NodeID, node.PublicConfig.Ipv4)
		ip, _, err := net.ParseCIDR(node.PublicConfig.Ipv4)
		if err != nil {
			log.Printf("could not parse public ip %s of node %d: %s", node.PublicConfig.Ipv4, node.NodeID, err.Error())
			continue
		}
		if ip.IsPrivate() {
			log.Printf("public ip %s of node %d is private", node.PublicConfig.Ipv4, node.NodeID)
			continue
		}
		return uint32(node.NodeID), nil
	}

	return 0, errors.New("no nodes with public ipv4")
}

func hasEnoughStorage(pools []client.PoolMetrics, storage uint64) bool {
	for _, pool := range pools {
		if pool.Type != zos.SSDDevice {
			continue
		}
		if pool.Size-pool.Used > gridtypes.Unit(storage) {
			return true
		}
	}
	return false
}

func serializeOptions(options types.NodeFilter) string {
	var filterStringBuilder strings.Builder
	if options.FarmIDs != nil {
		fmt.Fprintf(&filterStringBuilder, "farm ids: %v, ", options.FarmIDs)
	}
	if options.FarmName != nil {
		fmt.Fprintf(&filterStringBuilder, "farm name: %v, ", options.FarmName)
	}
	if options.FreeMRU != nil {
		fmt.Fprintf(&filterStringBuilder, "mru: %d GB, ", convertBytesToGB(*options.FreeMRU))
	}
	if options.FreeSRU != nil {
		fmt.Fprintf(&filterStringBuilder, "sru: %d GB, ", convertBytesToGB(*options.FreeSRU))
	}
	if options.FreeHRU != nil {
		fmt.Fprintf(&filterStringBuilder, "hru: %d GB, ", convertBytesToGB(*options.FreeHRU))
	}
	if options.FreeIPs != nil {
		fmt.Fprintf(&filterStringBuilder, "free ips: %d, ", *options.FreeIPs)
	}
	if options.Domain != nil {
		fmt.Fprintf(&filterStringBuilder, "domain: %t, ", *options.Domain)
	}
	if options.IPv4 != nil {
		fmt.Fprintf(&filterStringBuilder, "ipv4: %t, ", *options.IPv4)
	}
	filterString := filterStringBuilder.String()
	return filterString[:len(filterString)-2]
}

func convertBytesToGB(bytes uint64) uint64 {
	gb := bytes / (1024 * 1024 * 1024)
	return gb
}
