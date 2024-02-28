package cmd

import (
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
	tfrobot "github.com/threefoldtech/tfgrid-sdk-go/tfrobot/pkg/deployer"
)

func setup(conf tfrobot.Config, debug bool) (deployer.TFPluginClient, error) {
	network := conf.Network
	log.Debug().Str("network", network).Send()

	mnemonic := conf.Mnemonic
	log.Debug().Str("mnemonic", mnemonic).Send()

	var proxyURL string
	noNinjaProxyURL := strings.TrimSpace(strings.ToLower(os.Getenv("NO_NINJA_PROXY_URL")))

	if network == "main" && noNinjaProxyURL == "" {
		proxyURL = "https://gridproxy.bknd1.ninja.tf"
	}

	opts := []deployer.PluginOpt{
		deployer.WithProxyURL(proxyURL),
		deployer.WithRMBTimeout(30),
		deployer.WithNetwork(network),
	}
	if debug {
		opts = append(opts, deployer.WithLogs())
	}

	return deployer.NewTFPluginClient(mnemonic, opts...)
}
