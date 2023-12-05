// Package parser for parsing cmd configs
package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/models"
	"gopkg.in/yaml.v3"
)

// ReadFile reads a file and returns its contents
func ReadFile(path string) ([]byte, string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return []byte{}, "", err
	}

	return content, filepath.Ext(path)[1:], nil
}

// ParseIntoConfig parses the configuration
func ParseIntoConfig(content []byte, format string) (models.Config, error) {
	input := models.Config{}

	var err error
	switch {
	case strings.ToLower(format) == "json":
		err = json.Unmarshal(content, &input)
		// yaml will be the format
	case strings.ToLower(format) == "yml" || strings.ToLower(format) == "yaml":
		err = yaml.Unmarshal(content, &input)
	case strings.ToLower(format) == "toml":
		err = toml.Unmarshal(content, &input)
	default:
		err = fmt.Errorf("invalid config file format '%s'", format)
	}

	if err != nil {
		return models.Config{}, err
	}

	for _, in := range input.IncludedNodes {
		for _, ex := range input.ExcludedNodes {
			if ex == in {
				return models.Config{}, fmt.Errorf("cannot include and exclude the same node '%d'", in)
			}
		}
	}

	return input, nil
}
