package parser

import (
	"fmt"
	"io"
	"strings"

	"github.com/rs/zerolog/log"
	deployer "github.com/threefoldtech/tfgrid-sdk-go/mass-deployer/pkg/mass-deployer"
	"gopkg.in/validator.v2"
	"gopkg.in/yaml.v3"
)

const (
	mnemonicKey = "MNEMONIC"
	networkKey  = "NETWORK"
)

func ParseConfig(file io.Reader) (deployer.Config, error) {
	conf := deployer.Config{}
	nodeGroupsNames := []string{}

	configFile, err := io.ReadAll(file)
	if err != nil {
		return deployer.Config{}, fmt.Errorf("failed to read the config file")
	}

	err = yaml.Unmarshal(configFile, &conf)
	if err != nil {
		return deployer.Config{}, err
	}

	conf.Mnemonic, err = getValueOrEnv(conf.Mnemonic, mnemonicKey)
	if err != nil {
		return deployer.Config{}, err
	}

	conf.Network, err = getValueOrEnv(conf.Network, networkKey)
	if err != nil {
		return deployer.Config{}, err
	}

	log.Info().Msg("validating configuration file")

	if err := validator.Validate(conf); err != nil {
		return deployer.Config{}, fmt.Errorf("failed to validate config: %+v", err)
	}

	if err := validateNetwork(conf.Network); err != nil {
		return deployer.Config{}, err
	}

	if err := validateMnemonic(conf.Mnemonic); err != nil {
		return deployer.Config{}, err
	}

	for _, nodeGroup := range conf.NodeGroups {
		name := strings.TrimSpace(nodeGroup.Name)
		nodeGroupsNames = append(nodeGroupsNames, name)
	}

	if err := validateVMs(conf.Vms, nodeGroupsNames, conf.SSHKeys); err != nil {
		return deployer.Config{}, err
	}

	log.Info().Msg("done validating configuration file")
	return conf, nil
}

func getValueOrEnv(value, envKey string) (string, error) {
	envKey = strings.ToUpper(envKey)
	if strings.TrimSpace(value) == "" {
		if strings.TrimSpace(value) == "" {
			return "", fmt.Errorf("couldn't find valid %s", envKey)
		}
	}
	return value, nil
}
