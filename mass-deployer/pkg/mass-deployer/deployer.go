package deployer

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/sethvargo/go-retry"
	"gopkg.in/yaml.v3"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

const maxDeploymentRetries = 5

func RunDeployer(ctx context.Context, cfg Config, output string, debug bool) error {
	passedGroups := map[string][]vmOutput{}
	failedGroups := map[string]string{}

	tfPluginClient, err := setup(cfg, debug)
	if err != nil {
		return fmt.Errorf("failed to create deployer: %v", err)
	}

	deploymentStart := time.Now()

	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = maxDeploymentRetries
	}

	for _, nodeGroup := range cfg.NodeGroups {
		log.Info().Str("Node group", nodeGroup.Name).Msg("Running deployment")
		var groupDeployments groupDeploymentsInfo
		trial := 1

		if err := retry.Do(ctx, retry.WithMaxRetries(cfg.MaxRetries, retry.NewConstant(1*time.Nanosecond)), func(ctx context.Context) error {
			if trial != 1 {
				log.Info().Str("Node group", nodeGroup.Name).Int("Deployment trial", trial).Msg("Retrying to deploy")
			}

			info, err := deployNodeGroup(ctx, tfPluginClient, &groupDeployments, nodeGroup, cfg.Vms, cfg.SSHKeys)
			if err != nil {
				trial++
				log.Debug().Err(err).Str("Node group", nodeGroup.Name).Msg("failed to deploy")
				return retry.RetryableError(err)
			}

			log.Info().Str("Node group", nodeGroup.Name).Msg("Done deploying")
			passedGroups[nodeGroup.Name] = info
			return nil
		}); err != nil {

			failedGroups[nodeGroup.Name] = err.Error()
			err := tfPluginClient.CancelByProjectName(nodeGroup.Name)
			if err != nil {
				log.Debug().Err(err).Send()
			}
		}
	}

	log.Info().Msgf("Deployment took %s", time.Since(deploymentStart))

	outData := struct {
		OK    map[string][]vmOutput `json:"ok"`
		Error map[string]string     `json:"error"`
	}{
		OK:    passedGroups,
		Error: failedGroups,
	}

	var outputBytes []byte
	if filepath.Ext(output) == ".json" {
		outputBytes, err = json.MarshalIndent(outData, "", "  ")
	} else {
		outputBytes, err = yaml.Marshal(outData)
	}
	if err != nil {
		return err
	}

	fmt.Println(string(outputBytes))

	return os.WriteFile(output, outputBytes, 0644)
}

func deployNodeGroup(ctx context.Context, tfPluginClient deployer.TFPluginClient, groupDeployments *groupDeploymentsInfo, nodeGroup NodesGroup, vms []Vms, sshKeys map[string]string) ([]vmOutput, error) {
	log.Info().Str("Node group", nodeGroup.Name).Msg("Filter nodes")
	nodesIDs, err := filterNodes(ctx, tfPluginClient, nodeGroup)
	if err != nil {
		return nil, err
	}
	log.Debug().Ints("nodes IDs", nodesIDs).Send()

	if groupDeployments.networkDeployments == nil {
		log.Debug().Str("Node group", nodeGroup.Name).Msg("Parsing vms group")
		*groupDeployments = parseVMsGroup(vms, nodeGroup.Name, nodesIDs, sshKeys)
	} else {
		log.Debug().Str("Node group", nodeGroup.Name).Msg("Updating vms group")
		updateFailedDeployments(ctx, tfPluginClient, nodesIDs, groupDeployments)
	}

	log.Info().Str("Node group", nodeGroup.Name).Msg("Starting mass deployment")
	err = massDeploy(ctx, tfPluginClient, groupDeployments)
	if err != nil {
		return nil, err
	}

	log.Debug().Msg("Load deployments")
	vmsInfo, err := loadGroupInfo(tfPluginClient, groupDeployments.vmDeployments)

	return vmsInfo, err
}

