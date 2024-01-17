package parser

import (
	"fmt"
	"slices"
	"strings"

	"github.com/cosmos/go-bip39"
	deployer "github.com/threefoldtech/tfgrid-sdk-go/mass-deployer/pkg/mass-deployer"
	"gopkg.in/yaml.v3"
)

func ParseConfig(configFile []byte) (deployer.Config, error) {
	conf := deployer.Config{}
	nodeGroupsNames := []string{}

	err := yaml.Unmarshal(configFile, &conf)
	if err != nil {
		return deployer.Config{}, err
	}

	if len(conf.NodeGroups) == 0 {
		return deployer.Config{}, fmt.Errorf("couldn't find any node_groups to use in deployment")
	}
	if len(conf.Vms) == 0 {
		return deployer.Config{}, fmt.Errorf("couldn't find any vms that need to be deployed")
	}
	if len(conf.SSHKeys) == 0 {
		return deployer.Config{}, fmt.Errorf("user ssh_keys are invalid, ssh_keys shouldn't be empty")
	}
	if !bip39.IsMnemonicValid(conf.Mnemonic) {
		return deployer.Config{}, fmt.Errorf("invalid user mnemonic: %s", conf.Mnemonic)
	}
	networks := []string{"dev", "test", "qa", "main"}
	if !slices.Contains(networks, conf.Network) {
		return deployer.Config{}, fmt.Errorf("invalid netwok: %s", conf.Network)
	}

	for _, nodeGroup := range conf.NodeGroups {
		name := strings.TrimSpace(nodeGroup.Name)
		if name == "" {
			return deployer.Config{}, fmt.Errorf("node groups name is invalid, shouldn't be empty")
		}
		if nodeGroup.NodesCount == 0 {
			return deployer.Config{}, fmt.Errorf("nodes_count in node_group: %s is invalid, shouldn't be equal to 0", nodeGroup.Name)
		}
		if nodeGroup.FreeCPU == 0 {
			return deployer.Config{}, fmt.Errorf("free_cpu in node_group: %s is invalid, shouldn't be equal to 0", nodeGroup.Name)
		}
		if nodeGroup.FreeMRU == 0 {
			return deployer.Config{}, fmt.Errorf("free_mru in node_group: %s is invalid, shouldn't be equal to 0", nodeGroup.Name)
		}
		nodeGroupsNames = append(nodeGroupsNames, name)
	}

	for _, vm := range conf.Vms {
		if strings.TrimSpace(vm.Name) == "" {
			return deployer.Config{}, fmt.Errorf("vms group name is invalid, shouldn't be empty")
		}
		if vm.Count == 0 {
			return deployer.Config{}, fmt.Errorf("vms_count in vms group: %s is invalid, shouldn't be equal to 0", vm.Name)
		}
		if !slices.Contains(nodeGroupsNames, strings.TrimSpace(vm.Nodegroup)) {
			return deployer.Config{}, fmt.Errorf("invalid node_group: %s in vms group: %s", vm.Nodegroup, vm.Name)
		}
		if vm.FreeCPU == 0 {
			return deployer.Config{}, fmt.Errorf("cpu in vms group: %s is invalid, shouldn't be equal to 0", vm.Name)
		}
		if vm.FreeMRU == 0 {
			return deployer.Config{}, fmt.Errorf("mem in vms group: %s is invalid, shouldn't be equal to 0", vm.Name)
		}
		for _, disk := range vm.SSDDisks {
			if disk.Size <= 0 {
				return deployer.Config{}, fmt.Errorf("ssd disk size in vms group: %s is invalid, shouldn't be equal to 0", vm.Name)
			}
			if strings.TrimSpace(disk.Mount) == "" {
				return deployer.Config{}, fmt.Errorf("vms group %s mount point is invalid, shouldn't be empty", vm.Name)
			}
		}
		if strings.TrimSpace(vm.Flist) == "" {
			return deployer.Config{}, fmt.Errorf("vms group %s flist is invalid, shouldn't be empty", vm.Name)
		}
		if strings.TrimSpace(vm.Entrypoint) == "" {
			return deployer.Config{}, fmt.Errorf("vms group %s entry_point is invalid, shouldn't be empty", vm.Name)
		}
		if _, ok := conf.SSHKeys[vm.SSHKey]; !ok {
			return deployer.Config{}, fmt.Errorf("vms group %s ssh_key is invalid, should be valid name refers to one of ssh_keys list", vm.Name)
		}
	}

	return conf, nil
}
