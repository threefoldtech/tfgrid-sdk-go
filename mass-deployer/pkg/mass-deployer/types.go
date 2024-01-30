package deployer

import "github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"

// type config contains configuration used to deploy multible groups of vms in batches
// **note: please make sure to run validator (validator.Validate(conf))**
type Config struct {
	NodeGroups []NodesGroup      `yaml:"node_groups" validate:"nonzero" json:"node_groups"`
	Vms        []Vms             `yaml:"vms" validate:"nonzero" json:"vms"`
	SSHKeys    map[string]string `yaml:"ssh_keys" validate:"nonzero" json:"ssh_keys"`
	Mnemonic   string            `yaml:"mnemonic" validate:"nonzero" json:"mnemonic"`
	Network    string            `yaml:"network" validate:"nonzero" json:"network"`
}

type NodesGroup struct {
	Name       string `yaml:"name" validate:"nonzero" json:"name"`
	NodesCount uint64 `yaml:"nodes_count" validate:"nonzero" json:"nodes_count"`
	FreeCPU    uint64 `yaml:"free_cpu" validate:"nonzero,max=32" json:"free_cpu"`
	FreeMRU    uint64 `yaml:"free_mru" validate:"nonzero,min=256,max=262144" json:"free_mru"` // min: 256MB, max: 256 GB
	FreeSRU    uint64 `yaml:"free_ssd" json:"free_ssd"`
	FreeHRU    uint64 `yaml:"free_hdd" json:"free_hdd"`
	Dedicated  bool   `yaml:"dedicated" json:"dedicated"`
	PublicIP4  bool   `yaml:"public_ip4" json:"public_ip4"`
	PublicIP6  bool   `yaml:"public_ip6" json:"public_ip6"`
	Certified  bool   `yaml:"certified" json:"certified"`
	Regions    string `yaml:"regions" json:"regions"`
}

type Vms struct {
	Name       string            `yaml:"name" validate:"nonzero" json:"name"`
	Count      uint64            `yaml:"vms_count" validate:"nonzero" json:"vms_count"`
	Nodegroup  string            `yaml:"node_group" validate:"nonzero" json:"node_group"`
	FreeCPU    uint64            `yaml:"cpu" validate:"nonzero,max=32" json:"cpu"`
	FreeMRU    uint64            `yaml:"mem" validate:"nonzero,min=256,max=262144" json:"mem"` // min: 256MB, max: 256 GB
	SSDDisks   []Disk            `yaml:"ssd" json:"ssd"`
	PublicIP4  bool              `yaml:"public_ip4" json:"public_ip4"`
	PublicIP6  bool              `yaml:"public_ip6" json:"public_ip6"`
	Planetary  bool              `yaml:"planetary" json:"planetary"`
	Flist      string            `yaml:"flist" validate:"nonzero" json:"flist"`
	Rootsize   uint64            `yaml:"root_size" validate:"max=10240" json:"root_size"` // max 10 TB
	Entrypoint string            `yaml:"entry_point" validate:"nonzero" json:"entry_point"`
	SSHKey     string            `yaml:"ssh_key" validate:"nonzero" json:"ssh_key"`
	EnvVars    map[string]string `yaml:"env_vars" json:"env_vars"`
}

type Disk struct {
	Size  uint64 `yaml:"size" validate:"nonzero,min=15" json:"size"` // min 15 GB
	Mount string `yaml:"mount_point" validate:"nonzero" json:"mount_point"`
}

type groupDeploymentsInfo struct {
	vmDeployments      []*workloads.Deployment
	networkDeployments []*workloads.ZNet
	deploymentsInfo    []vmDeploymentInfo
}

type vmDeploymentInfo struct {
	nodeID         uint32
	vmName         string
	deploymentName string
}

type vmOutput struct {
	Name      string
	PublicIP4 string
	PublicIP6 string
	YggIP     string
	IP        string
	Mounts    []workloads.Mount
}
