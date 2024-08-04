package config

import (
	"fmt"
	"io"

	types "github.com/threefoldtech/tfgrid-sdk-go/grid-compose/pkg"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Version     string                   `yaml:"version"`
	Networks    map[string]types.Network `yaml:"networks"`
	Services    map[string]types.Service `yaml:"services"`
	Storage     map[string]types.Storage `yaml:"storage"`
	ProjectName string                   `yaml:"project_name"`
}

func NewConfig() *Config {
	return &Config{
		Networks: make(map[string]types.Network),
		Services: make(map[string]types.Service),
		Storage:  make(map[string]types.Storage),
	}
}

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

	for name, storage := range c.Storage {
		if storage.Type == "" {
			return fmt.Errorf("%w for storage %s", ErrStorageTypeNotSet, name)
		}

		if storage.Size == "" {
			return fmt.Errorf("%w for storage %s", ErrStorageSizeNotSet, name)
		}
	}

	return nil
}

func (c *Config) LoadConfigFromReader(configFile io.Reader) error {
	content, err := io.ReadAll(configFile)
	if err != nil {
		return fmt.Errorf("failed to read file %w", err)
	}

	if err := yaml.Unmarshal(content, &c); err != nil {
		return fmt.Errorf("failed to parse file %w", err)
	}

	return nil
}
