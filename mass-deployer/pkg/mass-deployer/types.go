package deployer

type Config struct {
	NodeGroups []NodesGroup      `yaml:"node_groups"`
	Vms        []Vms             `yaml:"vms"`
	SSHKeys    map[string]string `yaml:"ssh_keys"`
	Mnemonic   string            `yaml:"mnemonic"`
	Network    string            `yaml:"network"`
}

type NodesGroup struct {
	Name       string `yaml:"name"`
	NodesCount uint64 `yaml:"nodes_count"`
	FreeCPU    uint64 `yaml:"free_cpu"`
	FreeMRU    uint64 `yaml:"free_mru"`
	FreeSRU    uint64 `yaml:"free_ssd"`
	FreeHRU    uint64 `yaml:"free_hdd"`
	Dedicated  bool   `yaml:"dedicated"`
	Pubip4     bool   `yaml:"pubip4"`
	Pubip6     bool   `yaml:"pubip6"`
	Certified  bool   `yaml:"certified"`
	Regions    string `yaml:"regions"`
}

type Vms struct {
	Name       string `yaml:"name"`
	Count      uint64 `yaml:"vms_count"`
	Nodegroup  string `yaml:"node_group"`
	FreeCPU    uint64 `yaml:"cpu"`
	FreeMRU    uint64 `yaml:"mem"`
	SSDDisks   []Disk `yaml:"ssd"`
	Pubip4     bool   `yaml:"pubip4"`
	Pubip6     bool   `yaml:"pubip6"`
	Planetary  bool   `yaml:"planetary"`
	Flist      string `yaml:"flist"`
	Rootsize   uint64 `yaml:"root_size"`
	Entrypoint string `yaml:"entry_point"`
	SSHKey     string `yaml:"ssh_key"`
}

type Disk struct {
	Capacity uint64 `yaml:"capacity"`
	Mount    string `yaml:"mount_point"`
}
