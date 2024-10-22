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
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
)

const (
	DefaultMaxRetries  = 5
	maxGoroutinesCount = 100
)

func RunDeployer(ctx context.Context, cfg Config, tfPluginClient deployer.TFPluginClient, output string, debug bool) *multierror.Error {
	passedGroups := map[string][]*workloads.Deployment{}
	var failedGroupsErr *multierror.Error

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
			failedGroupsErr = multierror.Append(failedGroupsErr, fmt.Errorf("%s: %s", nodeGroup.Name, err.Error()))

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
	outputBytes, errs := loadAfterDeployment(ctx, tfPluginClient, passedGroups, cfg.MaxRetries, filepath.Ext(output) == ".json")
	if errs != nil {
		failedGroupsErr = multierror.Append(failedGroupsErr, errs.Errors...)
	}

	fmt.Println(string(outputBytes))
	log.Info().Msgf("Deployment took %s", endTime)

	err := os.WriteFile(output, outputBytes, 0644)
	if err != nil {
		log.Error().Err(err).Send()
	}

	return failedGroupsErr
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
	var ygg bool

	for _, group := range vms {
		if group.Ygg && group.NodeGroup == nodeGroup.Name {
			ygg = true
		}
	}

	log.Info().Str("Node group", nodeGroup.Name).Msg("Filter nodes")
	nodesIDs, err := filterNodes(ctx, tfPluginClient, nodeGroup, excludedNodes, ygg)
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
	retries uint64,
	asJson bool,
) ([]byte, *multierror.Error) {
	var loadedgroups map[string][]vmOutput
	var failedGroupsErr *multierror.Error

	if len(deployedGroups) > 0 {
		log.Info().Msg("Loading deployments")
		groupsContracts := getDeploymentsContracts(deployedGroups)

		loadedgroups, failedGroupsErr = batchLoadNodeGroupsInfo(ctx, tfPluginClient, groupsContracts, retries)
	}

	output, err := parseDeploymentOutput(loadedgroups, asJson)
	if err != nil {
		log.Error().Err(err).Send()
	}

	return output, failedGroupsErr
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
	var networksToBeCanceled []workloads.Network
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
		if deployment.ContractID == 0 || len(groupDeployments.networkDeployments[idx].GetNodeDeploymentID()) == 0 {
			nodeID := uint32(nodesIDs[idx%len(nodesIDs)])
			groupDeployments.vmDeployments[idx].NodeID = nodeID
			groupDeployments.networkDeployments[idx].SetNodes([]uint32{nodeID})

			myceliumKeys := groupDeployments.networkDeployments[idx].GetMyceliumKeys()
			if len(myceliumKeys) != 0 {
				myceliumKey, err := workloads.RandomMyceliumKey()
				if err != nil {
					log.Debug().Err(err).Send()
				}
				groupDeployments.networkDeployments[idx].SetMyceliumKeys(map[uint32][]byte{nodeID: myceliumKey})
			}
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
	var networkDeployments []workloads.Network
	var nodesIDsIdx int

	// here we loop over all groups of vms within the same node group, and for every group
	// we loop over all it's vms and create network and vm deployment for it
	// the nodesIDsIdx is a counter used to get nodeID to be able to distribute load over all nodes
	for _, vmGroup := range vms {
		solutionType := fmt.Sprintf("vm/%s", vmGroup.NodeGroup)

		for i := 0; i < int(vmGroup.Count); i++ {
			nodeID := uint32(nodesIDs[nodesIDsIdx])
			nodesIDsIdx = (nodesIDsIdx + 1) % len(nodesIDs)

			vmName := fmt.Sprintf("%s%d", vmGroup.Name, i)

			network := buildNetworkDeployment(&vmGroup, nodeID, vmName, solutionType)
			deployment := buildDeployment(vmGroup, nodeID, network.GetName(), vmName, solutionType, sshKeys[vmGroup.SSHKey])

			vmDeployments = append(vmDeployments, &deployment)
			networkDeployments = append(networkDeployments, network)
		}
	}
	return groupDeploymentsInfo{vmDeployments: vmDeployments, networkDeployments: networkDeployments}
}

func parseDisks(name string, disks []Disk) (disksWorkloads []workloads.Disk, mountsWorkloads []workloads.Mount) {
	for i, disk := range disks {
		DiskWorkload := workloads.Disk{
			Name:   fmt.Sprintf("%s_disk%d", name, i),
			SizeGB: disk.Size,
		}

		disksWorkloads = append(disksWorkloads, DiskWorkload)
		mountsWorkloads = append(mountsWorkloads, workloads.Mount{Name: DiskWorkload.Name, MountPoint: disk.Mount})
	}
	return
}

func parseVolumes(name string, volumes []Volume) (volWorkloads []workloads.Volume, mountsWorkloads []workloads.Mount) {
	for i, volume := range volumes {
		VolWorkload := workloads.Volume{
			Name:   fmt.Sprintf("%s_volume%d", name, i),
			SizeGB: volume.Size,
		}

		volWorkloads = append(volWorkloads, VolWorkload)
		mountsWorkloads = append(mountsWorkloads, workloads.Mount{Name: VolWorkload.Name, MountPoint: volume.Mount})
	}
	return
}

func getNotDeployedDeployments(groupDeployments *groupDeploymentsInfo) ([]workloads.Network, []*workloads.Deployment) {
	var failedVmDeployments []*workloads.Deployment
	var failedNetworkDeployments []workloads.Network

	for i := range groupDeployments.networkDeployments {
		if len(groupDeployments.networkDeployments[i].GetNodeDeploymentID()) == 0 {
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

func buildDeployment(vmGroup Vms, nodeID uint32, networkName, vmName, solutionType, sshKey string) workloads.Deployment {
	disks, diskMounts := parseDisks(vmName, vmGroup.SSDDisks)
	volumes, volumeMounts := parseVolumes(vmName, vmGroup.Volumes)

	deployment := workloads.NewDeployment("", nodeID, solutionType, nil, networkName, disks, nil, nil, nil, nil, volumes)

	if !vmGroup.WireGuard && !vmGroup.PublicIP4 && !vmGroup.PublicIP6 && !vmGroup.Ygg {
		vm := buildVMLightDeployment(vmGroup, nodeID, vmName, networkName, sshKey, append(diskMounts, volumeMounts...))
		deployment.VmsLight = append(deployment.VmsLight, vm)
		deployment.Name = vm.Name
	} else {
		vm := buildVMDeployment(vmGroup, nodeID, vmName, networkName, sshKey, append(diskMounts, volumeMounts...))
		deployment.Vms = append(deployment.Vms, vm)
		deployment.Name = vm.Name
	}

	return deployment
}

func buildNetworkDeployment(vm *Vms, nodeID uint32, name, solutionType string) workloads.Network {
	if !vm.PublicIP4 && !vm.Ygg && !vm.Mycelium {
		log.Warn().Str("vm name", name).Msg("ygg ip, mycelium ip and public IP options are false. Setting mycelium IP to true")
		vm.Mycelium = true
	}

	// set up mycelium keys
	myceliumKeys := make(map[uint32][]byte)
	if vm.Mycelium {
		key, err := workloads.RandomMyceliumKey()
		if err != nil {
			log.Debug().Err(err).Send()
		}

		myceliumKeys[nodeID] = key
	}

	if !vm.WireGuard && !vm.PublicIP4 && !vm.PublicIP6 && !vm.Ygg {
		return &workloads.ZNetLight{
			Name:        fmt.Sprintf("%s_network", name),
			Description: "network for mass deployment",
			Nodes:       []uint32{nodeID},
			IPRange: zos.IPNet{IPNet: net.IPNet{
				IP:   net.IPv4(10, 20, 0, 0),
				Mask: net.CIDRMask(16, 32),
			}},
			MyceliumKeys: myceliumKeys,
			SolutionType: solutionType,
		}
	}

	return &workloads.ZNet{
		Name:        fmt.Sprintf("%s_network", name),
		Description: "network for mass deployment",
		Nodes:       []uint32{nodeID},
		IPRange: zos.IPNet{IPNet: net.IPNet{
			IP:   net.IPv4(10, 20, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}},
		AddWGAccess:  vm.WireGuard,
		MyceliumKeys: myceliumKeys,
		SolutionType: solutionType,
	}
}

func buildVMDeployment(vm Vms, nodeID uint32, name, networkName, sshKey string, mounts []workloads.Mount) workloads.VM {
	envVars := vm.EnvVars
	if envVars == nil {
		envVars = map[string]string{}
	}
	envVars["SSH_KEY"] = sshKey

	// get random mycelium seeds
	var myceliumSeed []byte
	var err error
	if vm.Mycelium {
		myceliumSeed, err = workloads.RandomMyceliumIPSeed()
		if err != nil {
			log.Debug().Err(err).Send()
		}
	}

	return workloads.VM{
		Name:           name,
		NodeID:         nodeID,
		NetworkName:    networkName,
		Flist:          vm.Flist,
		CPU:            vm.FreeCPU,
		MemoryMB:       uint64(vm.FreeMRU * 1024), // Memory is in MB
		PublicIP:       vm.PublicIP4,
		PublicIP6:      vm.PublicIP6,
		MyceliumIPSeed: myceliumSeed,
		Planetary:      vm.Ygg,
		RootfsSizeMB:   vm.RootSize * 1024, // RootSize is in MB
		Entrypoint:     vm.Entrypoint,
		EnvVars:        envVars,
		Mounts:         mounts,
	}
}

func buildVMLightDeployment(vm Vms, nodeID uint32, name, networkName, sshKey string, mounts []workloads.Mount) workloads.VMLight {
	envVars := vm.EnvVars
	if envVars == nil {
		envVars = map[string]string{}
	}
	envVars["SSH_KEY"] = sshKey

	// get random mycelium seeds
	var myceliumSeed []byte
	var err error
	if vm.Mycelium {
		myceliumSeed, err = workloads.RandomMyceliumIPSeed()
		if err != nil {
			log.Debug().Err(err).Send()
		}
	}

	return workloads.VMLight{
		Name:           name,
		NodeID:         nodeID,
		NetworkName:    networkName,
		Flist:          vm.Flist,
		CPU:            vm.FreeCPU,
		MemoryMB:       uint64(vm.FreeMRU * 1024), // Memory is in MB
		MyceliumIPSeed: myceliumSeed,
		RootfsSizeMB:   vm.RootSize * 1024, // RootSize is in MB
		Entrypoint:     vm.Entrypoint,
		EnvVars:        envVars,
		Mounts:         mounts,
	}
}
