package config

import (
	"errors"
	"fmt"
	"io"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/types"
	"gopkg.in/yaml.v2"
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

	// ServicesGraph *dependency.DRGraph
	// // Constructed map from config file content to be used to generate deployments
	// DeploymentData map[string]*struct {
	// 	NodeID   uint32
	// 	Services map[string]*Service
	// }
}

// NewConfig creates a new instance of the configuration
func NewConfig() *Config {
	return &Config{
		Networks: make(map[string]types.Network),
		Services: make(map[string]types.Service),
		Volumes:  make(map[string]types.Volume),
	}
}

// LoadConfigFromReader loads the configuration file content from a reader
func (c *Config) LoadConfigFromReader(configFile io.Reader) error {
	content, err := io.ReadAll(configFile)
	if err != nil {
		return fmt.Errorf("failed to read file %w", err)
	}
	err = c.UnmarshalYAML(content)
	if err != nil {
		return fmt.Errorf("failed to unmarshal yaml %w", err)
	}
	return nil
}

// UnmarshalYAML unmarshals the configuration file content and populates the DeploymentData map
func (c *Config) UnmarshalYAML(content []byte) error {
	if err := yaml.Unmarshal(content, c); err != nil {
		return err
	}

	return nil
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
