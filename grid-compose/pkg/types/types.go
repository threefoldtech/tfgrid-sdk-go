package types

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type Specs struct {
	Version  string             `yaml:"version"`
	Networks map[string]Network `yaml:"networks"`
	Services map[string]Service `yaml:"services"`
	Storage  map[string]Storage `yaml:"storage"`
}

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
	Environment KVMap        `yaml:"environment"`
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

type KVMap map[string]string

func (m *KVMap) UnmarshalYAML(value *yaml.Node) error {
	var raw []string
	if err := value.Decode(&raw); err != nil {
		return err
	}

	*m = make(map[string]string)
	for _, ele := range raw {
		kv := strings.SplitN(ele, "=", 2)
		if len(kv) != 2 {
			return fmt.Errorf("invalid kvmap format: %s", ele)
		}
		(*m)[kv[0]] = kv[1]
	}

	return nil
}
