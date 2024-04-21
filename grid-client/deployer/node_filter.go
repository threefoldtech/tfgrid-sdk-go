package deployer

import (
	"context"
	"fmt"
	"math"
	"net"
	"sort"
	"strings"
	"sync"

	"github.com/gorilla/schema"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	client "github.com/threefoldtech/tfgrid-sdk-go/grid-client/node"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

const requestedPagesPerIteration = 5

// FilterNodes filters nodes using proxy
func FilterNodes(ctx context.Context, tfPlugin TFPluginClient, options types.NodeFilter, ssdDisks, hddDisks, rootfs []uint64, optionalLimit ...uint64) ([]types.Node, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if options.AvailableFor == nil {
		twinID := uint64(tfPlugin.TwinID)
		options.AvailableFor = &twinID
	}
	options.Healthy = &trueVal

	var nodes []types.Node
	var errs error
	var lock sync.Mutex

	// default values for no limit
	requestedPages := 1
	totalPagesCount := 1
	limit := types.Limit{}
	nodesCount := types.DefaultLimit().Size

	if len(optionalLimit) > 0 {
		var err error
		nodesCount = optionalLimit[0]
		limit = types.Limit{Size: nodesCount, RetCount: true}
		requestedPages = requestedPagesPerIteration
		totalPagesCount, err = getPagesCount(ctx, tfPlugin, options, limit)
		if err != nil {
			return []types.Node{}, err
		}
	}

	for i := 1; i <= totalPagesCount; i += requestedPages {
		nodesOutput := make(chan types.Node)

		var wg sync.WaitGroup
		for j := 0; j < requestedPages; j++ {
			limit.Page = uint64(i + j)
			if limit.Page > uint64(totalPagesCount) {
				break
			}

			wg.Add(1)
			go func(limit types.Limit) {
				defer wg.Done()
				err := getNodes(ctx, tfPlugin, options, ssdDisks, hddDisks, rootfs, limit, nodesOutput)
				if err != nil {
					lock.Lock()
					errs = multierror.Append(err)
					lock.Unlock()
				}
			}(limit)
		}

		go func() {
			wg.Wait()
			close(nodesOutput)
		}()

		for node := range nodesOutput {
			nodes = append(nodes, node)
			if uint64(len(nodes)) == nodesCount {
				return nodes, nil
			}
		}

		if len(optionalLimit) == 0 { // no limit and didn't reach default limit
			if len(nodes) == 0 {
				opts, err := serializeOptions(options)
				if err != nil {
					log.Debug().Err(err).Send()
				}
				return []types.Node{}, fmt.Errorf("could not find enough nodes with options: %s", opts)
			}
			return nodes, nil
		}
	}

	if errs != nil {
		return []types.Node{}, errors.Errorf("could not find enough nodes, found errors: %v", errs)
	}

	opts, err := serializeOptions(options)
	if err != nil {
		log.Debug().Err(err).Send()
	}
	return []types.Node{}, errors.Errorf("could not find enough nodes with options: %s", opts)
}

func getNodes(ctx context.Context, tfPlugin TFPluginClient, options types.NodeFilter, ssdDisks, hddDisks, rootfs []uint64, limit types.Limit, output chan<- types.Node) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	nodes, _, err := tfPlugin.GridProxyClient.Nodes(ctx, options, limit)
	if err != nil {
		return errors.Wrap(err, "could not fetch nodes from the rmb proxy")
	}

	if len(nodes) == 0 {
		return nil
	}

	// if no storage needed
	if options.FreeSRU == nil && options.FreeHRU == nil {
		for _, node := range nodes {
			output <- node
		}
		return nil
	}

	sort.Slice(ssdDisks, func(i, j int) bool {
		return ssdDisks[i] > ssdDisks[j]
	})

	// add rootfs at the end to as zos provisions zmounts first.
	ssdDisks = append(ssdDisks, rootfs...)

	sort.Slice(hddDisks, func(i, j int) bool {
		return hddDisks[i] > hddDisks[j]
	})

	for _, node := range nodes {
		client, err := tfPlugin.NcPool.GetNodeClient(tfPlugin.SubstrateConn, uint32(node.NodeID))
		if err != nil {
			log.Debug().Err(err).Int("node ID", node.NodeID).Msgf("failed to get node client")
			continue
		}

		pools, err := client.Pools(ctx)
		if err != nil {
			log.Debug().Err(err).Int("node ID", node.NodeID).Msg("failed to get node pools")
			continue
		}

		if !hasEnoughStorage(pools, hddDisks, zos.HDDDevice) {
			log.Debug().Err(err).Int("node ID", node.NodeID).Msg("no enough HDDs in node")
			continue
		}

		if !hasEnoughStorage(pools, ssdDisks, zos.SSDDevice) {
			log.Debug().Err(err).Int("node ID", node.NodeID).Msg("no enough SSDs in node")
			continue
		}

		select {
		case output <- node:
		case <-ctx.Done():
			return nil
		}
	}

	return nil
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

	nodes, err := FilterNodes(
		ctx,
		tfPlugin,
		types.NodeFilter{
			IPv4:   &trueVal,
			Status: &statusUp,
		},
		nil,
		nil,
		nil)
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
		nodeInfo, err := tfPlugin.GridProxyClient.Node(ctx, node)
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
		log.Printf("found a node with ipv4 public config: %d %s", node.NodeID, node.PublicConfig.Ipv4)
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

// hasEnoughStorage checks if all deployment storage requirements can be satisfied with node's pools based on given disks order.
func hasEnoughStorage(pools []client.PoolMetrics, storages []uint64, poolType zos.DeviceType) bool {
	if len(storages) == 0 {
		return true
	}

	filteredPools := make([]client.PoolMetrics, 0)
	for _, pool := range pools {
		if pool.Type == poolType {
			filteredPools = append(filteredPools, pool)
		}
	}
	if len(filteredPools) == 0 {
		return false
	}
	for _, storage := range storages {
		sort.Slice(filteredPools, func(i, j int) bool {
			return (filteredPools[i].Size - filteredPools[i].Used) > (filteredPools[j].Size - filteredPools[j].Used)
		})
		// assuming zos provision to the largest pool always
		if filteredPools[0].Size-filteredPools[0].Used < gridtypes.Unit(storage) {
			return false
		}
		filteredPools[0].Used += gridtypes.Unit(storage)
	}
	return true
}

// serializeOptions used to encode a struct of NodeFilter type and convert it to string
// with only non-zero values and drop any field with zero-value
func serializeOptions(options types.NodeFilter) (string, error) {
	params := make(map[string][]string)
	err := schema.NewEncoder().Encode(options, params)
	if err != nil {
		return "", nil
	}

	// convert the map to string with `key: value` format
	//
	// example:
	//
	// map[string][]string{Status: [up]} -> "Status: [up]"
	var sb strings.Builder
	for key, val := range params {
		fmt.Fprintf(&sb, "%s: %v, ", key, val[0])
	}

	filter := sb.String()
	if len(filter) > 2 {
		filter = filter[:len(filter)-2]
	}
	return filter, nil
}

func getPagesCount(ctx context.Context, tfPlugin TFPluginClient, options types.NodeFilter, limit types.Limit) (int, error) {
	_, totalNodesCount, err := tfPlugin.GridProxyClient.Nodes(ctx, options, limit)
	if err != nil {
		return 0, err
	}
	return int(math.Ceil(float64(totalNodesCount) / float64(limit.Size))), nil
}
