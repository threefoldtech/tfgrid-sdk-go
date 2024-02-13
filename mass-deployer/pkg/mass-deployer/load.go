package deployer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pkg/errors"

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

	groupsDeploymentsInfo, failed := getContractsOfNodeGroups(ctx, tfPluginClient, cfg.NodeGroups)
	passedGroups, failedGroups := batchLoadNodeGroupsInfo(ctx, tfPluginClient, groupsDeploymentsInfo, cfg.MaxRetries, asJson)

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

func batchLoadNodeGroupsInfo(ctx context.Context, tfPluginClient deployer.TFPluginClient, nodeGroupsDeploymentsInfo map[string][]contractsInfo, retries uint64, asJson bool) (map[string][]vmOutput, map[string]string) {
	trial := 1
	failedGroups := map[string]string{}
	nodeGroupsInfo := map[string][]vmOutput{}

	// load contracts with node group name
	for nodeGroup, contractsInfo := range nodeGroupsDeploymentsInfo {
		if err := retry.Do(ctx, retry.WithMaxRetries(retries, retry.NewConstant(1*time.Nanosecond)), func(ctx context.Context) error {
			if trial != 1 {
				log.Debug().Str("Node group", nodeGroup).Int("Deployment trial", trial).Msg("Retrying to load")
			}

			info, err := batchLoadDeployments(ctx, tfPluginClient, contractsInfo)
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

	return nodeGroupsInfo, failedGroups
}

// batch load deployment info with contractID and nodeID
func batchLoadDeployments(ctx context.Context, tfPluginClient deployer.TFPluginClient, contracts []contractsInfo) ([]vmOutput, error) {
	st := tfPluginClient.State

	var multiErr error
	var vmsInfo []vmOutput

	for _, contract := range contracts {
		log.Debug().
			Uint64("contract ID", contract.contractID).
			Uint32("node ID", contract.nodeID).
			Msg("loading vm info")

		nodeClient, err := st.NcPool.GetNodeClient(st.Substrate, contract.nodeID)
		if err != nil {
			log.Debug().Err(err).
				Uint64("contract ID", contract.contractID).
				Uint32("node ID", contract.nodeID).
				Msg("couldn't load ")
			multiErr = multierror.Append(multiErr, errors.Wrapf(err, "could not get node client: %d", contract.nodeID))
			continue
		}

		dl, err := nodeClient.DeploymentGet(ctx, contract.contractID)
		if err != nil {
			log.Debug().Err(err).
				Uint64("contract ID", contract.contractID).
				Uint32("node ID", contract.nodeID).
				Msg("couldn't load ")
			multiErr = multierror.Append(multiErr, errors.Wrapf(err, "could not get network deployment %d from node %d", contract.contractID, contract.nodeID))
			continue
		}

		vmDeployment, err := workloads.NewDeploymentFromZosDeployment(dl, contract.nodeID)
		if err != nil {
			log.Debug().Err(err).
				Uint64("contract ID", contract.contractID).
				Uint32("node ID", contract.nodeID).
				Msg("couldn't load ")
			multiErr = multierror.Append(multiErr, err)
			continue
		}

		for _, vm := range vmDeployment.Vms {
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
			vmsInfo = append(vmsInfo, vmInfo)
		}
	}
	return vmsInfo, multiErr
}

func getContractsOfNodeGroups(ctx context.Context, tfPluginClient deployer.TFPluginClient, nodeGroups []NodesGroup) (map[string][]contractsInfo, map[string]string) {
	loadedContracts := map[string][]contractsInfo{}
	failedGroups := map[string]string{}

	// load contracts with node group name
	for _, nodeGroup := range nodeGroups {
		nodeGroup := nodeGroup.Name

		contracts, err := tfPluginClient.ContractsGetter.ListContractsOfProjectName(nodeGroup, true)
		if err != nil || len(contracts.NodeContracts) == 0 {
			log.Debug().Err(err).Str("node group", nodeGroup).Msg("couldn't list contracts")
			failedGroups[nodeGroup] = errors.Wrapf(err, "couldn't list contracts of node group %s", nodeGroup).Error()
			continue
		}

		// build contracts info struct
		var multiErr error
		var deployments []contractsInfo
		for _, contract := range contracts.NodeContracts {
			contractID, err := strconv.ParseUint(contract.ContractID, 10, 64)
			if err != nil {
				log.Debug().Err(err).Str("node group", nodeGroup).Msg("couldn't parse contract id")
				multiErr = multierror.Append(multiErr, errors.Wrapf(err, "couldn't parse contract id of node group %s", nodeGroup))
				continue
			}

			deployments = append(deployments, contractsInfo{contract.NodeID, contractID})
			tfPluginClient.State.CurrentNodeDeployments[contract.NodeID] = append(tfPluginClient.State.CurrentNodeDeployments[contract.NodeID], contractID)
		}

		if multiErr != nil {
			failedGroups[nodeGroup] = multiErr.Error()
			continue
		}
		loadedContracts[nodeGroup] = deployments
	}

	return loadedContracts, failedGroups
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
