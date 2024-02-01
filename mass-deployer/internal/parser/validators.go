package parser

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/cosmos/go-bip39"
	deployer "github.com/threefoldtech/tfgrid-sdk-go/mass-deployer/pkg/mass-deployer"
)

var alphanumeric = regexp.MustCompile("^[a-z0-9_]*$")

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

func validateVMs(vms []deployer.Vms, nodeGroups []deployer.NodesGroup, sskKeys map[string]string) error {
	usedResources := make(map[string]map[string]interface{}, len(nodeGroups))
	var vmNodeGroupExists bool

	for _, nodeGroup := range nodeGroups {
		nodeGroupName := strings.TrimSpace(nodeGroup.Name)
		if !alphanumeric.MatchString(nodeGroupName) {
			return fmt.Errorf("node group name: '%s' is invalid, should be lowercase alphanumeric and underscore only", nodeGroupName)
		}
	}

	for _, vm := range vms {
		vmName := strings.TrimSpace(vm.Name)
		if !alphanumeric.MatchString(vmName) {
			return fmt.Errorf("vms group name: '%s' is invalid, should be lowercase alphanumeric and underscore only", vmName)
		}

		for _, nodeGroup := range nodeGroups {
			nodeGroupName := strings.TrimSpace(nodeGroup.Name)
			if strings.TrimSpace(vm.NodeGroup) == nodeGroupName {
				vmNodeGroupExists = true

				usedVMsResources := setVMUsedResources(vm, usedResources[nodeGroupName])
				usedResources[nodeGroupName] = usedVMsResources

				if usedVMsResources["free_cpu"].(uint64) > nodeGroup.FreeCPU {
					return fmt.Errorf("cannot find enough cpu in node group '%s' for vm group '%s', needed cpu is %d while available cpu is %d, maybe previous vms groups used it", nodeGroupName, vmName, usedVMsResources["free_cpu"], nodeGroup.FreeCPU)
				}

				if usedVMsResources["free_mru"].(float32) > nodeGroup.FreeMRU {
					return fmt.Errorf("cannot find enough memory in node group '%s' for vm group '%s', needed memory is %v GB while available memory is %v GB, maybe previous vms groups used it", nodeGroupName, vmName, usedVMsResources["free_mru"], nodeGroup.FreeMRU)
				}

				if usedVMsResources["free_ssd"].(uint64) > nodeGroup.FreeSRU {
					return fmt.Errorf("cannot find enough ssd in node group '%s' for vm group '%s', needed ssd is %d GB while available ssd is %d GB, maybe previous vms groups used it", nodeGroupName, vmName, usedVMsResources["free_ssd"], nodeGroup.FreeSRU)
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

		if _, ok := sskKeys[vm.SSHKey]; !ok {
			return fmt.Errorf("vms group '%s' ssh key is not found, should refer to one from ssh keys map", vm.Name)
		}

		if err := validateFlist(vm.Flist, vm.Name); err != nil {
			return err
		}
	}

	return nil
}

func validateFlist(flist, name string) error {
	flistExt := path.Ext(flist)
	if flistExt != ".fl" && flistExt != ".flist" {
		return fmt.Errorf("vms group '%s' flist: '%s' is invalid, should have a valid flist extension", name, flist)
	}

	if flistExt == ".flist" {
		hash := md5.Sum([]byte(flist + ".md5"))
		response, err := http.Get(flist + fmt.Sprintf("%x", hash))
		if err != nil {
			return fmt.Errorf("vms group '%s' flist: '%s' is invalid, failed to download flist", name, flist)
		}
		defer response.Body.Close()
	}

	return nil
}

func setVMUsedResources(vmsGroup deployer.Vms, vmUsedResources map[string]interface{}) map[string]interface{} {
	if _, ok := vmUsedResources["free_cpu"]; !ok {
		vmUsedResources = make(map[string]interface{}, 5)
		vmUsedResources["free_cpu"] = uint64(0)
		vmUsedResources["free_mru"] = float32(0)
		vmUsedResources["free_ssd"] = uint64(0)
		vmUsedResources["public_ip4"] = uint64(0)
		vmUsedResources["public_ip6"] = uint64(0)
	}

	if vmsGroup.FreeCPU > vmUsedResources["free_cpu"].(uint64) {
		vmUsedResources["free_cpu"] = vmsGroup.FreeCPU
	}

	if vmsGroup.FreeMRU > vmUsedResources["free_mru"].(float32) {
		vmUsedResources["free_mru"] = vmsGroup.FreeMRU
	}

	var ssdDisks uint64
	for _, disk := range vmsGroup.SSDDisks {
		ssdDisks += disk.Size
	}

	vmUsedResources["free_ssd"] = vmUsedResources["free_ssd"].(uint64) + vmsGroup.RootSize + ssdDisks

	if vmsGroup.PublicIP4 {
		vmUsedResources["public_ip4"] = vmUsedResources["public_ip4"].(uint64) + vmsGroup.Count
	}

	if vmsGroup.PublicIP6 {
		vmUsedResources["public_ip6"] = vmUsedResources["public_ip6"].(uint64) + vmsGroup.Count
	}

	return vmUsedResources
}
