package config

import (
	"fmt"
	"io"

	types "github.com/threefoldtech/tfgrid-sdk-go/grid-compose/pkg"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Version     string                      `yaml:"version"`
	Networks    map[string]types.Network    `yaml:"networks"`
	Services    map[string]types.Service    `yaml:"services"`
	Storage     map[string]types.Storage    `yaml:"storage"`
	Deployments map[string]types.Deployment `yaml:"deployments"`
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

	if err := c.UnmarshalYAML(content); err != nil {
		return fmt.Errorf("failed to parse file %w", err)
	}

	return nil
}

func (c *Config) UnmarshalYAML(content []byte) error {
	if err := yaml.Unmarshal(content, c); err != nil {
		return err
	}

	for serviceName, service := range c.Services {
		deployTo := service.DeployTo

		if deployment, exists := c.Deployments[deployTo]; exists {
			if deployment.Workloads == nil {
				deployment.Workloads = make([]string, 0)
			}
			deployment.Workloads = append(deployment.Workloads, serviceName)

			c.Deployments[deployTo] = deployment
		}
	}

	return nil
}
