package deployer

import (
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

func setup(conf Config, debug bool) (deployer.TFPluginClient, error) {
	network := conf.Network
	log.Debug().Str("network", network).Send()

	mnemonic := conf.Mnemonic
	log.Debug().Str("mnemonic", mnemonic).Send()

	var proxyURL string
	noNinjaProxyURL := strings.TrimSpace(strings.ToLower(os.Getenv("NO_NINJA_PROXY_URL")))

	if network == "main" && noNinjaProxyURL == "" {
		proxyURL = "https://gridproxy.bknd1.ninja.tf"
	}
	return deployer.NewTFPluginClient(mnemonic, peer.KeyTypeSr25519, network, "", "", proxyURL, 30, debug, true)
}

func convertGBToBytes(gb uint64) uint64 {
	bytes := gb * 1024 * 1024 * 1024
	return bytes
}

func convertMBToBytes(mb uint64) uint64 {
	bytes := mb * 1024 * 1024
	return bytes
}
