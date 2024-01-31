package deployer

import (
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
)

func setup(conf Config) (deployer.TFPluginClient, error) {
	network := conf.Network
	log.Debug().Str("network", network).Send()

	mnemonic := conf.Mnemonic
	log.Debug().Str("mnemonic", mnemonic).Send()

	return deployer.NewTFPluginClient(mnemonic, "sr25519", network, "", "", "", 30, false)
}

func convertGBToBytes(gb uint64) uint64 {
	bytes := gb * 1024 * 1024 * 1024
	return bytes
}

func convertMBToBytes(mb uint64) uint64 {
	bytes := mb * 1024 * 1024
	return bytes
}
