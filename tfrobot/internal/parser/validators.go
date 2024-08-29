package parser

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/cosmos/go-bip39"
	"github.com/go-playground/validator/v10"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	tfrobot "github.com/threefoldtech/tfgrid-sdk-go/tfrobot/pkg/deployer"
	"golang.org/x/sync/errgroup"
)

var alphanumeric = regexp.MustCompile("^[a-z0-9_]+$")

func validateMnemonic(mnemonic string) error {
	if !bip39.IsMnemonicValid(mnemonic) {
		return fmt.Errorf("invalid mnemonic: '%s'", mnemonic)
	}
	return nil
}

func validateNetwork(network string) error {
	networks := []string{"dev", "test", "qa", "main"}
	if !slices.Contains(networks, network) {
		return fmt.Errorf("invalid network: '%s', network can be one of %+v", network, networks)
	}
	return nil
}

func validateNodeGroups(nodeGroups []tfrobot.NodesGroup, tfPluginClient deployer.TFPluginClient) error {
	errGroup := new(errgroup.Group)
	for _, group := range nodeGroups {
		group := group

		errGroup.Go(func() error {
			nodeGroupName := strings.TrimSpace(group.Name)

			if !alphanumeric.MatchString(nodeGroupName) {
				return fmt.Errorf("node group name: '%s' is invalid, should be lowercase alphanumeric and underscore only", nodeGroupName)
			}

			// check if the node group name was used previously with the old name format <name>
			contracts, err := tfPluginClient.ContractsGetter.ListContractsOfProjectName(nodeGroupName, true)
			if err != nil {
				return err
			}

			if len(contracts.NodeContracts) != 0 {
				return fmt.Errorf("node group name: '%s' is invalid, should be unique name across all deployments", nodeGroupName)
			}

			// check if the node group name was used previously with the new name format "vm/<name>"
			contracts, err = tfPluginClient.ContractsGetter.ListContractsOfProjectName(fmt.Sprintf("vm/%s", nodeGroupName), true)
			if err != nil {
				return err
			}

			if len(contracts.NodeContracts) != 0 {
				return fmt.Errorf("node group name: '%s' is invalid, should be unique name across all deployments", nodeGroupName)
			}

			return nil
		})
	}

	return errGroup.Wait()
}

