package deployer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/rs/zerolog/log"
	"github.com/sethvargo/go-retry"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/graphql"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"gopkg.in/yaml.v3"
)

func RunLoader(ctx context.Context, cfg Config, tfPluginClient deployer.TFPluginClient, debug bool, output string) error {
	log.Info().Msg("Loading deployments")

	asJson := filepath.Ext(output) == ".json"

	groupsContractIDs, failed := getContractsOfNodeGroups(ctx, tfPluginClient, cfg.NodeGroups)
	passedGroups, failedGroups := batchLoadNodeGroupsInfo(ctx, tfPluginClient, groupsContractIDs, cfg.MaxRetries, asJson)

	// add projects failed to be loaded
	for group, err := range failed {
		failedGroups[group] = err
	}

	outputBytes, err := parseDeploymentOutput(passedGroups, failedGroups, asJson)
	if err != nil {
		return err
	}

	fmt.Println(string(outputBytes))
	return os.WriteFile(output, outputBytes, 0644)
}

func getContractsOfNodeGroups(ctx context.Context, tfPluginClient deployer.TFPluginClient, nodeGroups []NodesGroup) (map[string][]uint64, map[string]string) {
	loadedContracts := map[string][]uint64{}
	failedGroups := map[string]string{}

	var lock sync.Mutex
	var wg sync.WaitGroup

	// load contracts with node group name
	for _, nodeGroup := range nodeGroups {
		wg.Add(1)

		go func(nodeGroup string) {
			defer wg.Done()

			ContractIDs, err := getContractsWithProjectName(ctx, tfPluginClient, nodeGroup)
			if err != nil {
				lock.Lock()
				failedGroups[nodeGroup] = err.Error()
				lock.Unlock()
				return
			}
			lock.Lock()

			loadedContracts[nodeGroup] = ContractIDs
			lock.Unlock()
		}(nodeGroup.Name)
	}

	wg.Wait()
	return loadedContracts, failedGroups
}

func getContractIDs(contracts []graphql.Contract) ([]uint64, error) {
	var ContractIDs []uint64
	for _, contract := range contracts {
		contractID, err := strconv.ParseUint(contract.ContractID, 10, 64)
		if err != nil {
			return []uint64{}, err
		}

		ContractIDs = append(ContractIDs, contractID)
	}
	return ContractIDs, nil
}

func batchLoadNodeGroupsInfo(
	ctx context.Context,
	tfPluginClient deployer.TFPluginClient,
	groupsContracts map[string][]uint64,
	retries uint64,
	asJson bool,
) (map[string][]vmOutput, map[string]string) {
	trial := 1
	failedGroups := map[string]string{}
	nodeGroupsInfo := map[string][]vmOutput{}

	var lock sync.Mutex
	var wg sync.WaitGroup

	// load contracts with node group name
	for nodeGroup, contracts := range groupsContracts {
		wg.Add(1)

		go func(nodeGroup string, contracts []uint64) {
			defer wg.Done()
			if err := retry.Do(ctx, retry.WithMaxRetries(retries, retry.NewConstant(1*time.Nanosecond)), func(ctx context.Context) error {
				if trial != 1 {
					log.Debug().Str("Node group", nodeGroup).Int("Load trial", trial).Msg("Retrying to load")
				}

				info, err := batchLoadDeployments(ctx, tfPluginClient, contracts)
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
				failedGroups[nodeGroup] = err.Error()
				lock.Unlock()
			}
		}(nodeGroup, contracts)
	}

	wg.Wait()
	return nodeGroupsInfo, failedGroups
}

func parseDeploymentOutput(passedGroups map[string][]vmOutput, failedGroups map[string]string, asJson bool) ([]byte, error) {
	var err error
	var outputBytes []byte
	outData := struct {
		OK    map[string][]vmOutput `json:"ok"`
		Error map[string]string     `json:"error"`
	}{
		OK:    passedGroups,
		Error: failedGroups,
	}

	if asJson {
		outputBytes, err = json.MarshalIndent(outData, "", "  ")
	} else {
		outputBytes, err = yaml.Marshal(outData)
	}

	return outputBytes, err
}

// batch load deployment info with contractID and nodeID
func batchLoadDeployments(ctx context.Context, tfPluginClient deployer.TFPluginClient, contracts []uint64) ([]vmOutput, error) {
	var lock sync.Mutex
	var vmsInfo []vmOutput

	errGroup := new(errgroup.Group)
	errGroup.SetLimit(maxGoroutinesCount)

	for _, contract := range contracts {
		contractID := contract

		errGroup.Go(func() error {
			log.Debug().
				Uint64("contract ID", contractID).
				Msg("loading deployment")

			deployment, err := loadDeploymentWithContractID(ctx, tfPluginClient, contractID)
			if err != nil {
				log.Debug().Err(err).
					Uint64("contract ID", contractID).
					Msg("couldn't load ")

				return fmt.Errorf("could load deployment %d: %w", contractID, err)
			}

			lock.Lock()
			defer lock.Unlock()

			for _, vm := range deployment.Vms {
				vmInfo := vmOutput{
					Name:        vm.Name,
					NetworkName: deployment.NetworkName,
					NodeID:      deployment.NodeID,
					ContractID:  deployment.ContractID,
					PublicIP4:   vm.ComputedIP,
					PublicIP6:   vm.ComputedIP6,
					PlanetaryIP: vm.PlanetaryIP,
					IP:          vm.IP,
					Mounts:      vm.Mounts,
				}
				vmsInfo = append(vmsInfo, vmInfo)
			}
			return nil
		})
	}
	err := errGroup.Wait()
	return vmsInfo, err
}

func loadDeploymentWithContractID(ctx context.Context, tfPluginClient deployer.TFPluginClient, contractID uint64) (workloads.Deployment, error) {
	st := tfPluginClient.State

	contract, err := tfPluginClient.SubstrateConn.GetContract(contractID)
	if err != nil {
		return workloads.Deployment{}, fmt.Errorf("could not get contract info %d: %w", contractID, err)
	}
	nodeID := uint32(contract.ContractType.NodeContract.Node)

	nodeClient, err := st.NcPool.GetNodeClient(st.Substrate, nodeID)
	if err != nil {
		return workloads.Deployment{}, fmt.Errorf("could not get node client %d: %w", nodeID, err)
	}

	dl, err := nodeClient.DeploymentGet(ctx, contractID)
	if err != nil {
		return workloads.Deployment{}, fmt.Errorf("could not get network deployment %d from node %d: %w", contractID, nodeID, err)
	}

	return workloads.NewDeploymentFromZosDeployment(dl, nodeID)
}

func getContractsWithProjectName(ctx context.Context, tfPluginClient deployer.TFPluginClient, nodeGroup string) ([]uint64, error) {
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

	ContractIDs, err := getContractIDs(contracts.NodeContracts)
	if err != nil {
		log.Debug().Err(err).Str("node group", nodeGroup).Msg("couldn't parse contract id")

		return nil, fmt.Errorf("couldn't parse contract id of node group %s: %w", nodeGroup, err)
	}
	return ContractIDs, nil
}
