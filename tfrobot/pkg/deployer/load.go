package deployer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog/log"
	"github.com/sethvargo/go-retry"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/graphql"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"gopkg.in/yaml.v3"
)

type NodeContracts map[uint32][]uint64

func RunLoader(ctx context.Context, cfg Config, tfPluginClient deployer.TFPluginClient, debug bool, output string) *multierror.Error {
	log.Info().Msg("Loading deployments")

	asJson := filepath.Ext(output) == ".json"

	groupsContracts, failedGroupsErr := getGroupsContracts(tfPluginClient, cfg.NodeGroups)
	passedGroups, errs := batchLoadNodeGroupsInfo(ctx, tfPluginClient, groupsContracts, cfg.MaxRetries)

	// add projects failed to be loaded
	if errs != nil {
		failedGroupsErr = multierror.Append(failedGroupsErr, errs.Errors...)
	}

	outputBytes, err := parseDeploymentOutput(passedGroups, asJson)
	if err != nil {
		failedGroupsErr = multierror.Append(failedGroupsErr, err)
	}

	fmt.Println(string(outputBytes))

	err = os.WriteFile(output, outputBytes, 0644)
	if err != nil {
		log.Info().Err(err).Send()
	}

	return failedGroupsErr
}

func getGroupsContracts(tfPluginClient deployer.TFPluginClient, nodeGroups []NodesGroup) (map[string]NodeContracts, *multierror.Error) {
	loadedContracts := make(map[string]NodeContracts)
	var failedGroupsErr *multierror.Error

	var lock sync.Mutex
	var wg sync.WaitGroup

	// load contracts with node group name
	for _, nodeGroup := range nodeGroups {
		wg.Add(1)

		go func(nodeGroup string) {
			defer wg.Done()

			nodesContracts, err := getContractsWithProjectName(tfPluginClient, nodeGroup)

			lock.Lock()
			defer lock.Unlock()

			if err != nil {
				failedGroupsErr = multierror.Append(failedGroupsErr, fmt.Errorf("%s: %s", nodeGroup, err.Error()))
				return
			}

			loadedContracts[nodeGroup] = nodesContracts
		}(nodeGroup.Name)
	}

	wg.Wait()
	return loadedContracts, failedGroupsErr
}

func batchLoadNodeGroupsInfo(
	ctx context.Context,
	tfPluginClient deployer.TFPluginClient,
	groupsContracts map[string]NodeContracts,
	retries uint64,
) (map[string][]vmOutput, *multierror.Error) {
	trial := 1
	nodeGroupsInfo := map[string][]vmOutput{}
	var failedGroupsErr *multierror.Error

	var lock sync.Mutex
	var wg sync.WaitGroup

	// load contracts with node group name
	for nodeGroup, contracts := range groupsContracts {
		wg.Add(1)

		go func(nodeGroup string, nodesContracts NodeContracts) {
			defer wg.Done()
			if err := retry.Do(ctx, retry.WithMaxRetries(retries, retry.NewConstant(1*time.Nanosecond)), func(ctx context.Context) error {
				if trial != 1 {
					log.Debug().Str("Node group", nodeGroup).Int("Load trial", trial).Msg("Retrying to load")
				}

				info, err := loadGroupDeployments(ctx, tfPluginClient, nodesContracts)
				if err != nil {
					trial++
					log.Debug().Err(err).Str("node group", nodeGroup).Msg("couldn't load from grid")
					return retry.RetryableError(err)
				}

				lock.Lock()
				nodeGroupsInfo[nodeGroup] = info
				lock.Unlock()

				return nil
			}); err != nil {
				lock.Lock()
				failedGroupsErr = multierror.Append(failedGroupsErr, fmt.Errorf("%s: %s", nodeGroup, err.Error()))
				lock.Unlock()
			}
		}(nodeGroup, contracts)
	}

	wg.Wait()
	return nodeGroupsInfo, failedGroupsErr
}

func parseDeploymentOutput(passedGroups map[string][]vmOutput, asJson bool) ([]byte, error) {
	var err error
	var outputBytes []byte
	outData := struct {
		OK map[string][]vmOutput
	}{
		OK: passedGroups,
	}

	if asJson {
		outputBytes, err = json.MarshalIndent(outData, "", "  ")
	} else {
		outputBytes, err = yaml.Marshal(outData)
	}

	return outputBytes, err
}

