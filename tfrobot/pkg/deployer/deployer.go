package deployer

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"syscall"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/sethvargo/go-retry"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

const (
	DefaultMaxRetries  = 5
	maxGoroutinesCount = 100
)

func RunDeployer(ctx context.Context, cfg Config, tfPluginClient deployer.TFPluginClient, output string, debug bool) error {
	passedGroups := map[string][]*workloads.Deployment{}
	failedGroups := map[string]string{}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	deploymentStart := time.Now()

	// close ctx on SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = DefaultMaxRetries
	}

	var excludedNodes []uint64

	for _, nodeGroup := range cfg.NodeGroups {
		log.Info().Str("Node group", nodeGroup.Name).Msg("Running deployment")

		var groupDeployments groupDeploymentsInfo
		trial := 1

		if err := retry.Do(ctx, retry.WithMaxRetries(cfg.MaxRetries, retry.NewConstant(1*time.Nanosecond)), func(ctx context.Context) error {
			if trial != 1 {
				log.Info().Str("Node group", nodeGroup.Name).Int("Deployment trial", trial).Msg("Retrying to deploy")
			}

			if err := deployNodeGroup(ctx, tfPluginClient, &groupDeployments, nodeGroup, excludedNodes, cfg.Vms, cfg.SSHKeys); err != nil {
				trial++
				log.Debug().Err(err).Str("Node group", nodeGroup.Name).Msg("failed to deploy")

				blockedNodes := getBlockedNodes(groupDeployments)
				nodeGroup.NodesCount = uint64(len(blockedNodes))
				excludedNodes = append(excludedNodes, blockedNodes...)

				return retry.RetryableError(err)
			}

			log.Info().Str("Node group", nodeGroup.Name).Msg("Done deploying")
			passedGroups[nodeGroup.Name] = groupDeployments.vmDeployments

			return nil
		}); err != nil {

			failedGroups[nodeGroup.Name] = err.Error()
			err := tfPluginClient.CancelByProjectName(fmt.Sprintf("vm/%s", nodeGroup.Name))
			if err != nil {
				log.Debug().Err(err).Send()
			}
		}
	}

	// cancel all deployments if ctx is closed
	if err := ctx.Err(); err != nil {
		for _, group := range cfg.NodeGroups {
			err := tfPluginClient.CancelByProjectName(fmt.Sprintf("vm/%s", group.Name))
			if err != nil {
				log.Debug().Err(err).Send()
			}
		}
		log.Fatal().Err(fmt.Errorf("failed to run deployer, deployment was interrupted with signal SIGTERM")).Send()
	}

	endTime := time.Since(deploymentStart)

	// load deployed deployments
	outputBytes, err := loadAfterDeployment(ctx, tfPluginClient, passedGroups, failedGroups, cfg.MaxRetries, filepath.Ext(output) == ".json")
	if err != nil {
		return err
	}

	fmt.Println(string(outputBytes))
	log.Info().Msgf("Deployment took %s", endTime)

	return os.WriteFile(output, outputBytes, 0644)
}