func validateVMs(vms []tfrobot.Vms, nodeGroups []tfrobot.NodesGroup, sskKeys map[string]string) error {
	usedResources := make(map[string]map[string]interface{}, len(nodeGroups))
	var vmNodeGroupExists bool

	for _, vm := range vms {
		vmName := strings.TrimSpace(vm.Name)
		if !alphanumeric.MatchString(vmName) {
			return fmt.Errorf("vms group name: '%s' is invalid, should be lowercase alphanumeric and underscore only", vmName)
		}
		if _, ok := sskKeys[vm.SSHKey]; !ok {
			return fmt.Errorf("vms group '%s' ssh key is not found, should refer to one from ssh keys map", vm.Name)
		}

		if err := workloads.ValidateFlist(vm.Flist, ""); err != nil {
			return fmt.Errorf("invalid flist for vms group '%s', %w", vm.Name, err)
		}

		for _, nodeGroup := range nodeGroups {
			nodeGroupName := strings.TrimSpace(nodeGroup.Name)
			if strings.TrimSpace(vm.NodeGroup) == nodeGroupName {
				vmNodeGroupExists = true

				usedVMsResources := setVMUsedResources(vm, usedResources[nodeGroupName])
				usedResources[nodeGroupName] = usedVMsResources

				if usedVMsResources["free_cpu"].(uint8) > uint8(nodeGroup.FreeCPU) {
					return fmt.Errorf("cannot find enough cpu in node group '%s' for vm group '%s', needed cpu is %d while available cpu is %d", nodeGroupName, vmName, usedVMsResources["free_cpu"], nodeGroup.FreeCPU)
				}

				if usedVMsResources["free_mru"].(float32) > nodeGroup.FreeMRU*float32(nodeGroup.NodesCount) {
					return fmt.Errorf("cannot find enough memory in node group '%s' for vm group '%s', needed memory is %v GB while available memory is %v GB", nodeGroupName, vmName, usedVMsResources["free_mru"], nodeGroup.FreeMRU*float32(nodeGroup.NodesCount))
				}

				if usedVMsResources["free_ssd"].(uint64) > nodeGroup.FreeSRU*nodeGroup.NodesCount {
					return fmt.Errorf("cannot find enough ssd in node group '%s' for vm group '%s', needed ssd is %d GB while available ssd is %d GB, maybe previous vms groups used it", nodeGroupName, vmName, usedVMsResources["free_ssd"], nodeGroup.FreeSRU*nodeGroup.NodesCount)
				}

				var nodeGroupPublicIPs uint64
				var nodeGroupPublicIP6s uint64
				if nodeGroup.PublicIP4 {
					nodeGroupPublicIPs = nodeGroup.NodesCount
				}
				if nodeGroup.PublicIP6 {
					nodeGroupPublicIP6s = nodeGroup.NodesCount
				}

				if usedVMsResources["public_ip4"].(uint64) > nodeGroupPublicIPs {
					return fmt.Errorf("cannot find enough public ipv4 in node group '%s' for vm group '%s', needed public ipv4 is %d while available public ipv4 is %d, maybe previous vms groups used it", nodeGroupName, vmName, usedVMsResources["public_ip4"], nodeGroupPublicIPs)
				}

				if usedVMsResources["public_ip6"].(uint64) > nodeGroupPublicIP6s {
					return fmt.Errorf("cannot find enough public ipv6 in node group '%s' for vm group '%s', needed public ipv6 is %d while available public ipv6 is %d, maybe previous vms groups used it", nodeGroupName, vmName, usedVMsResources["public_ip6"], nodeGroupPublicIP6s)
				}
			}
		}

		if !vmNodeGroupExists {
			return fmt.Errorf("node group: '%s' in vms group: '%s' is not found", vm.NodeGroup, vm.Name)
		}

		v := validator.New(validator.WithRequiredStructEnabled())
		for _, disk := range vm.SSDDisks {
			if err := v.Struct(disk); err != nil {
				return parseValidationError(err)
			}
		}

		for _, volume := range vm.Volumes {
			if err := v.Struct(volume); err != nil {
				return parseValidationError(err)
			}
		}

	}

	return nil
}

func setVMUsedResources(vmsGroup tfrobot.Vms, vmUsedResources map[string]interface{}) map[string]interface{} {
	if _, ok := vmUsedResources["free_cpu"]; !ok {
		vmUsedResources = make(map[string]interface{}, 5)
		vmUsedResources["free_cpu"] = uint8(0)
		vmUsedResources["free_mru"] = float32(0)
		vmUsedResources["free_ssd"] = uint64(0)
		vmUsedResources["public_ip4"] = uint64(0)
		vmUsedResources["public_ip6"] = uint64(0)
	}

	if vmsGroup.FreeCPU > vmUsedResources["free_cpu"].(uint8) {
		vmUsedResources["free_cpu"] = vmsGroup.FreeCPU
	}

	vmUsedResources["free_mru"] = vmUsedResources["free_mru"].(float32) + (vmsGroup.FreeMRU * float32(vmsGroup.Count))

	var ssdDisks uint64
	for _, disk := range vmsGroup.SSDDisks {
		ssdDisks += disk.Size * vmsGroup.Count
	}
	// free volume calc
	var volumes uint64
	for _, volume := range vmsGroup.Volumes {
		volumes += volume.Size * vmsGroup.Count
	}

	vmUsedResources["free_ssd"] = vmUsedResources["free_ssd"].(uint64) + (vmsGroup.RootSize * vmsGroup.Count) + ssdDisks + volumes

	if vmsGroup.PublicIP4 {
		vmUsedResources["public_ip4"] = vmUsedResources["public_ip4"].(uint64) + vmsGroup.Count
	}

	if vmsGroup.PublicIP6 {
		vmUsedResources["public_ip6"] = vmUsedResources["public_ip6"].(uint64) + vmsGroup.Count
	}

	return vmUsedResources
}
