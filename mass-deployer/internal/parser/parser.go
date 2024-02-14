package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gliderlabs/ssh"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	massDeployer "github.com/threefoldtech/tfgrid-sdk-go/mass-deployer/pkg/mass-deployer"
	"gopkg.in/yaml.v3"
)

const (
	mnemonicKey = "MNEMONIC"
	networkKey  = "NETWORK"
)

func ParseConfig(file io.Reader, jsonFmt bool) (massDeployer.Config, error) {
	conf := massDeployer.Config{}

	configFile, err := io.ReadAll(file)
	if err != nil {
		return massDeployer.Config{}, fmt.Errorf("failed to read the config file: %+w", err)
	}
	if jsonFmt {
		err = json.Unmarshal(configFile, &conf)
	} else {
		err = yaml.Unmarshal(configFile, &conf)
	}
	if err != nil {
		return massDeployer.Config{}, err
	}

	if conf.Mnemonic, err = getValueOrEnv(conf.Mnemonic, mnemonicKey); err != nil {
		return massDeployer.Config{}, err
	}

	if conf.Network, err = getValueOrEnv(conf.Network, networkKey); err != nil {
		return massDeployer.Config{}, err
	}

	if err := validateNetwork(conf.Network); err != nil {
		return massDeployer.Config{}, err
	}

	if err := validateMnemonic(conf.Mnemonic); err != nil {
		return massDeployer.Config{}, err
	}

	for _, nodeGroup := range conf.NodeGroups {
		nodeGroupName := strings.TrimSpace(nodeGroup.Name)
		if !alphanumeric.MatchString(nodeGroupName) {
			return massDeployer.Config{}, fmt.Errorf("node group name: '%s' is invalid, should be lowercase alphanumeric and underscore only", nodeGroupName)
		}
	}

	return conf, nil
}

func ValidateConfig(conf massDeployer.Config, tfPluginClient deployer.TFPluginClient) error {
	log.Info().Msg("validating configuration file")

	v := validator.New(validator.WithRequiredStructEnabled())
	if err := v.Struct(conf); err != nil {
		err = parseValidationError(err)
		return err
	}

	for name, sshKey := range conf.SSHKeys {
		if _, _, _, _, err := ssh.ParseAuthorizedKey([]byte(strings.TrimSpace(sshKey))); err != nil {
			return fmt.Errorf("ssh key for `%s` is invalid: %+w", name, err)
		}
	}

	if err := validateNodeGroups(conf.NodeGroups, tfPluginClient); err != nil {
		return err
	}

	if err := validateVMs(conf.Vms, conf.NodeGroups, conf.SSHKeys); err != nil {
		return err
	}

	log.Info().Msg("done validating configuration file")
	return nil
}

func getValueOrEnv(value, envKey string) (string, error) {
	envKey = strings.ToUpper(envKey)
	if len(strings.TrimSpace(value)) == 0 {
		value = os.Getenv(envKey)
		if len(strings.TrimSpace(value)) == 0 {
			return "", fmt.Errorf("could not find valid %s", envKey)
		}
	}
	return value, nil
}

func parseValidationError(err error) error {
	if _, ok := err.(*validator.InvalidValidationError); ok {
		return err
	}

	for _, err := range err.(validator.ValidationErrors) {
		tag := err.Tag()
		value := err.Value()
		boundary := err.Param()
		nameSpace := err.Namespace()

		switch tag {
		case "required":
			return fmt.Errorf("field '%s' should not be empty", nameSpace)
		case "max":
			return fmt.Errorf("value of '%s': '%v' is out of range, max value is '%s'", nameSpace, value, boundary)
		case "min":
			return fmt.Errorf("value of '%s': '%v' is out of range, min value is '%s'", nameSpace, value, boundary)
		case "unique":
			return fmt.Errorf("value of '%s': '%v' is invalid, %s should have unique names", nameSpace, value, nameSpace)
		}
	}
	return nil
}
