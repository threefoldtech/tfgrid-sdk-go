package types

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
