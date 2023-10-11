package monitor

import (
	"errors"
	"io"
	"os"
	"strconv"
	"strings"

	env "github.com/hashicorp/go-envparse"
)

var invalidCfgError = errors.New("Invalid or Missing Fields in configration file")

func readFile(envPath string) (io.Reader, error) {
	envFile, err := os.Open(envPath)
	if err != nil {
		return strings.NewReader(""), err
	}

	return envFile, nil
}

func parseMonitor(envFile io.Reader) (Monitor, error) {
	mon := Monitor{}

	envMap, err := env.Parse(envFile)
	if err != nil {
		return mon, err
	}

	mon.Mnemonic = envMap["MNEMONIC"]
	mon.BotToken = envMap["BOT_TOKEN"]
	mon.Network = envMap["NETWORK"]

	interval, err := strconv.Atoi(envMap["INTERVAL"])
	if err != nil {
		return Monitor{}, errors.New("invalid or missing 'INTERVAL' field")
	}
	mon.interval = interval

	if mon.Mnemonic == "" {
		return Monitor{}, errors.New("missing 'MNEMONIC' field")
	}

	if mon.Network == "" {
		return Monitor{}, errors.New("missing 'NETWORK' field")
	}

	if mon.BotToken == "" {
		return Monitor{}, errors.New("missing 'BOT_TOKEN' field")
	}

	return mon, nil
}
