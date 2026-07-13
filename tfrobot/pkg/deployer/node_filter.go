package deployer

import (
	"context"
	"errors"

	"github.com/threefoldtech/zos_sdk_go/grid-client/deployer"
	"github.com/threefoldtech/zos_sdk_go/grid-client/zos"
	"github.com/threefoldtech/zos_sdk_go/grid-proxy/pkg/types"
)

func filterNodes(ctx context.Context,
	tfPluginClient deployer.TFPluginClient,
	group NodesGroup,
	excludedNodes []uint64,
	yggOrWgExistsInVms bool,
) (nodesIDs []int, isLight bool, err error) {
	filter := types.NodeFilter{}
	filter.Excluded = excludedNodes

	freeMRU := convertMBToBytes(uint64(group.FreeMRU * 1024))

	filter.Status = []string{"up"}
	filter.TotalCRU = &group.FreeCPU
	filter.FreeMRU = &freeMRU

	if group.FreeSRU > 0 {
		freeSRU := convertGBToBytes(group.FreeSRU)
		filter.FreeSRU = &freeSRU
	}
	if group.FreeHRU > 0 {
		freeHRU := convertGBToBytes(group.FreeHRU)
		filter.FreeHRU = &freeHRU
	}
	if group.Region != "" {
		filter.Region = &group.Region
	}
	if group.Certified {
		certified := "Certified"
		filter.CertificationType = &certified
	}
	if group.PublicIP4 {
		filter.IPv4 = &group.PublicIP4
	}
	if group.PublicIP6 {
		filter.IPv6 = &group.PublicIP6
	}
	if group.Dedicated {
		filter.Dedicated = &group.Dedicated
	}
	freeSSD := []uint64{group.FreeSRU}
	if group.FreeSRU == 0 {
		freeSSD = nil
	}
	freeHDD := []uint64{group.FreeHRU}
	if group.FreeHRU == 0 {
		freeHDD = nil
	}
	if !group.PublicIP4 && !group.PublicIP6 && !yggOrWgExistsInVms {
		isLight = true
		filter.Features = []string{zos.NetworkLightType, zos.ZMachineLightType}
	}

	nodes, err := deployer.FilterNodes(ctx, tfPluginClient, filter, freeSSD, freeHDD, nil, group.NodesCount)
	if isLight && errors.Is(err, deployer.ErrNoNodesMatchesResources) {
		isLight = false
		filter.Features = []string{}
		nodes, err = deployer.FilterNodes(ctx, tfPluginClient, filter, freeSSD, freeHDD, nil, group.NodesCount)
	}
	if err != nil {
		return
	}

	for _, node := range nodes {
		nodesIDs = append(nodesIDs, node.NodeID)
	}

	return
}

func convertGBToBytes(gb uint64) uint64 {
	bytes := gb * 1024 * 1024 * 1024
	return bytes
}

func convertMBToBytes(mb uint64) uint64 {
	bytes := mb * 1024 * 1024
	return bytes
}
