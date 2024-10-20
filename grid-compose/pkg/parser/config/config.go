package config

import (
	"errors"
	"fmt"
	"io"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/types"
	"gopkg.in/yaml.v3"
)

var (
	ErrVersionNotSet      = errors.New("version not set")
	ErrNetworkTypeNotSet  = errors.New("network type not set")
	ErrServiceFlistNotSet = errors.New("service flist not set")
	ErrStorageTypeNotSet  = errors.New("storage type not set")
	ErrStorageSizeNotSet  = errors.New("storage size not set")
)

// Config represents the configuration file content
type Config struct {
	Version  string                   `yaml:"version"`
	Networks map[string]types.Network `yaml:"networks"`
	Services map[string]types.Service `yaml:"services"`
	Volumes  map[string]types.Volume  `yaml:"volumes"`
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
	err = yaml.Unmarshal(content, &c)
	if err != nil {
		return fmt.Errorf("failed to unmarshal yaml %w", err)
	}
	return nil
}

// Validate validates the configuration file content
// TODO: Create more validation rules
func (c *Config) Validate() (err error) {
	if c.Version == "" {
		return ErrVersionNotSet
	}

	for name, service := range c.Services {
		if len(name) > 50 {
			return fmt.Errorf("service name %s is too long", name)
		}

		if service.Flist == "" {
			return fmt.Errorf("%w for service %s", ErrServiceFlistNotSet, name)
		}

		if service.Entrypoint == "" {
			return fmt.Errorf("entrypoint not set for service %s", name)
		}

		if service.Resources.Memory != 0 && service.Resources.Memory < 256 {
			return fmt.Errorf("minimum memory resource is 256 megabytes for service %s", name)
		}

		if service.Resources.Rootfs != 0 && service.Resources.Rootfs < 2048 {
			return fmt.Errorf("minimum rootfs resource is 2 gigabytes for service %s", name)
		}

	}

	return nil
}
