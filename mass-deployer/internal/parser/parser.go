package parser

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func ParseConfig(filtName string) (Config, error) {
	if filepath.Ext(filtName) != ".yaml" {
		return Config{}, errors.New("invalid config file extension")
	}

	conf := Config{}

	configFile, err := os.ReadFile(filtName)
	if err != nil {
		return Config{}, err
	}

	err = yaml.Unmarshal(configFile, &conf)
	if err != nil {
		return Config{}, err
	}
	return conf, nil
}
