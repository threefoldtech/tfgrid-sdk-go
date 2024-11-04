package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/tfrobot/internal/parser"
	tfrobot "github.com/threefoldtech/tfgrid-sdk-go/tfrobot/pkg/deployer"
	"golang.org/x/sys/unix"
)

const jsonExt = ".json"

var allowedExt = []string{".yml", ".yaml", jsonExt}

func setup(conf tfrobot.Config, debug bool) (deployer.TFPluginClient, error) {
	network := conf.Network
	log.Debug().Str("network", network).Send()

	mnemonic := conf.Mnemonic
	log.Debug().Str("mnemonic", mnemonic).Send()

	opts := []deployer.PluginOpt{
		deployer.WithTwinCache(),
		deployer.WithRMBTimeout(30),
		deployer.WithNetwork(network),
	}
	if debug {
		opts = append(opts, deployer.WithLogs())
	}

	return deployer.NewTFPluginClient(mnemonic, opts...)
}

func readConfig(configPath string) (tfrobot.Config, error) {
	if configPath == "" {
		return tfrobot.Config{}, errors.New("required configuration file path is empty")
	}

	configFile, err := os.Open(configPath)
	if err != nil {
		return tfrobot.Config{}, errors.Wrapf(err, "failed to open configuration file '%s'", configPath)
	}
	defer configFile.Close()

	if !slices.Contains(allowedExt, filepath.Ext(configPath)) {
		return tfrobot.Config{}, fmt.Errorf("unsupported configuration file format '%s', should be [yaml, yml, json]", configPath)
	}

	cfg, err := parser.ParseConfig(configFile, filepath.Ext(configPath) == jsonExt)
	if err != nil {
		return tfrobot.Config{}, errors.Wrapf(err, "failed to parse configuration file '%s' with error", configPath)
	}
	return cfg, nil
}

func checkOutputFile(outputPath string) error {
	if !slices.Contains(allowedExt, filepath.Ext(outputPath)) {
		return fmt.Errorf("unsupported output file format '%s', should be [yaml, yml, json]", outputPath)
	}

	_, err := os.Stat(outputPath)
	// check if output file is writable
	if !errors.Is(err, os.ErrNotExist) && unix.Access(outputPath, unix.W_OK) != nil {
		return fmt.Errorf("output path '%s' is not writable", outputPath)
	}
	return nil
}
