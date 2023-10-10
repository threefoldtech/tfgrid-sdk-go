package monitor

import (
	"errors"
	"os"
	"strconv"
	"strings"

	env "github.com/hashicorp/go-envparse"
)

var invalidCfgError = errors.New("Invalid or Missing Fields in configration file")

func parseFile(envPath string) (string, error) {
	envContent, err := os.ReadFile(envPath)
	if err != nil {
		return "", err
	}

	return string(envContent), nil
}

func parseMonitor(envContent string) (Monitor, error) {
	mon := Monitor{}

	envMap, err := env.Parse(strings.NewReader(string(envContent)))
	if err != nil {
		return mon, err
	}

	mon.Mnemonic = envMap["MNEMONIC"]
	mon.BotToken = envMap["BOT_TOKEN"]
	mon.Network = envMap["NETWORK"]

	interval, err := strconv.Atoi(envMap["INTERVAL"])
	if err != nil {
		return Monitor{}, invalidCfgError
	}
	mon.interval = interval

	if mon.Mnemonic == "" || mon.Network == "" || mon.BotToken == "" {
		return Monitor{}, invalidCfgError
	}
	return mon, nil
}
