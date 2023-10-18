// Package cmd for handling commands
package cmd

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/threefoldtech/tfgrid-sdk-go/gridify/internal/config"
	"github.com/threefoldtech/tfgrid-sdk-go/gridify/internal/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/gridify/internal/repository"
	"github.com/threefoldtech/tfgrid-sdk-go/gridify/internal/tfplugin"
)

// Deploy handles deploy command logic
func Deploy(ctx context.Context, vmSpec deployer.VMSpec, ports []uint, debug bool) error {

	path, err := config.GetConfigPath()
	if err != nil {
		return errors.Wrap(err, "failed to get configuration file")
	}
	var cfg config.Config
	err = cfg.Load(path)
	if err != nil {
		return errors.Wrap(err, "failed to load configuration try to login again using gridify login")
	}

	repoURL, err := repository.GetRepositoryURL(".")
	if err != nil {
		return errors.Wrap(err, "failed to get remote repository url")
	}

	logLevel := zerolog.InfoLevel
	if debug {
		logLevel = zerolog.DebugLevel
	}

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).Level(logLevel).
		With().
		Timestamp().
		Logger()

	tfPluginClient, err := tfplugin.NewTFPluginClient(cfg.Mnemonics, cfg.Network)
	if err != nil {
		return errors.Wrapf(err,
			"failed to get threefold plugin client using mnemonics: '%s' on grid network '%s'",
			cfg.Mnemonics,
			cfg.Network,
		)
	}
	deployer, err := deployer.NewDeployer(&tfPluginClient, string(repoURL), logger)
	if err != nil {
		return err
	}

	deploymentName := randName(10)

	fqdns, err := deployer.Deploy(ctx, vmSpec, ports, deploymentName)
	if err != nil {
		return err
	}
	for port, fqdn := range fqdns {
		logger.Info().Msgf("%d: %s", port, fqdn)
	}
	return nil
}
