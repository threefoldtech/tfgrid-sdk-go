package deployer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/graphql"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"gopkg.in/yaml.v3"
)

const (
	maxGoroutinesToFetchState = 100
)

func RunLoader(ctx context.Context, cfg Config, debug bool, output string) error {
	log.Info().Msg("Loading deployments")

	tfPluginClient, err := setup(cfg, debug)
	if err != nil {
		return err
	}

	nodeGroupsInfo := map[string][]vmOutput{}
	failedGroups := map[string]string{}

	// load contracts with node group name
	for _, nodeGroup := range cfg.NodeGroups {
		contracts, err := tfPluginClient.ContractsGetter.ListContractsOfProjectName(nodeGroup.Name)
		if err != nil {
			log.Debug().Err(err).
				Str("node group", nodeGroup.Name).
				Msg("couldn't load from grid")

			failedGroups[nodeGroup.Name] = err.Error()
			continue
		}

		info, err := loadNodeGroupInfo(ctx, tfPluginClient, nodeGroup.Name, contracts.NodeContracts)
		if err != nil {
			failedGroups[nodeGroup.Name] = err.Error()
			continue
		}
		nodeGroupsInfo[nodeGroup.Name] = info
	}

	outputBytes, err := ParseDeploymentData(nodeGroupsInfo, failedGroups, output)
	fmt.Println(string(outputBytes))

	return os.WriteFile(output, outputBytes, 0644)
}

func loadNodeGroupInfo(ctx context.Context, tfPluginClient deployer.TFPluginClient, nodeGroup string, contracts []graphql.Contract) ([]vmOutput, error) {
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
			if err != nil {
				log.Debug().Err(err).
					Str("node group", nodeGroup).
					Msg("couldn't parse deployment data")

				lock.Lock()
				multiErr = multierror.Append(multiErr, err)
				lock.Unlock()
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

func ParseDeploymentData(passedGroups map[string][]vmOutput, failedGroups map[string]string, output string) ([]byte, error) {
	outData := struct {
		OK    map[string][]vmOutput `json:"ok"`
		Error map[string]string     `json:"error"`
	}{
		OK:    passedGroups,
		Error: failedGroups,
	}

	var outputBytes []byte
	var err error
	if filepath.Ext(output) == ".json" {
		outputBytes, err = json.MarshalIndent(outData, "", "  ")
	} else {
		outputBytes, err = yaml.Marshal(outData)
	}
	if err != nil {
		return []byte{}, err
	}
	return outputBytes, err
}
