package parser

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/cosmos/go-bip39"
	env "github.com/hashicorp/go-envparse"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/internal"
	"github.com/vedhavyas/go-subkey"
	"gopkg.in/yaml.v3"
)

// ReadFile reads a file and returns its contents
func ReadFile(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
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

// ParseEnv parses content to farmerbot environment vars
func ParseEnv(content string) (network string, mnemonicOrSeed string, err error) {
	envMap, err := env.Parse(strings.NewReader(content))
	if err != nil {
		return
	}

	for key, value := range envMap {
		switch key {
		case "NETWORK":
			network = value

		case "MNEMONIC_OR_SEED":
			mnemonicOrSeed = value

		default:
			return "", "", fmt.Errorf("invalid key '%s'", key)
		}
	}

	switch {
	case network == "":
		network = internal.MainNetwork
	case mnemonicOrSeed == "":
		return "", "", fmt.Errorf("MNEMONIC_OR_SEED is required")
	}

	if !slices.Contains([]string{internal.DevNetwork, internal.QaNetwork, internal.TestNetwork, internal.MainNetwork}, network) {
		err = fmt.Errorf("network must be one of %s, %s, %s, and %s not '%s'", internal.DevNetwork, internal.QaNetwork, internal.TestNetwork, internal.MainNetwork, network)
		return
	}

	if _, ok := subkey.DecodeHex(mnemonicOrSeed); !bip39.IsMnemonicValid(mnemonicOrSeed) && !ok {
		return "", "", fmt.Errorf("invalid seed or mnemonic input '%s'", mnemonicOrSeed)
	}

	return
}
