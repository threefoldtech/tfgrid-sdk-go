package parser

import (
	"fmt"
	"slices"

	"github.com/cosmos/go-bip39"
	"gopkg.in/yaml.v3"
)

func ParseConfig(configFile []byte) (Config, error) {
	conf := Config{}

	err := yaml.Unmarshal(configFile, &conf)
	if err != nil {
		return Config{}, err
	}

	if len(conf.NodeGroups) == 0 {
		return Config{}, fmt.Errorf("couldn't find any node_groups to use in deployment")
	}
	if len(conf.Vms) == 0 {
		return Config{}, fmt.Errorf("couldn't find any vms that need to be deployed")
	}
	if len(conf.SSHKeys) == 0 {
		return Config{}, fmt.Errorf("user ssh_keys are invalid, ssh_keys shouldn't be empty")
	}
	if bip39.IsMnemonicValid(conf.Mnemonic) {
		return Config{}, fmt.Errorf("invalid user mnemonic: %s", conf.Mnemonic)
	}
	networks := []string{"dev", "test", "qa", "main"}
	if !slices.Contains(networks, conf.Network) {
		return Config{}, fmt.Errorf("invalid netwok: %s", conf.Network)
	}

	for _, nodeGroup := range conf.NodeGroups {
		if nodeGroup.Name == "" {
			return Config{}, fmt.Errorf("node groups name is invalid, shouldn't be empty")
		}
		if nodeGroup.NodesCount == 0 {
			return Config{}, fmt.Errorf("nodes_count in node_group: %s is invalid, shouldn't be equal to 0", nodeGroup.Name)
		}
		if nodeGroup.FreeCPU == 0 {
			return Config{}, fmt.Errorf("free_cpu in node_group: %s is invalid, shouldn't be equal to 0", nodeGroup.Name)
		}
		if nodeGroup.FreeMRU == 0 {
			return Config{}, fmt.Errorf("free_mru in node_group: %s is invalid, shouldn't be equal to 0", nodeGroup.Name)
		}
	}

	for _, vm := range conf.Vms {
		if vm.Name == "" {
			return Config{}, fmt.Errorf("vms name is invalid, shouldn't be empty")
		}
		if vm.Count == 0 {
			return Config{}, fmt.Errorf("vms_count in vms: %s is invalid, shouldn't be equal to 0", vm.Name)
		}
		if vm.Nodegroup == "" {
			return Config{}, fmt.Errorf("vms node_group is invalid, shouldn't be empty")
		}
		if vm.FreeCPU == 0 {
			return Config{}, fmt.Errorf("cpu in vms: %s is invalid, shouldn't be equal to 0", vm.Name)
		}
		if vm.FreeMRU == 0 {
			return Config{}, fmt.Errorf("mem in vms: %s is invalid, shouldn't be equal to 0", vm.Name)
		}
		for _, disk := range vm.SSDDisks {
			if disk.Capacity <= 0 {
				return Config{}, fmt.Errorf("ssd disk capacity in vms: %s is invalid, shouldn't be equal to 0", vm.Name)
			}
			if disk.Mount == "" {
				return Config{}, fmt.Errorf("vms mount point is invalid, shouldn't be empty")
			}
		}
		if vm.Flist == "" {
			return Config{}, fmt.Errorf("vms flist is invalid, shouldn't be empty")
		}
		if vm.Entrypoint == "" {
			return Config{}, fmt.Errorf("vms entry_point is invalid, shouldn't be empty")
		}
		if _, ok := conf.SSHKeys[vm.SSHKey]; !ok {
			return Config{}, fmt.Errorf("vms ssh_key is invalid, should be valid ssh_key refers to one of ssh_keys list")
		}
	}

	return conf, nil
}
