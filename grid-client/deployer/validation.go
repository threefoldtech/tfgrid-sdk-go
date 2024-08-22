package deployer

import (
	"math"
	"math/big"
	"regexp"
	"strings"

	"github.com/cosmos/go-bip39"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/subi"
	proxy "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/client"
)

// validateAccount checks the mnemonics is associated with an account with key type ed25519
func validateAccount(sub subi.SubstrateExt, identity substrate.Identity, mnemonics string) error {
	_, err := sub.GetAccount(identity)
	if err != nil && !errors.Is(err, substrate.ErrAccountNotFound) {
		return errors.Wrap(err, "failed to get account with the given mnemonics")
	}

	if err != nil { // Account not found
		funcs := map[string]func(string) (substrate.Identity, error){"ed25519": substrate.NewIdentityFromEd25519Phrase, "sr25519": substrate.NewIdentityFromSr25519Phrase}
		for keyType, f := range funcs {
			ident, err2 := f(mnemonics)
			if err2 != nil { // shouldn't happen, return original error
				log.Error().Err(err2).Msgf("could not convert the mnemonics to %s key", keyType)
				return err
			}
			_, err2 = sub.GetAccount(ident)
			if err2 == nil { // found an identity with key type other than the provided
				return errors.Errorf("found an account with %s key type and the same mnemonics, make sure you provided the correct key type", keyType)
			}
		}
		// didn't find an account with any key type
		return err
	}
	return nil
}

func validateRMBProxyServer(gridProxyClient proxy.Client) error {
	return gridProxyClient.Ping()
}

func validateMnemonics(mnemonics string) bool {
	return bip39.IsMnemonicValid(mnemonics)
}

func validateWssURL(url string) error {
	if len(strings.TrimSpace(url)) == 0 {
		return errors.New("url is required")
	}

	alphaOnly := regexp.MustCompile(`^wss:\/\/[a-z0-9]+\.[a-z0-9]\/?([^\s<>\#%"\,\{\}\\|\\\^\[\]]+)?$`)
	if !alphaOnly.MatchString(url) {
		return errors.Errorf("wss url '%s' is invalid", url)
	}

	return nil
}

func validateProxyURL(url string) error {
	if len(strings.TrimSpace(url)) == 0 {
		return errors.New("proxy url is required")
	}

	alphaOnly := regexp.MustCompile(`^https:\/\/[a-z0-9]+\.[a-z0-9]\/?([^\s<>\#%"\,\{\}\\|\\\^\[\]]+)?$`)
	if !alphaOnly.MatchString(url) {
		return errors.New("proxy url is invalid")
	}

	return nil
}

func validateGraphQlURL(url string) error {
	if len(strings.TrimSpace(url)) == 0 {
		return errors.New("graphql url is required")
	}

	alphaOnly := regexp.MustCompile(`^https:\/\/[a-z0-9]+\.[a-z0-9]\/?([^\s<>\#%"\,\{\}\\|\\\^\[\]]+)?$`)
	if !alphaOnly.MatchString(url) {
		return errors.New("graphql url is invalid")
	}

	return nil
}

func validateAccountBalanceForExtrinsics(sub subi.SubstrateExt, identity substrate.Identity) error {
	balance, err := sub.GetBalance(identity)
	if err != nil {
		return errors.Wrap(err, "failed to get account with the given mnemonics")
	}

	if balance.Free.Cmp(big.NewInt(20000000)) == -1 {
		return errors.Errorf("account contains %f tft, min fee is 2 tft", float64(balance.Free.Int64())/math.Pow(10, 7))
	}

	return nil
}
