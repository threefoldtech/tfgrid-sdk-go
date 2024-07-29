package parser

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/cosmos/go-bip39"
	env "github.com/hashicorp/go-envparse"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/internal"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
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
func ParseEnv(content string) (network string, mnemonicOrSeed string, keyType string, err error) {
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

		case "KEY_TYPE":
			keyType = value

		default:
			return "", "", "", fmt.Errorf("invalid key '%s'", key)
		}
	}

	if len(strings.TrimSpace(network)) == 0 {
		network = internal.MainNetwork
	}

	if len(strings.TrimSpace(mnemonicOrSeed)) == 0 {
		return "", "", "", fmt.Errorf("MNEMONIC_OR_SEED is required")
	}

	if !slices.Contains([]string{internal.DevNetwork, internal.QaNetwork, internal.TestNetwork, internal.MainNetwork}, network) {
		err = fmt.Errorf("network must be one of %s, %s, %s, and %s not '%s'", internal.DevNetwork, internal.QaNetwork, internal.TestNetwork, internal.MainNetwork, network)
		return
	}

	if _, ok := subkey.DecodeHex(mnemonicOrSeed); !bip39.IsMnemonicValid(mnemonicOrSeed) && !ok {
		return "", "", "", fmt.Errorf("invalid seed or mnemonic input '%s'", mnemonicOrSeed)
	}

	if len(strings.TrimSpace(keyType)) == 0 {
		keyType = peer.KeyTypeSr25519
	}

	if keyType != peer.KeyTypeEd25519 && keyType != peer.KeyTypeSr25519 {
		return "", "", "", fmt.Errorf("invalid key type input %q", keyType)
	}

	return
}

func ValidateInput(input *internal.Config, sub internal.Substrate) error {
	nodes, err := sub.GetNodes(input.FarmID)
	if err != nil {
		return fmt.Errorf("couldn't retrieve node for %d : %v", input.FarmID, err)
	}
	nodesMap := make(map[uint32]bool)
	for _, node := range nodes {
		nodesMap[node] = true
	}

	//validate included nodes
	for _, includedNode := range input.IncludedNodes {
		if _, ok := nodesMap[includedNode]; !ok {
			return fmt.Errorf("included node with id %d doesn't exist in the farm", includedNode)
		}
	}
	//validate excluded nodes
	for _, excludedNode := range input.ExcludedNodes {
		if _, ok := nodesMap[excludedNode]; !ok {
			return fmt.Errorf("excluded node with id %d doesn't exist in the farm", excludedNode)
		}
		index := slices.Index(input.IncludedNodes, excludedNode)
		if index >= 0 {
			return fmt.Errorf("cannot include and exclude the same node %d", excludedNode)
		}
	}

	//validate priority nodes
	for _, priorityNode := range input.PriorityNodes {
		index := slices.Index(input.IncludedNodes, priorityNode)
		if index < 0 {
			return fmt.Errorf("priority node with id %d doesn't exist included nodes", priorityNode)
		}
	}
	return nil
}
