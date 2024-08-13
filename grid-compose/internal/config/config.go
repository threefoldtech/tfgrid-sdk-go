package config

import (
	"errors"
	"fmt"
	"io"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/types"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/utils"

	"gopkg.in/yaml.v3"
)

var (
	ErrVersionNotSet               = errors.New("version not set")
	ErrNetworkTypeNotSet           = errors.New("network type not set")
	ErrServiceFlistNotSet          = errors.New("service flist not set")
	ErrServiceCPUResourceNotSet    = errors.New("service cpu resource not set")
	ErrServiceMemoryResourceNotSet = errors.New("service memory resource not set")
	ErrStorageTypeNotSet           = errors.New("storage type not set")
	ErrStorageSizeNotSet           = errors.New("storage size not set")
)

// Config represents the configuration file content
type Config struct {
	Version  string                   `yaml:"version"`
	Networks map[string]types.Network `yaml:"networks"`
	Services map[string]types.Service `yaml:"services"`
	Volumes  map[string]types.Volume  `yaml:"volumes"`

	// Constructed map from config file content to be used to generate deployments
	DeploymentData map[string]*struct {
		Services []*types.Service
		NodeID   uint32
	}
}

// NewConfig creates a new instance of the configuration
func NewConfig() *Config {
	return &Config{
		Networks: make(map[string]types.Network),
		Services: make(map[string]types.Service),
		Volumes:  make(map[string]types.Volume),
	}
}

// ValidateConfig validates the configuration file content
// TODO: Create more validation rules
func (c *Config) ValidateConfig() (err error) {
	if c.Version == "" {
		return ErrVersionNotSet
	}

	for name, service := range c.Services {
		if service.Flist == "" {
			return fmt.Errorf("%w for service %s", ErrServiceFlistNotSet, name)
		}

		if service.Resources.CPU == 0 {
			return fmt.Errorf("%w for service %s", ErrServiceCPUResourceNotSet, name)
		}

		if service.Resources.Memory == 0 {
			return fmt.Errorf("%w for service %s", ErrServiceMemoryResourceNotSet, name)
		}
	}

	return nil
}

// LoadConfigFromReader loads the configuration file content from a reader
func (c *Config) LoadConfigFromReader(configFile io.Reader) error {
	content, err := io.ReadAll(configFile)
	if err != nil {
		return fmt.Errorf("failed to read file %w", err)
	}

	if err := c.UnmarshalYAML(content); err != nil {
		return err
	}

	return nil
}

// UnmarshalYAML unmarshals the configuration file content and populates the DeploymentData map
func (c *Config) UnmarshalYAML(content []byte) error {
	if err := yaml.Unmarshal(content, c); err != nil {
		return err
	}

	defaultNetName := utils.GenerateDefaultNetworkName(c.Services)
	c.DeploymentData = make(map[string]*struct {
		Services []*types.Service
		NodeID   uint32
	})

	for serviceName, service := range c.Services {
		svc := service
		var netName string
		if svc.Network == "" {
			netName = defaultNetName
		} else {
			netName = svc.Network
		}

		if _, ok := c.DeploymentData[netName]; !ok {
			c.DeploymentData[netName] = &struct {
				Services []*types.Service
				NodeID   uint32
			}{
				Services: make([]*types.Service, 0),
				NodeID:   svc.NodeID,
			}
		}

		if c.DeploymentData[netName].NodeID == 0 && svc.NodeID != 0 {
			c.DeploymentData[netName].NodeID = svc.NodeID
		}

		if svc.NodeID != 0 && svc.NodeID != c.DeploymentData[netName].NodeID {
			return fmt.Errorf("service name %s node_id %d should be the same for all or some or left blank for services in the same network", serviceName, svc.NodeID)
		}

		svc.Name = serviceName

		c.DeploymentData[netName].Services = append(c.DeploymentData[netName].Services, &svc)
	}

	return nil
}
