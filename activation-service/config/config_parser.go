// Package config for config details
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	env "github.com/hashicorp/go-envparse"
)

// Configuration struct to hold app configurations
type Configuration struct {
	ActivationAmount uint64
	SubstrateURL     string
	Mnemonic         string
	KycPublicKey     string
}

// ReadConfFile read configurations of env file
func ReadConfFile(path string) (Configuration, error) {
	config := Configuration{}
	content, err := os.ReadFile(path)
	if err != nil {
		return Configuration{}, fmt.Errorf("failed to open config file: %w", err)
	}

	configMap, err := env.Parse(strings.NewReader(string(content)))
	if err != nil {
		return Configuration{}, fmt.Errorf("failed to load config: %w", err)
	}

	for key, value := range configMap {
		switch key {
		case "URL":
			config.SubstrateURL = value

		case "MNEMONIC":
			config.Mnemonic = value

		case "KYC_PUBLIC_KEY":
			config.KycPublicKey = value

		case "ACTIVATION_AMOUNT":
			amount, err := strconv.Atoi(value)
			if err != nil {
				return Configuration{}, err
			}
			config.ActivationAmount = uint64(amount)

		default:
			return Configuration{}, fmt.Errorf("key %v is invalid", key)
		}
	}

	switch {
	case config.SubstrateURL == "":
		return Configuration{}, fmt.Errorf("URL is missing")
	case config.Mnemonic == "":
		return Configuration{}, fmt.Errorf("MNEMONIC is missing")
	case config.KycPublicKey == "":
		return Configuration{}, fmt.Errorf("KYC_PUBLIC_KEY is missing")
	case config.ActivationAmount == 0:
		config.ActivationAmount = 1000000
	}

	return config, nil
}
