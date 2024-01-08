package parser

import (
	"encoding/json"
	"errors"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func ParseConfig(filtName string) (Config, error) {
	str := strings.Split(filtName, ".")
	if len(str) < 2 {
		return Config{}, errors.New("invalid config file extension")
	}

	conf := Config{}
	if str[len(str)-1] == "json" {
		configFile, err := os.Open(filtName)
		if err != nil {
			return Config{}, err
		}

		jsonParser := json.NewDecoder(configFile)
		if err = jsonParser.Decode(&conf); err != nil {
			return Config{}, err
		}
		return conf, nil

	} else if str[len(str)-1] == "yaml" {
		configFile, err := os.ReadFile(filtName)
		if err != nil {
			return Config{}, err
		}

		err = yaml.Unmarshal(configFile, &conf)
		if err != nil {
			return Config{}, err
		}
		return Config{}, nil
	}

	return Config{}, errors.New("invalid config file extension")
}
