package deployer

import "github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"

// type config contains configuration used to deploy multiple groups of vms in batches
// **note: please make sure to run validator (validator.Validate(conf))**
type Config struct {
	NodeGroups []NodesGroup      `yaml:"node_groups" validate:"required,unique=Name,min=1,dive,required" json:"node_groups"`
	Vms        []Vms             `yaml:"vms" validate:"required,min=1,dive,required" json:"vms"`
	SSHKeys    map[string]string `yaml:"ssh_keys" validate:"required" json:"ssh_keys"`
	Mnemonic   string            `yaml:"mnemonic" validate:"required" json:"mnemonic"`
	Network    string            `yaml:"network" validate:"required" json:"network"`
	MaxRetries uint64            `yaml:"max_retries" json:"max_retries"`
}

type NodesGroup struct {
	Name       string  `yaml:"name" validate:"required" json:"name"`
	NodesCount uint64  `yaml:"nodes_count" validate:"required" json:"nodes_count"`
	FreeCPU    uint64  `yaml:"free_cpu" validate:"required,max=32" json:"free_cpu"`
	FreeMRU    float32 `yaml:"free_mru" validate:"required,min=0.25,max=256" json:"free_mru"` // min: 0.25 GB, max: 256 GB
	FreeSRU    uint64  `yaml:"free_ssd" json:"free_ssd"`
	FreeHRU    uint64  `yaml:"free_hdd" json:"free_hdd"`
	Dedicated  bool    `yaml:"dedicated" json:"dedicated"`
	PublicIP4  bool    `yaml:"public_ip4" json:"public_ip4"`
	PublicIP6  bool    `yaml:"public_ip6" json:"public_ip6"`
	Certified  bool    `yaml:"certified" json:"certified"`
	Region     string  `yaml:"region" json:"region"`
}

type Vms struct {
	Name       string            `yaml:"name" validate:"required" json:"name"`
	Count      uint64            `yaml:"vms_count" validate:"required" json:"vms_count"`
	NodeGroup  string            `yaml:"node_group" validate:"required" json:"node_group"`
	FreeCPU    uint8             `yaml:"cpu" validate:"required,max=32" json:"cpu"`
	FreeMRU    float32           `yaml:"mem" validate:"required,min=0.25,max=256" json:"mem"` // min: 0.25 GB, max: 256 GB
	SSDDisks   []Disk            `yaml:"ssd" json:"ssd"`
	Volumes    []Volume          `yaml:"volume" json:"volume"`
	PublicIP4  bool              `yaml:"public_ip4" json:"public_ip4"`
	PublicIP6  bool              `yaml:"public_ip6" json:"public_ip6"`
	Ygg        bool              `yaml:"ygg_ip" json:"ygg_ip"`
	Mycelium   bool              `yaml:"mycelium_ip" json:"mycelium_ip"`
	Flist      string            `yaml:"flist" validate:"required" json:"flist"`
	RootSize   uint64            `yaml:"root_size" validate:"max=10240" json:"root_size"` // max 10 TB
	Entrypoint string            `yaml:"entry_point" validate:"required" json:"entry_point"`
	SSHKey     string            `yaml:"ssh_key" validate:"required" json:"ssh_key"`
	EnvVars    map[string]string `yaml:"env_vars" json:"env_vars"`
	WireGuard  bool              `yaml:"wireguard" json:"wireguard"`
}

type Disk struct {
	Size  uint64 `yaml:"size" validate:"required,min=15" json:"size"` // min 15 GB
	Mount string `yaml:"mount_point" validate:"required" json:"mount_point"`
}

type Volume struct {
	Size  uint64 `yaml:"size" validate:"required,min=15" json:"size"` // min 15 GB
	Mount string `yaml:"mount_point" validate:"required" json:"mount_point"`
}

type groupDeploymentsInfo struct {
	vmDeployments      []*workloads.Deployment
	networkDeployments []workloads.Network
}

type vmOutput struct {
	Name        string            `yaml:"name" json:"name"`
	NetworkName string            `yaml:"network_name" json:"network_name"`
	PublicIP4   string            `yaml:"public_ip4" json:"public_ip4"`
	PublicIP6   string            `yaml:"public_ip6" json:"public_ip6"`
	YggIP       string            `yaml:"ygg_ip" json:"ygg_ip"`
	MyceliumIP  string            `yaml:"mycelium_ip" json:"mycelium_ip"`
	IP          string            `yaml:"ip" json:"ip"`
	Mounts      []workloads.Mount `yaml:"mounts" json:"mounts"`
	NodeID      uint32            `yaml:"node_id" json:"node_id"`
	ContractID  uint64            `yaml:"contract_id" json:"contract_id"`
}
