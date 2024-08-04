package pkg

type Network struct {
	Type string `yaml:"type"`
}

type Storage struct {
	Type string `yaml:"type"`
	Size string `yaml:"size"`
}

type Service struct {
	Flist       string       `yaml:"flist"`
	Entrypoint  string       `yaml:"entrypoint,omitempty"`
	Environment []string     `yaml:"environment"`
	Resources   Resources    `yaml:"resources"`
	NodeID      uint         `yaml:"node_id"`
	Volumes     []string     `yaml:"volumes"`
	Networks    []string     `yaml:"networks"`
	HealthCheck *HealthCheck `yaml:"healthcheck,omitempty"`
	DependsOn   []string     `yaml:"depends_on,omitempty"`
}

type HealthCheck struct {
	Test     []string `yaml:"test"`
	Interval string   `yaml:"interval"`
	Timeout  string   `yaml:"timeout"`
	Retries  int      `yaml:"retries"`
}

type Resources struct {
	CPU    uint `yaml:"cpu" json:"cpu"`
	Memory uint `yaml:"memory" json:"memory"`
	SSD    uint `yaml:"ssd" json:"ssd"`
	HDD    uint `yaml:"hdd" json:"hdd"`
}

type WorkloadData struct {
	Flist           string    `json:"flist"`
	Network         Net       `json:"network"`
	ComputeCapacity Resources `json:"compute_capacity"`
	Size            int       `json:"size"`
	Mounts          []struct {
		Name       string `json:"name"`
		MountPoint string `json:"mountpoint"`
	} `json:"mounts"`
	Entrypoint string            `json:"entrypoint"`
	Env        map[string]string `json:"env"`
	Corex      bool              `json:"corex"`
}

type Net struct {
	PublicIP   string `json:"public_ip"`
	Planetary  bool   `json:"planetary"`
	Interfaces []struct {
		Network string `json:"network"`
		IP      string `json:"ip"`
	}
}
