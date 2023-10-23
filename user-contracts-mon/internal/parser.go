package monitor

import (
	"errors"
	"os"
	"strconv"

	env "github.com/hashicorp/go-envparse"
)

// configrations parsed from the env file
type Config struct {
	botToken string `env:"BOT_TOKEN"`
	interval int    `env:"INTERVAL"`
}

func ParseConfig(envPath string) (Config, error) {
	conf := Config{}

	envFile, err := os.Open(envPath)
	if err != nil {
		return conf, errors.New("failed to open config file " + envPath)
	}

	envMap, err := env.Parse(envFile)
	if err != nil {
		return conf, err
	}

	interval, err := strconv.Atoi(envMap["INTERVAL"])
	if err != nil {
		return Config{}, errors.New("invalid or missing 'INTERVAL' field")
	}
	conf.interval = interval

	conf.botToken = envMap["BOT_TOKEN"]
	if conf.botToken == "" {
		return Config{}, errors.New("missing 'BOT_TOKEN' field")
	}

	return conf, nil
}
