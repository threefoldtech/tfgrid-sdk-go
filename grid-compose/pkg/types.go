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
	CPU    uint `yaml:"cpu"`
	Memory uint `yaml:"memory"`
	SSD    uint `yaml:"ssd"`
	HDD    uint `yaml:"hdd"`
}
