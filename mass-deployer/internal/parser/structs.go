package parser

type Config struct {
	NodeGroups []NodesGroup `yaml:"node_groups" json:"node_groups"`
	Vms        []Vm         `yaml:"vms" json:"vms"`
	SSHKey     string       `yaml:"sshkey" json:"sshkey"`
	Mnemonic   string       `yaml:"mnemonic" json:"mnemonic"`
	Network    string       `yaml:"network" json:"network"`
}

type NodesGroup struct {
	Name              string `yaml:"name" json:"name"`
	NodesCount        uint64 `yaml:"nodes_count" json:"nodes_count"`
	FreeCPU           uint64 `yaml:"free_cpu" json:"free_cpu"`
	FreeMRU           uint64 `yaml:"free_mru" json:"free_mru"`
	FreeSSD           uint64 `yaml:"free_ssd" json:"free_ssd"`
	FreeHDD           uint64 `yaml:"free_hdd" json:"free_hdd"`
	Dedicated         bool   `yaml:"dedicated" json:"dedicated"`
	Pubip4            bool   `yaml:"pubip4" json:"pubip4"`
	Pubip6            bool   `yaml:"pubip6" json:"pubip6"`
	CertificationType string `yaml:"certification_type" json:"certification_type"`
	Region            string `yaml:"region" json:"region"`
	MinBwd            uint64 `yaml:"min_bandwidth_ms" json:"min_bandwidth_ms"`
}

type Vm struct {
	Name        string `yaml:"name" json:"name"`
	Nodegroup   string `yaml:"nodegroup" json:"nodegroup"`
	FreeCPU     int    `yaml:"nrcpu" json:"nrcpu"`
	FreeMRU     int    `yaml:"mem" json:"mem"`
	Disk        Disk   `yaml:"ssd" json:"ssd"`
	HDDAttached bool   `yaml:"hdd" json:"hdd"`
	Pubip4      bool   `yaml:"pubip4" json:"pubip4"`
	Pubip6      bool   `yaml:"pubip6" json:"pubip6"`
	Flist       string `yaml:"flist" json:"flist"`
	Rootsize    int    `yaml:"rootsize" json:"rootsize"`
	Entrypoint  string `yaml:"entrypoint" json:"entrypoint"`
}

type Disk struct {
	Capacity uint64
	Mount    string
}