func deployNodeGroup(
	ctx context.Context,
	tfPluginClient deployer.TFPluginClient,
	groupDeployments *groupDeploymentsInfo,
	nodeGroup NodesGroup,
	excludedNodes []uint64,
	vms []Vms,
	sshKeys map[string]string,
) error {
	log.Info().Str("Node group", nodeGroup.Name).Msg("Filter nodes")
	nodesIDs, err := filterNodes(ctx, tfPluginClient, nodeGroup, excludedNodes)
	if err != nil {
		return err
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
	return massDeploy(ctx, tfPluginClient, groupDeployments)
}

func loadAfterDeployment(
	ctx context.Context,
	tfPluginClient deployer.TFPluginClient,
	deployedGroups map[string][]*workloads.Deployment,
	failedGroups map[string]string,
	retries uint64,
	asJson bool,
) ([]byte, error) {
	var loadedgroups map[string][]vmOutput

	if len(deployedGroups) > 0 {
		log.Info().Msg("Loading deployments")
		groupsContracts := getDeploymentsContracts(deployedGroups)

		var failed map[string]string
		loadedgroups, failed = batchLoadNodeGroupsInfo(ctx, tfPluginClient, groupsContracts, retries)

		for nodeGroup, err := range failed {
			failedGroups[nodeGroup] = err
		}
	}

	return parseDeploymentOutput(loadedgroups, failedGroups, asJson)
}

func parseVMsGroup(vms []Vms, nodeGroup string, nodesIDs []int, sshKeys map[string]string) groupDeploymentsInfo {
	vmsOfNodeGroup := []Vms{}
	for _, vm := range vms {
		if vm.NodeGroup == nodeGroup {
			vmsOfNodeGroup = append(vmsOfNodeGroup, vm)
		}
	}

	log.Debug().Str("Node group", nodeGroup).Msg("Build deployments")
	return buildDeployments(vmsOfNodeGroup, nodesIDs, sshKeys)
}

func updateFailedDeployments(ctx context.Context, tfPluginClient deployer.TFPluginClient, nodesIDs []int, groupDeployments *groupDeploymentsInfo) {
	var networksToBeCanceled []*workloads.ZNet
	for idx, network := range groupDeployments.networkDeployments {
		if groupDeployments.vmDeployments[idx].ContractID == 0 {
			networksToBeCanceled = append(networksToBeCanceled, network)
		}
	}

	err := tfPluginClient.NetworkDeployer.BatchCancel(ctx, networksToBeCanceled)
	if err != nil {
		log.Debug().Err(err).Send()
	}

	for idx, deployment := range groupDeployments.vmDeployments {
		if deployment.ContractID == 0 || len(groupDeployments.networkDeployments[idx].NodeDeploymentID) == 0 {
			nodeID := uint32(nodesIDs[idx%len(nodesIDs)])
			groupDeployments.vmDeployments[idx].NodeID = nodeID
			groupDeployments.networkDeployments[idx].Nodes = []uint32{nodeID}
		}
	}
}

func massDeploy(ctx context.Context, tfPluginClient deployer.TFPluginClient, deployments *groupDeploymentsInfo) error {
	// deploy only contracts that need to be deployed
	networks, vms := getNotDeployedDeployments(deployments)
	var multiErr error

	log.Debug().Msg(fmt.Sprintf("Deploying %d networks, this may to take a while", len(deployments.networkDeployments)))
	if err := tfPluginClient.NetworkDeployer.BatchDeploy(ctx, networks, false); err != nil {
		log.Debug().Err(err).Send()
		multiErr = multierror.Append(multiErr, err)
	}

	log.Debug().Msg(fmt.Sprintf("Deploying %d virtual machines, this may to take a while", len(deployments.vmDeployments)))
	if err := tfPluginClient.DeploymentDeployer.BatchDeploy(ctx, vms); err != nil {
		log.Debug().Err(err).Send()
		multiErr = multierror.Append(multiErr, err)
	}

	return multiErr
}

func buildDeployments(vms []Vms, nodesIDs []int, sshKeys map[string]string) groupDeploymentsInfo {
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
		solutionType := fmt.Sprintf("vm/%s", vmGroup.NodeGroup)

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
				SolutionType: solutionType,
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
			deployment := workloads.NewDeployment(vm.Name, nodeID, solutionType, nil, network.Name, disks, nil, []workloads.VM{vm}, nil)

			vmDeployments = append(vmDeployments, &deployment)
			networkDeployments = append(networkDeployments, &network)
		}
	}
	return groupDeploymentsInfo{vmDeployments: vmDeployments, networkDeployments: networkDeployments}
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

func getNotDeployedDeployments(groupDeployments *groupDeploymentsInfo) ([]*workloads.ZNet, []*workloads.Deployment) {
	var failedVmDeployments []*workloads.Deployment
	var failedNetworkDeployments []*workloads.ZNet

	for i := range groupDeployments.networkDeployments {
		if len(groupDeployments.networkDeployments[i].NodeDeploymentID) == 0 {
			failedNetworkDeployments = append(failedNetworkDeployments, groupDeployments.networkDeployments[i])
		}

		if groupDeployments.vmDeployments[i].ContractID == 0 {
			failedVmDeployments = append(failedVmDeployments, groupDeployments.vmDeployments[i])
		}

	}

	return failedNetworkDeployments, failedVmDeployments
}

func getDeploymentsContracts(groupsInfo map[string][]*workloads.Deployment) map[string]NodeContracts {
	nodeGroupsContracts := make(map[string]NodeContracts)
	for nodeGroup, groupDeployments := range groupsInfo {
		contracts := make(NodeContracts)
		for _, deployment := range groupDeployments {
			contracts[deployment.NodeID] = append(contracts[deployment.NodeID], deployment.ContractID)
		}
		nodeGroupsContracts[nodeGroup] = contracts
	}
	return nodeGroupsContracts
}

func getBlockedNodes(groupDeployments groupDeploymentsInfo) []uint64 {
	var blockedNodes []uint64

	for _, deployment := range groupDeployments.vmDeployments {
		if deployment.ContractID == 0 && !slices.Contains(blockedNodes, uint64(deployment.NodeID)) {
			blockedNodes = append(blockedNodes, uint64(deployment.NodeID))
		}
	}

	return blockedNodes
}
