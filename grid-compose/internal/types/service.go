package types

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/pkg/parser"
	"gopkg.in/yaml.v3"
)

// Service represents a service in the deployment
type Service struct {
	Flist       string       `yaml:"flist"`
	Entrypoint  string       `yaml:"entrypoint,omitempty"`
	Environment KVMap        `yaml:"environment"`
	Resources   Resources    `yaml:"resources"`
	Volumes     []string     `yaml:"volumes"`
	NodeID      uint32       `yaml:"node_id"`
	IPTypes     []string     `yaml:"ip_types"`
	Network     string       `yaml:"network"`
	HealthCheck *HealthCheck `yaml:"healthcheck,omitempty"`
	DependsOn   []string     `yaml:"depends_on,omitempty"`
}

// KVMap represents a key-value map and implements the Unmarshaler interface
type KVMap map[string]string

// UnmarshalYAML unmarshals a YAML node into a KVMap
func (m *KVMap) UnmarshalYAML(value *yaml.Node) error {
	var raw []string
	if err := value.Decode(&raw); err != nil {
		return err
	}

	*m = make(map[string]string)
	for _, ele := range raw {
		kv := strings.SplitN(ele, "=", 2)
		if len(kv) != 2 {
			return fmt.Errorf("invalid kvmap format %s", ele)
		}
		(*m)[kv[0]] = kv[1]
	}

	return nil
}

// Resources represents the resources required by the service
type Resources struct {
	CPU    uint16 `yaml:"cpu"`
	Memory uint64 `yaml:"memory"`
	Rootfs uint64 `yaml:"rootfs"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface for custom unmarshalling.
func (r *Resources) UnmarshalYAML(value *yaml.Node) error {
	for i := 0; i < len(value.Content); i += 2 {
		key := value.Content[i].Value
		val := value.Content[i+1].Value

		switch key {
		case "cpu":
			cpuVal, err := strconv.ParseUint(val, 10, 16)
			if err != nil {
				return fmt.Errorf("invalid cpu value %w", err)
			}
			r.CPU = uint16(cpuVal)
		case "memory":
			memVal, err := parser.ParseStorage(val)
			if err != nil {
				return fmt.Errorf("invalid memory value %w", err)
			}
			r.Memory = memVal

		case "rootfs":
			rootfsVal, err := parser.ParseStorage(val)
			if err != nil {
				return fmt.Errorf("invalid rootfs value %w", err)
			}
			r.Rootfs = rootfsVal
		}
	}

	return nil
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
