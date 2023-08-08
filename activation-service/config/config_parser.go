// Package config for config details
package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/cosmos/go-bip39"
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
			err := validateWssURL(value)
			if err != nil {
				return Configuration{}, fmt.Errorf("substrate url '%s' is invalid: %v", value, err)
			}
			config.SubstrateURL = value

		case "MNEMONIC":
			valid := bip39.IsMnemonicValid(value)
			if !valid {
				return Configuration{}, errors.New("mnemonic is invalid")
			}
			config.Mnemonic = value

		case "KYC_PUBLIC_KEY":
			if len(strings.TrimSpace(value)) == 0 {
				return Configuration{}, errors.New("KYC_PUBLIC_KEY is invalid")
			}
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
		config.ActivationAmount = 1 // 1 TFT
	}

	return config, nil
}

func validateWssURL(url string) error {
	if len(strings.TrimSpace(url)) == 0 {
		return errors.New("substrate url is required")
	}

	alphaOnly := regexp.MustCompile(`^wss:\/\/[a-z0-9]+\.[a-z0-9]\/?([^\s<>\#%"\,\{\}\\|\\\^\[\]]+)?$`)
	if !alphaOnly.MatchString(url) {
		return fmt.Errorf("wss url '%s' is invalid", url)
	}

	return nil
}
