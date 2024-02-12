package deployer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog/log"
	"github.com/sethvargo/go-retry"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"gopkg.in/yaml.v3"
)

func RunLoader(ctx context.Context, cfg Config, debug bool, output string) error {
	log.Info().Msg("Loading deployments")

	tfPluginClient, err := setup(cfg, debug)
	if err != nil {
		return err
	}
	asJson := filepath.Ext(output) == ".json"

	groupsDeploymentsInfo, failed := getDeploymentsInfoFromProjectName(ctx, tfPluginClient, cfg.NodeGroups, cfg.MaxRetries)
	passedGroups, failedGroups := getNodeGroupsInfo(ctx, tfPluginClient, groupsDeploymentsInfo, cfg.MaxRetries, asJson)

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

func getNodeGroupsInfo(ctx context.Context, tfPluginClient deployer.TFPluginClient, nodeGroupsDeploymentsInfo map[string][]deploymentInfo, retries uint64, asJson bool) (map[string][]vmOutput, map[string]string) {
	trial := 1
	var lock sync.Mutex
	var wg sync.WaitGroup
	failedGroups := map[string]string{}
	nodeGroupsInfo := map[string][]vmOutput{}

	// load contracts with node group name
	for nodeGroup, info := range nodeGroupsDeploymentsInfo {
		wg.Add(1)
		go func(nodeGroup string, info []deploymentInfo) {
			defer wg.Done()
			if err := retry.Do(ctx, retry.WithMaxRetries(retries, retry.NewConstant(1*time.Nanosecond)), func(ctx context.Context) error {
				if trial != 1 {
					log.Debug().Str("Node group", nodeGroup).Int("Deployment trial", trial).Msg("Retrying to load")
				}

				info, err := loadDeploymentsInfo(ctx, tfPluginClient, nodeGroup, info)
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
		}(nodeGroup, info)
	}

	wg.Wait()
	return nodeGroupsInfo, failedGroups
}

func loadDeploymentsInfo(ctx context.Context, tfPluginClient deployer.TFPluginClient, nodeGroup string, groupInfo []deploymentInfo) ([]vmOutput, error) {
	vmsInfo := []vmOutput{}
	var multiErr error
	var lock sync.Mutex
	var wg sync.WaitGroup

	// Create a channel to act as a semaphore with a capacity of maxGoroutinesToFetchState
	sem := make(chan struct{}, maxGoroutinesToFetchState)

	for _, deployment := range groupInfo {
		wg.Add(1)

		// Acquire a slot in the semaphore before starting the goroutine
		sem <- struct{}{}

		go func(deployment deploymentInfo) {
			defer wg.Done()
			// Ensure the slot is released as soon as the goroutine completes or errors
			defer func() { <-sem }()

			log.Debug().
				Str("vm", deployment.name).
				Msg("loading vm info from state")

			vmDeployment, err := tfPluginClient.State.LoadDeploymentFromGrid(ctx, deployment.nodeID, deployment.name)
			if err != nil {
				lock.Lock()
				multiErr = multierror.Append(multiErr, err)
				lock.Unlock()

				log.Debug().Err(err).
					Str("vm", deployment.name).
					Str("deployment", deployment.name).
					Uint32("node ID", deployment.nodeID).
					Msg("couldn't load from state")
				return
			}

			vm := vmDeployment.Vms[0]
			vmInfo := vmOutput{
				Name:        vm.Name,
				NetworkName: vmDeployment.NetworkName,
				NodeID:      vmDeployment.NodeID,
				ContractID:  vmDeployment.ContractID,
				PublicIP4:   vm.ComputedIP,
				PublicIP6:   vm.ComputedIP6,
				PlanetaryIP: vm.PlanetaryIP,
				IP:          vm.IP,
				Mounts:      vm.Mounts,
			}

			lock.Lock()
			vmsInfo = append(vmsInfo, vmInfo)
			lock.Unlock()
		}(deployment)
	}
	wg.Wait()

	return vmsInfo, multiErr
}

func getDeploymentsInfoFromProjectName(ctx context.Context, tfPluginClient deployer.TFPluginClient, nodegroups []NodesGroup, retries uint64) (map[string][]deploymentInfo, map[string]string) {
	trial := 1
	var lock sync.Mutex
	var wg sync.WaitGroup
	failedGroups := map[string]string{}
	nodeGroupsInfo := map[string][]deploymentInfo{}

	// load contracts with node group name
	for _, nodeGroup := range nodegroups {
		wg.Add(1)
		go func(nodeGroup NodesGroup) {
			defer wg.Done()
			if err := retry.Do(ctx, retry.WithMaxRetries(retries, retry.NewConstant(1*time.Nanosecond)), func(ctx context.Context) error {
				if trial != 1 {
					log.Debug().Str("Node group", nodeGroup.Name).Int("Deployment trial", trial).Msg("Retrying to load")
				}

				// get contracts of node group
				contracts, err := tfPluginClient.ContractsGetter.ListContractsOfProjectName(nodeGroup.Name, true)
				if err != nil || len(contracts.NodeContracts) == 0 {
					trial++
					log.Debug().Err(err).Str("node group", nodeGroup.Name).Msg("couldn't list contracts")

					err = errors.Join(err, fmt.Errorf("couldn't list contracts of node group %s", nodeGroup.Name))
					return retry.RetryableError(err)
				}

				// build node group deployments map
				var deployments []deploymentInfo
				for _, contract := range contracts.NodeContracts {
					contractID, err := strconv.ParseUint(contract.ContractID, 10, 64)
					if err != nil {
						return retry.RetryableError(err)
					}

					tfPluginClient.State.CurrentNodeDeployments[contract.NodeID] = append(tfPluginClient.State.CurrentNodeDeployments[contract.NodeID], contractID)
					deployment, err := workloads.ParseDeploymentData(contract.DeploymentData)
					if deployment.Type != "vm" {
						continue
					}

					if err != nil {
						return retry.RetryableError(err)
					}

					deployments = append(deployments, deploymentInfo{contract.NodeID, deployment.Name})
				}

				lock.Lock()
				nodeGroupsInfo[nodeGroup.Name] = deployments
				lock.Unlock()
				return nil
			}); err != nil {
				lock.Lock()
				failedGroups[nodeGroup.Name] = err.Error()
				lock.Unlock()
			}
		}(nodeGroup)
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
