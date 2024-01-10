package parser

type Config struct {
	NodeGroups []NodesGroup      `yaml:"node_groups"`
	Vms        []Vm              `yaml:"vms"`
	SSHKeys    map[string]string `yaml:"ssh_keys"`
	Mnemonic   string            `yaml:"mnemonic"`
	Network    string            `yaml:"network"`
}

type NodesGroup struct {
	Name       string `yaml:"name"`
	NodesCount uint64 `yaml:"nodes_count"`
	FreeCPU    uint64 `yaml:"free_cpu"`
	FreeMRU    uint64 `yaml:"free_mru"`
	FreeSSD    uint64 `yaml:"free_ssd"`
	FreeHDD    uint64 `yaml:"free_hdd"`
	Dedicated  bool   `yaml:"dedicated"`
	Pubip4     bool   `yaml:"pubip4"`
	Pubip6     bool   `yaml:"pubip6"`
	Certified  bool   `yaml:"certified"`
	Regions    string `yaml:"regions"`
	MinBwd     uint64 `yaml:"min_bandwidth_ms"`
}

type Vm struct {
	Name       string `yaml:"name"`
	Count      int    `yaml:"vms_count"`
	Nodegroup  string `yaml:"node_group"`
	FreeCPU    int    `yaml:"cpu"`
	FreeMRU    int    `yaml:"mem"`
	SSHDisks   []Disk `yaml:"ssd"`
	Pubip4     bool   `yaml:"pubip4"`
	Pubip6     bool   `yaml:"pubip6"`
	Flist      string `yaml:"flist"`
	Rootsize   int    `yaml:"root_size"`
	Entrypoint string `yaml:"entry_point"`
	SSHKey     string `yaml:"ssh_key"`
}

type Disk struct {
	Capacity int    `yaml:"capacity"`
	Mount    string `yaml:"mount_point"`
}
