package deployer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog/log"
	"github.com/sethvargo/go-retry"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/graphql"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"gopkg.in/yaml.v3"
)

func RunLoader(ctx context.Context, cfg Config, debug bool, output string) error {
	log.Info().Msg("Loading deployments")

	tfPluginClient, err := setup(cfg, debug)
	if err != nil {
		return err
	}
	var nodeGroups []string
	for _, group := range cfg.NodeGroups {
		nodeGroups = append(nodeGroups, group.Name)
	}

	outputBytes, err := loadNodeGroupsInfo(ctx, tfPluginClient, nodeGroups, cfg.MaxRetries, output)
	if err != nil {
		return err
	}

	fmt.Println(string(outputBytes))
	return os.WriteFile(output, outputBytes, 0644)
}

func loadNodeGroupsInfo(ctx context.Context, tfPluginClient deployer.TFPluginClient, nodegroups []string, maxRetries uint64, output string) ([]byte, error) {
	trial := 1
	nodeGroupsInfo := map[string][]vmOutput{}
	failedGroups := map[string]string{}

	// load contracts with node group name
	for _, nodeGroup := range nodegroups {
		if err := retry.Do(ctx, retry.WithMaxRetries(maxRetries, retry.NewConstant(1*time.Second)), func(ctx context.Context) error {
			if trial != 1 {
				log.Debug().Str("Node group", nodeGroup).Int("Deployment trial", trial).Msg("Retrying to load")
			}

			contracts, err := tfPluginClient.ContractsGetter.ListContractsOfProjectName(nodeGroup)
			if err != nil {
				trial++
				log.Debug().Err(err).Str("node group", nodeGroup).Msg("couldn't list contracts")
				return retry.RetryableError(err)
			}

			info, err := loadContractsInfo(ctx, tfPluginClient, nodeGroup, contracts.NodeContracts)
			if err != nil {
				trial++
				log.Debug().Err(err).Str("node group", nodeGroup).Msg("couldn't load from grid")
				return retry.RetryableError(err)
			}

			nodeGroupsInfo[nodeGroup] = info
			return nil
		}); err != nil {
			failedGroups[nodeGroup] = err.Error()
		}
	}

	return parseDeploymentData(nodeGroupsInfo, failedGroups, output)
}

func loadContractsInfo(ctx context.Context, tfPluginClient deployer.TFPluginClient, nodeGroup string, contracts []graphql.Contract) ([]vmOutput, error) {
	vmsInfo := []vmOutput{}
	var multiErr error
	var lock sync.Mutex
	var wg sync.WaitGroup

	// Create a channel to act as a semaphore with a capacity of maxGoroutinesToFetchState
	sem := make(chan struct{}, maxGoroutinesToFetchState)

	for _, contract := range contracts {
		wg.Add(1)

		// Acquire a slot in the semaphore before starting the goroutine
		sem <- struct{}{}

		go func(contract graphql.Contract) {
			defer wg.Done()
			// Ensure the slot is released as soon as the goroutine completes or errors
			defer func() { <-sem }()

			deployment, err := workloads.ParseDeploymentData(contract.DeploymentData)
			if deployment.Type != "vm" {
				return
			}

			if err != nil {
				log.Debug().Err(err).
					Str("node group", nodeGroup).
					Msg("couldn't parse deployment data")

				lock.Lock()
				multiErr = multierror.Append(multiErr, err)
				lock.Unlock()
				return
			}

			log.Debug().
				Str("vm", deployment.Name).
				Msg("loading vm info from state")

			vmDeployment, err := tfPluginClient.State.LoadDeploymentFromGrid(ctx, contract.NodeID, deployment.Name)
			if err != nil {
				lock.Lock()
				multiErr = multierror.Append(multiErr, err)
				lock.Unlock()

				log.Debug().Err(err).
					Str("vm", deployment.Name).
					Str("deployment", deployment.Name).
					Uint32("node ID", contract.NodeID).
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
		}(contract)
	}
	wg.Wait()

	return vmsInfo, multiErr
}

func parseDeploymentData(passedGroups map[string][]vmOutput, failedGroups map[string]string, output string) ([]byte, error) {
	var err error
	var outputBytes []byte
	outData := struct {
		OK    map[string][]vmOutput `json:"ok"`
		Error map[string]string     `json:"error"`
	}{
		OK:    passedGroups,
		Error: failedGroups,
	}

	if filepath.Ext(output) == ".json" {
		outputBytes, err = json.MarshalIndent(outData, "", "  ")
	} else {
		outputBytes, err = yaml.Marshal(outData)
	}

	return outputBytes, err
}