func loadGroupDeployments(ctx context.Context, tfPluginClient deployer.TFPluginClient, contracts NodeContracts) ([]vmOutput, error) {
	var lock sync.Mutex
	var vmsInfo []vmOutput

	errGroup := new(errgroup.Group)
	errGroup.SetLimit(maxGoroutinesCount)

	for nodeID, contractIDs := range contracts {
		nodeID := nodeID
		contractIDs := contractIDs

		errGroup.Go(func() error {
			deployments, err := loadNodeDeployments(ctx, tfPluginClient, nodeID, contractIDs)
			if err != nil {
				return err
			}

			lock.Lock()
			defer lock.Unlock()

			for _, deployment := range deployments {
				for _, vm := range deployment.Vms {
					vmInfo := vmOutput{
						Name:        vm.Name,
						NetworkName: deployment.NetworkName,
						NodeID:      deployment.NodeID,
						ContractID:  deployment.ContractID,
						PublicIP4:   vm.ComputedIP,
						PublicIP6:   vm.ComputedIP6,
						YggIP:       vm.PlanetaryIP,
						MyceliumIP:  vm.MyceliumIP,
						IP:          vm.IP,
						Mounts:      vm.Mounts,
					}
					vmsInfo = append(vmsInfo, vmInfo)
				}
				for _, vm := range deployment.VmsLight {
					vmInfo := vmOutput{
						Name:        vm.Name,
						NetworkName: deployment.NetworkName,
						NodeID:      deployment.NodeID,
						ContractID:  deployment.ContractID,
						MyceliumIP:  vm.MyceliumIP,
						IP:          vm.IP,
						Mounts:      vm.Mounts,
					}
					vmsInfo = append(vmsInfo, vmInfo)
				}
			}
			return nil
		})
	}

	err := errGroup.Wait()
	return vmsInfo, err
}

func loadNodeDeployments(ctx context.Context, tfPluginClient deployer.TFPluginClient, nodeID uint32, contractIDs []uint64) ([]workloads.Deployment, error) {
	nodeClient, err := tfPluginClient.State.NcPool.GetNodeClient(tfPluginClient.State.Substrate, nodeID)
	if err != nil {
		return []workloads.Deployment{}, fmt.Errorf("could not get node client %d: %w", nodeID, err)
	}

	dls, err := nodeClient.DeploymentList(ctx)
	if err != nil {
		return []workloads.Deployment{}, fmt.Errorf("could not list deployments of node %d: %w", nodeID, err)
	}

	var deployments []workloads.Deployment
	for _, dl := range dls {

		if !slices.Contains(contractIDs, dl.ContractID) {
			continue
		}

		log.Debug().Uint64("contract ID", dl.ContractID).Msg("loading deployment ")

		deployment, err := workloads.NewDeploymentFromZosDeployment(dl, nodeID)
		if err != nil {
			log.Debug().Err(err).Uint64("contract ID", dl.ContractID).Msg("couldn't load ")
			return nil, err
		}

		if len(deployment.Vms) != 0 || len(deployment.VmsLight) != 0 {
			deployments = append(deployments, deployment)
		}

	}

	return deployments, nil
}

func getContractsWithProjectName(tfPluginClient deployer.TFPluginClient, nodeGroup string) (NodeContracts, error) {
	// try to load group with the new name format "vm/<group name>"
	name := fmt.Sprintf("vm/%s", nodeGroup)

	contracts, err := tfPluginClient.ContractsGetter.ListContractsOfProjectName(name, true)
	if err != nil {
		log.Debug().Err(err).Str("node group", nodeGroup).Msg("couldn't list contracts")
		return nil, fmt.Errorf("couldn't list contracts of node group %s: %w", nodeGroup, err)
	}

	if len(contracts.NodeContracts) == 0 {
		// if load failed try to load group with the old name format "<group name>"
		contracts, err = tfPluginClient.ContractsGetter.ListContractsOfProjectName(nodeGroup, true)
		if err != nil {
			log.Debug().Err(err).Str("node group", nodeGroup).Msg("couldn't list contracts")
			return nil, fmt.Errorf("couldn't list contracts of node group %s: %w", nodeGroup, err)
		}

		if len(contracts.NodeContracts) == 0 {
			log.Debug().Err(err).Str("node group", nodeGroup).Msg("couldn't find any contracts")
			return nil, fmt.Errorf("couldn't find any contracts of node group %s", nodeGroup)
		}
	}

	nodesContracts, err := getNodesContracts(contracts.NodeContracts)
	if err != nil {
		log.Debug().Err(err).Str("node group", nodeGroup).Msg("couldn't parse contract id")

		return nil, fmt.Errorf("couldn't parse contract id of node group %s: %w", nodeGroup, err)
	}
	return nodesContracts, nil
}

func getNodesContracts(contracts []graphql.Contract) (NodeContracts, error) {
	nodeContracts := make(NodeContracts)

	for _, contract := range contracts {
		contractID, err := strconv.ParseUint(contract.ContractID, 10, 64)
		if err != nil {
			return NodeContracts{}, err
		}

		nodeContracts[contract.NodeID] = append(nodeContracts[contract.NodeID], contractID)
	}

	return nodeContracts, nil
}
