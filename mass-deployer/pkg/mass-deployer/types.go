package deployer

import "github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"

// type config contains configuration used to deploy multible groups of vms in batches
// **note: please make sure to run validator (validator.Validate(conf))**
type Config struct {
	NodeGroups []NodesGroup      `yaml:"node_groups" validate:"nonzero"`
	Vms        []Vms             `yaml:"vms" validate:"nonzero"`
	SSHKeys    map[string]string `yaml:"ssh_keys" validate:"nonzero"`
	Mnemonic   string            `yaml:"mnemonic" validate:"nonzero"`
	Network    string            `yaml:"network" validate:"nonzero"`
}

type NodesGroup struct {
	Name       string `yaml:"name" validate:"nonzero"`
	NodesCount uint64 `yaml:"nodes_count" validate:"nonzero"`
	FreeCPU    uint64 `yaml:"free_cpu" validate:"nonzero"`
	FreeMRU    uint64 `yaml:"free_mru" validate:"nonzero"`
	FreeSRU    uint64 `yaml:"free_ssd"`
	FreeHRU    uint64 `yaml:"free_hdd"`
	Dedicated  bool   `yaml:"dedicated"`
	PublicIP4  bool   `yaml:"public_ip4"`
	PublicIP6  bool   `yaml:"public_ip6"`
	Certified  bool   `yaml:"certified"`
	Regions    string `yaml:"regions"`
}

type Vms struct {
	Name       string `yaml:"name" validate:"nonzero"`
	Count      uint64 `yaml:"vms_count" validate:"nonzero"`
	Nodegroup  string `yaml:"node_group" validate:"nonzero"`
	FreeCPU    uint64 `yaml:"cpu" validate:"nonzero,max=32"`
	FreeMRU    uint64 `yaml:"mem" validate:"nonzero,min=256,max=262144"` // min: 256MB, max: 256 GB
	SSDDisks   []Disk `yaml:"ssd"`
	PublicIP4  bool   `yaml:"public_ip4"`
	PublicIP6  bool   `yaml:"public_ip6"`
	Planetary  bool   `yaml:"planetary"`
	Flist      string `yaml:"flist" validate:"nonzero"`
	Rootsize   uint64 `yaml:"root_size" validate:"max=10240"` // max 10 TB
	Entrypoint string `yaml:"entry_point" validate:"nonzero"`
	SSHKey     string `yaml:"ssh_key" validate:"nonzero"`
}

type Disk struct {
	Size  uint64 `yaml:"size" validate:"nonzero,min=15"` // min 15 GB
	Mount string `yaml:"mount_point" validate:"nonzero"`
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
