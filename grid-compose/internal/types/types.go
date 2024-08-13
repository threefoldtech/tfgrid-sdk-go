package types

// Service represents a service in the deployment
type Service struct {
	Flist       string       `yaml:"flist"`
	Entrypoint  string       `yaml:"entrypoint,omitempty"`
	Environment []string     `yaml:"environment"`
	Resources   Resources    `yaml:"resources"`
	Volumes     []string     `yaml:"volumes"`
	NodeID      uint32       `yaml:"node_id"`
	IPTypes     []string     `yaml:"ip_types"`
	Network     string       `yaml:"network"`
	HealthCheck *HealthCheck `yaml:"healthcheck,omitempty"`
	DependsOn   []string     `yaml:"depends_on,omitempty"`

	Name string
}

// Resources represents the resources required by the service
type Resources struct {
	CPU    uint64 `yaml:"cpu"`
	Memory uint64 `yaml:"memory"`
	Rootfs uint64 `yaml:"rootfs"`
}

// HealthCheck represents the health check configuration for the service
type HealthCheck struct {
	Test     []string `yaml:"test"`
	Interval string   `yaml:"interval"`
	Timeout  string   `yaml:"timeout"`
	Retries  uint     `yaml:"retries"`
}

// Volume represents a volume in the deployment
type Volume struct {
	MountPoint string `yaml:"mountpoint"`
	Size       string `yaml:"size"`
}

// Network represents the network configuration
type Network struct {
	Name         string            `yaml:"name"`
	Description  string            `yaml:"description"`
	IPRange      IPNet             `yaml:"range"`
	AddWGAccess  bool              `yaml:"wg"`
	MyceliumKeys map[uint32][]byte `yaml:"mycelium_keys"`
}

// IPNet represents the IP and mask of a network
type IPNet struct {
	IP   IP     `yaml:"ip"`
	Mask IPMask `yaml:"mask"`
}

// IP represents the IP of a network
type IP struct {
	Type string `yaml:"type"`
	IP   string `yaml:"ip"`
}

// IPMask represents the mask of a network
type IPMask struct {
	Type string `yaml:"type"`
	Mask string `yaml:"mask"`
}
