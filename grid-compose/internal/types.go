package internal

import "github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"

type NetworkTypes struct {
	Type string `yaml:"type"`
}

type Storage struct {
	Size string `yaml:"size"`
}

type Service struct {
	Flist        string       `yaml:"flist"`
	Entrypoint   string       `yaml:"entrypoint,omitempty"`
	Environment  []string     `yaml:"environment"`
	Resources    Resources    `yaml:"resources"`
	Volumes      []string     `yaml:"volumes"`
	NetworkTypes []string     `yaml:"network_types"`
	Networks     []string     `yaml:"networks"`
	HealthCheck  *HealthCheck `yaml:"healthcheck,omitempty"`
	DependsOn    []string     `yaml:"depends_on,omitempty"`
	DeployTo     string       `yaml:"deploy_to,omitempty"`
}

type Deployment struct {
	Name    string `yaml:"name"`
	NodeID  uint32 `yaml:"node_id"`
	Network *struct {
		Name string `yaml:"name"`
	} `yaml:"network"`

	Workloads []string
}
type HealthCheck struct {
	Test     []string `yaml:"test"`
	Interval string   `yaml:"interval"`
	Timeout  string   `yaml:"timeout"`
	Retries  uint     `yaml:"retries"`
}

type Resources struct {
	CPU    uint `yaml:"cpu" json:"cpu"`
	Memory uint `yaml:"memory" json:"memory"`
	Rootfs uint `yaml:"ssd" json:"ssd"`
}

type WorkloadData struct {
	Flist           string              `json:"flist"`
	Network         WorkloadDataNetwork `json:"network"`
	ComputeCapacity Resources           `json:"compute_capacity"`
	Size            uint                `json:"size"`
	Mounts          []struct {
		Name       string `json:"name"`
		MountPoint string `json:"mountpoint"`
	} `json:"mounts"`
	Entrypoint string            `json:"entrypoint"`
	Env        map[string]string `json:"env"`
	Corex      bool              `json:"corex"`
}

type WorkloadDataNetwork struct {
	PublicIP   string `json:"public_ip"`
	Planetary  bool   `json:"planetary"`
	Interfaces []struct {
		Network string `json:"network"`
		IP      string `json:"ip"`
	}
}

type Network struct {
	Name         string            `yaml:"name"`
	Description  string            `yaml:"description"`
	Nodes        []uint32          `yaml:"nodes"`
	IPRange      IPNet             `yaml:"range"`
	AddWGAccess  bool              `yaml:"wg"`
	MyceliumKeys map[uint32][]byte `yaml:"mycelium_keys"`
}

type IPNet struct {
	IP   IP     `yaml:"ip"`
	Mask IPMask `yaml:"mask"`
}

type IP struct {
	Type string `yaml:"type"`
	IP   string `yaml:"ip"`
}

type IPMask struct {
	Type string `yaml:"type"`
	Mask string `yaml:"mask"`
}

type DeploymentData struct {
	Vms   []workloads.VM
	Disks []workloads.Disk
}
