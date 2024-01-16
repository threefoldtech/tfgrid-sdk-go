package parser

import (
	"fmt"
	"slices"

	"gopkg.in/yaml.v3"
)

func ParseConfig(configFile []byte) (Config, error) {
	conf := Config{}

	err := yaml.Unmarshal(configFile, &conf)
	if err != nil {
		return Config{}, err
	}

	if len(conf.NodeGroups) == 0 {
		return Config{}, fmt.Errorf("couldn't find any nodes groups to use in deployment")
	}
	if len(conf.Vms) == 0 {
		return Config{}, fmt.Errorf("couldn't find any vms that need to be deployed")
	}
	if len(conf.SSHKeys) == 0 {
		return Config{}, fmt.Errorf("user ssh_keys is invalid, ssh_keys shouldn't be empty")
	}
	if len(conf.Mnemonic) == 0 {
		return Config{}, fmt.Errorf("user mnemonics is invalid, mnemonics shouldn't be empty")
	}
	networks := []string{"dev", "test", "qa", "main"}
	if !slices.Contains(networks, conf.Network) {
		return Config{}, fmt.Errorf("netwok %s is invalid", conf.Network)
	}

	for _, nodeGroup := range conf.NodeGroups {
		if nodeGroup.Name == "" {
			return Config{}, fmt.Errorf("node groups name is invalid, shouldn't be empty")
		}
		if nodeGroup.NodesCount <= 0 {
			return Config{}, fmt.Errorf("nodes_count in node_group: %s is invalid, should be a positive number", nodeGroup.Name)
		}
		if nodeGroup.FreeCPU <= 0 {
			return Config{}, fmt.Errorf("free_cpu in node_group: %s is invalid, should be a positive number", nodeGroup.Name)
		}
		if nodeGroup.FreeMRU <= 0 {
			return Config{}, fmt.Errorf("free_mru in node_group: %s is invalid, should be a positive number", nodeGroup.Name)
		}
	}

	for _, vm := range conf.Vms {
		if vm.Name == "" {
			return Config{}, fmt.Errorf("vms name is invalid, shouldn't be empty")
		}
		if vm.Count <= 0 {
			return Config{}, fmt.Errorf("vms_count in vms: %s is invalid, should be a positive number", vm.Name)
		}
		if vm.Nodegroup == "" {
			return Config{}, fmt.Errorf("vms node_group is invalid, shouldn't be empty")
		}
		if vm.FreeCPU <= 0 {
			return Config{}, fmt.Errorf("cpu in vms: %s is invalid, should be a positive number", vm.Name)
		}
		if vm.FreeMRU <= 0 {
			return Config{}, fmt.Errorf("mem in vms: %s is invalid, should be a positive number", vm.Name)
		}
		for _, disk := range vm.SSDDisks {
			if disk.Capacity <= 0 {
				return Config{}, fmt.Errorf("ssd disk capacity in vms: %s is invalid, should be a positive number", vm.Name)
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
		if vm.Rootsize < 0 {
			return Config{}, fmt.Errorf("root_size in vms: %s is invalid, should be a positive number", vm.Name)
		}
	}

	return conf, nil
}