func parseVMsGroup(vms []Vms, nodeGroup string, nodesIDs []int, sshKeys map[string]string) groupDeploymentsInfo {
	vmsOfNodeGroup := []Vms{}
	for _, vm := range vms {
		if vm.NodeGroup == nodeGroup {
			vmsOfNodeGroup = append(vmsOfNodeGroup, vm)
		}
	}

	log.Debug().Str("Node group", nodeGroup).Msg("Build deployments")
	return buildDeployments(vmsOfNodeGroup, nodeGroup, nodesIDs, sshKeys)
}

func massDeploy(ctx context.Context, tfPluginClient deployer.TFPluginClient, deployments *groupDeploymentsInfo) error {
	// deploy only contracts that need to be deployed
	networks, vms := getNotDeployedDeployments(tfPluginClient, deployments)
	var multiErr error

	log.Debug().Msg(fmt.Sprintf("Deploying %d networks, this may to take a while", len(deployments.networkDeployments)))
	if err := tfPluginClient.NetworkDeployer.BatchDeploy(ctx, networks); err != nil {
		multiErr = multierror.Append(multiErr, err)
	}

	log.Debug().Msg(fmt.Sprintf("Deploying %d virtual machines, this may to take a while", len(deployments.vmDeployments)))
	if err := tfPluginClient.DeploymentDeployer.BatchDeploy(ctx, vms); err != nil {
		multiErr = multierror.Append(multiErr, err)
	}

	return multiErr
}

func buildDeployments(vms []Vms, nodeGroup string, nodesIDs []int, sshKeys map[string]string) groupDeploymentsInfo {
	var vmDeployments []*workloads.Deployment
	var networkDeployments []*workloads.ZNet
	var nodesIDsIdx int

	// here we loop over all groups of vms within the same node group, and for every group
	// we loop over all it's vms and create network and vm deployment for it
	// the nodesIDsIdx is a counter used to get nodeID to be able to distribute load over all nodes
	for _, vmGroup := range vms {
		envVars := vmGroup.EnvVars
		if envVars == nil {
			envVars = map[string]string{}
		}
		envVars["SSH_KEY"] = sshKeys[vmGroup.SSHKey]

		for i := 0; i < int(vmGroup.Count); i++ {
			nodeID := uint32(nodesIDs[nodesIDsIdx])
			nodesIDsIdx = (nodesIDsIdx + 1) % len(nodesIDs)

			vmName := fmt.Sprintf("%s%d", vmGroup.Name, i)
			disks, mounts := parseDisks(vmName, vmGroup.SSDDisks)

			network := workloads.ZNet{
				Name:        fmt.Sprintf("%s_network", vmName),
				Description: "network for mass deployment",
				Nodes:       []uint32{nodeID},
				IPRange: gridtypes.NewIPNet(net.IPNet{
					IP:   net.IPv4(10, 20, 0, 0),
					Mask: net.CIDRMask(16, 32),
				}),
				AddWGAccess:  false,
				SolutionType: nodeGroup,
			}

			if !vmGroup.PublicIP4 && !vmGroup.Planetary {
				log.Warn().Str("vms group", vmGroup.Name).Msg("Planetary and public IP options are false. Setting planetary IP to true")
				vmGroup.Planetary = true
			}

			vm := workloads.VM{
				Name:        vmName,
				NetworkName: network.Name,
				Flist:       vmGroup.Flist,
				CPU:         int(vmGroup.FreeCPU),
				Memory:      int(vmGroup.FreeMRU * 1024), // Memory is in MB
				PublicIP:    vmGroup.PublicIP4,
				PublicIP6:   vmGroup.PublicIP6,
				Planetary:   vmGroup.Planetary,
				RootfsSize:  int(vmGroup.RootSize * 1024), // RootSize is in MB
				Entrypoint:  vmGroup.Entrypoint,
				EnvVars:     envVars,
				Mounts:      mounts,
			}
			deployment := workloads.NewDeployment(vm.Name, nodeID, nodeGroup, nil, network.Name, disks, nil, []workloads.VM{vm}, nil)

			vmDeployments = append(vmDeployments, &deployment)
			networkDeployments = append(networkDeployments, &network)
		}
	}
	return groupDeploymentsInfo{vmDeployments: vmDeployments, networkDeployments: networkDeployments}
}

