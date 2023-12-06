package parser

import (
	"fmt"
	"os"

	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/internal"
	"gopkg.in/yaml.v3"
)

// ReadFile reads a file and returns its contents
func ReadFile(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return []byte{}, err
	}

	return content, nil
}

// ParseIntoConfig parses the configuration
func ParseIntoConfig(content []byte) (internal.Config, error) {
	input := internal.Config{}

	err := yaml.Unmarshal(content, &input)
	if err != nil {
		return internal.Config{}, err
	}

	for _, in := range input.IncludedNodes {
		for _, ex := range input.ExcludedNodes {
			if ex == in {
				return internal.Config{}, fmt.Errorf("cannot include and exclude the same node '%d'", in)
			}
		}
	}

	return input, nil
}
