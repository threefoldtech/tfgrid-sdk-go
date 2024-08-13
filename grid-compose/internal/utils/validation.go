package utils

import "github.com/cosmos/go-bip39"

func ValidateCredentials(mnemonics, network string) bool {
	return validateMnemonics(mnemonics) && validateNetwork(network)
}

func validateMnemonics(mnemonics string) bool {
	return bip39.IsMnemonicValid(mnemonics)
}

func validateNetwork(network string) bool {
	switch network {
	case "test", "dev", "main", "qa":
		return true
	default:
		return false
	}
}