func getNotDeployedDeployments(tfPluginClient deployer.TFPluginClient, groupDeployments *groupDeploymentsInfo) ([]*workloads.ZNet, []*workloads.Deployment) {
	var failedVmDeployments []*workloads.Deployment
	var failedNetworkDeployments []*workloads.ZNet

	for i := range groupDeployments.networkDeployments {
		if isFailedNetwork(*groupDeployments.networkDeployments[i]) {
			failedNetworkDeployments = append(failedNetworkDeployments, groupDeployments.networkDeployments[i])
		}

		if groupDeployments.vmDeployments[i].ContractID == 0 {
			failedVmDeployments = append(failedVmDeployments, groupDeployments.vmDeployments[i])
		}

	}

	return failedNetworkDeployments, failedVmDeployments
}

func loadGroupInfo(tfPluginClient deployer.TFPluginClient, vmDeployments []*workloads.Deployment) ([]vmOutput, error) {
	vmsInfo := []vmOutput{}
	var multiErr error
	var lock sync.Mutex
	var wg sync.WaitGroup

	for _, deployment := range vmDeployments {
		wg.Add(1)

		go func(deployment workloads.Deployment) {
			defer wg.Done()
			log.Debug().
				Str("vm", deployment.Name).
				Msg("loading vm info from state")

			vmDeployment, err := tfPluginClient.State.LoadDeploymentFromGrid(deployment.NodeID, deployment.Name)
			if err != nil {
				lock.Lock()
				defer lock.Unlock()
				multiErr = multierror.Append(multiErr, err)
				log.Debug().Err(err).
					Str("vm", deployment.Vms[0].Name).
					Str("deployment", deployment.Name).
					Uint32("node ID", deployment.NodeID).
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
			defer lock.Unlock()
			vmsInfo = append(vmsInfo, vmInfo)
		}(*deployment)
	}
	wg.Wait()

	return vmsInfo, multiErr
}

func parseDisks(name string, disks []Disk) (disksWorkloads []workloads.Disk, mountsWorkloads []workloads.Mount) {
	for i, disk := range disks {
		DiskWorkload := workloads.Disk{
			Name:   fmt.Sprintf("%s_disk%d", name, i),
			SizeGB: int(disk.Size),
		}

		disksWorkloads = append(disksWorkloads, DiskWorkload)
		mountsWorkloads = append(mountsWorkloads, workloads.Mount{DiskName: DiskWorkload.Name, MountPoint: disk.Mount})
	}
	return
}

func updateFailedDeployments(ctx context.Context, tfPluginClient deployer.TFPluginClient, nodesIDs []int, groupDeployments *groupDeploymentsInfo) {
	for idx, deployment := range groupDeployments.vmDeployments {
		if deployment.ContractID == 0 || isFailedNetwork(*groupDeployments.networkDeployments[idx]) {
			err := tfPluginClient.NetworkDeployer.Cancel(ctx, groupDeployments.networkDeployments[idx])
			if err != nil {
				log.Debug().Err(err)
			}

			nodeID := uint32(nodesIDs[idx%len(nodesIDs)])
			groupDeployments.vmDeployments[idx].NodeID = nodeID
			groupDeployments.networkDeployments[idx].Nodes = []uint32{nodeID}
		}
	}
}

func isFailedNetwork(network workloads.ZNet) bool {
	if len(network.NodeDeploymentID) == 0 {
		return true
	} else {
		for _, contract := range network.NodeDeploymentID {
			if contract == 0 {
				return true
			}
		}
	}
	return false
}
