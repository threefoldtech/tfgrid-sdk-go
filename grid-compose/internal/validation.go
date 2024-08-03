package internal

import "github.com/cosmos/go-bip39"

func ValidateMnemonics(mnemonics string) bool {
	return bip39.IsMnemonicValid(mnemonics)
}
