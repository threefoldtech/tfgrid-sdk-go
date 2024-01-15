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

	networks := []string{"dev", "test", "qa", "main"}
	if slices.Contains(networks, conf.Network) {
		return Config{}, fmt.Errorf("invalid network: %s", conf.Network)
	}
	if len(conf.Mnemonic) == 0 {
		return Config{}, fmt.Errorf("user mnemonics shouldn't be empty")
	}
	if len(conf.SSHKeys) == 0 {
		return Config{}, fmt.Errorf("ssh_keys shouldn't be empty")
	}
	if len(conf.Vms) == 0 {
		return Config{}, fmt.Errorf("couldn't find any vms that need to be deployed")
	}
	if len(conf.NodeGroups) == 0 {
		return Config{}, fmt.Errorf("couldn't find any nodes groups to use in deployment")
	}

	for _, nodeGroup := range conf.NodeGroups {
		if nodeGroup.Name == "" {
			return Config{}, fmt.Errorf("node groups name shouldn't be empty")
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
		if nodeGroup.FreeSSD <= 0 {
			return Config{}, fmt.Errorf("free_ssd in node_group: %s is invalid, should be a positive number", nodeGroup.Name)
		}
		if nodeGroup.FreeHDD <= 0 {
			return Config{}, fmt.Errorf("free_hdd in node_group: %s is invalid, should be a positive number", nodeGroup.Name)
		}
		if nodeGroup.MinBwd <= 0 {
			return Config{}, fmt.Errorf("min_bwd in node_group: %s is invalid, should be a positive number", nodeGroup.Name)
		}
	}
	// name"`
	// vms_count"`
	// node_group"
	// cpu"`
	// mem"`
	// ssd"`
	// pubip4"`
	// pubip6"`
	// planetary"`
	// flist"`
	// root_size"`
	// entry_point
	// ssh_key"`

	for _, vm := range conf.Vms {
		if vm.Name == "" {
		}
	}

	return conf, nil
}
