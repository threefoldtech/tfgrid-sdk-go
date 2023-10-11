package monitor

import (
	"errors"
	"os"
	"strconv"

	env "github.com/hashicorp/go-envparse"
)

// configrations parsed from the env file
type Config struct {
	mnemonic string `env:"MNEMONIC"`
	botToken string `env:"BOT_TOKEN"`
	network  string `env:"NETWORK"`
	interval int    `env:"INTERVAL"`
}

func ParseConfig(envPath string) (Config, error) {
	conf := Config{}

	envFile, err := os.Open(envPath)
	if err != nil {
		return conf, errors.New("failed to open config")
	}

	envMap, err := env.Parse(envFile)
	if err != nil {
		return conf, err
	}

	conf.mnemonic = envMap["MNEMONIC"]
	conf.botToken = envMap["BOT_TOKEN"]
	conf.network = envMap["NETWORK"]
	interval, err := strconv.Atoi(envMap["INTERVAL"])
	if err != nil {
		return Config{}, errors.New("invalid or missing 'INTERVAL' field")
	}
	conf.interval = interval

	if conf.mnemonic == "" {
		return Config{}, errors.New("missing 'MNEMONIC' field")
	}

	if conf.network == "" {
		return Config{}, errors.New("missing 'NETWORK' field")
	}

	if conf.botToken == "" {
		return Config{}, errors.New("missing 'BOT_TOKEN' field")
	}

	return conf, nil
}
