package config

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	aliceMnemonic = "bottom drive obey lake curtain smoke basket hold race lonely fit walk"
	substrateURL  = "wss://tfchain.dev.grid.tf"
	kycPublicKey  = "kyc service 25119 public key"

	rightConfig = fmt.Sprintf(`
URL=%s
MNEMONIC=%s
KYC_PUBLIC_KEY=%s
ACTIVATION_AMOUNT=0
`, substrateURL, aliceMnemonic, kycPublicKey)
)

func TestConfig(t *testing.T) {
	t.Run("read env file ", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "/.env")

		err := os.WriteFile(configPath, []byte(rightConfig), 0644)
		assert.NoError(t, err)

		data, err := ReadConfFile(configPath)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)
	})

	t.Run("change permissions of env file", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "/.env")

		err := os.WriteFile(configPath, []byte(rightConfig), fs.FileMode(os.O_RDONLY))
		assert.NoError(t, err)

		data, err := ReadConfFile(configPath)
		assert.Error(t, err)
		assert.Empty(t, data)
	})

	t.Run("no file exists", func(t *testing.T) {
		data, err := ReadConfFile("./config.env")
		assert.Error(t, err)
		assert.Empty(t, data)
	})

	t.Run("invalid env", func(t *testing.T) {
		config := `key`

		dir := t.TempDir()
		configPath := filepath.Join(dir, "/.env")

		err := os.WriteFile(configPath, []byte(config), 0644)
		assert.NoError(t, err)

		_, err = ReadConfFile(configPath)
		assert.Error(t, err)
	})

	t.Run("no keys valid", func(t *testing.T) {
		config := `key=value`

		dir := t.TempDir()
		configPath := filepath.Join(dir, "/.env")

		err := os.WriteFile(configPath, []byte(config), 0644)
		assert.NoError(t, err)

		_, err = ReadConfFile(configPath)
		assert.Error(t, err)
	})

	t.Run("parse config file", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "/.env")

		err := os.WriteFile(configPath, []byte(rightConfig), 0644)
		assert.NoError(t, err)

		got, err := ReadConfFile(configPath)
		assert.NoError(t, err)

		expected := Configuration{
			ActivationAmount: 1,
			SubstrateURL:     substrateURL,
			Mnemonic:         aliceMnemonic,
			KycPublicKey:     kycPublicKey,
		}

		assert.NoError(t, err)
		assert.Equal(t, got.ActivationAmount, expected.ActivationAmount)
		assert.Equal(t, got.SubstrateURL, expected.SubstrateURL)
		assert.Equal(t, got.Mnemonic, expected.Mnemonic)
		assert.Equal(t, got.KycPublicKey, expected.KycPublicKey)
	})

	t.Run("test no mnemonic", func(t *testing.T) {
		config := fmt.Sprintf(`
		URL=%s
		KYC_PUBLIC_KEY=%s
		ACTIVATION_AMOUNT=0
		`, substrateURL, kycPublicKey)

		dir := t.TempDir()
		configPath := filepath.Join(dir, "/.env")

		err := os.WriteFile(configPath, []byte(config), 0644)
		assert.NoError(t, err)

		_, err = ReadConfFile(configPath)
		assert.Error(t, err)
	})

	t.Run("test no substrate url", func(t *testing.T) {
		config := fmt.Sprintf(`
		MNEMONIC=%s
		KYC_PUBLIC_KEY=%s
		ACTIVATION_AMOUNT=0
		`, aliceMnemonic, kycPublicKey)

		dir := t.TempDir()
		configPath := filepath.Join(dir, "/.env")

		err := os.WriteFile(configPath, []byte(config), 0644)
		assert.NoError(t, err)

		_, err = ReadConfFile(configPath)
		assert.Error(t, err)
	})

	t.Run("test no kyc public key", func(t *testing.T) {
		config := fmt.Sprintf(`
		URL=%s
		MNEMONIC=%s
		ACTIVATION_AMOUNT=0
		`, substrateURL, aliceMnemonic)

		dir := t.TempDir()
		configPath := filepath.Join(dir, "/.env")

		err := os.WriteFile(configPath, []byte(config), 0644)
		assert.NoError(t, err)

		_, err = ReadConfFile(configPath)
		assert.Error(t, err)
	})

	t.Run("test invalid activation amount", func(t *testing.T) {
		config := fmt.Sprintf(`
		URL=%s
		MNEMONIC=%s
		KYC_PUBLIC_KEY=%s
		ACTIVATION_AMOUNT=str
		`, substrateURL, aliceMnemonic, kycPublicKey)

		dir := t.TempDir()
		configPath := filepath.Join(dir, "/.env")

		err := os.WriteFile(configPath, []byte(config), 0644)
		assert.NoError(t, err)

		_, err = ReadConfFile(configPath)
		assert.Error(t, err)
	})

	t.Run("test invalid mnemonic", func(t *testing.T) {
		config := fmt.Sprintf(`
		URL=%s
		MNEMONIC=%s
		KYC_PUBLIC_KEY=%s
		ACTIVATION_AMOUNT=0
		`, substrateURL, "mnemonic", kycPublicKey)

		dir := t.TempDir()
		configPath := filepath.Join(dir, "/.env")

		err := os.WriteFile(configPath, []byte(config), 0644)
		assert.NoError(t, err)

		_, err = ReadConfFile(configPath)
		assert.Error(t, err)
	})

	t.Run("test invalid substrate url", func(t *testing.T) {
		config := fmt.Sprintf(`
		URL=%s
		MNEMONIC=%s
		KYC_PUBLIC_KEY=%s
		ACTIVATION_AMOUNT=0
		`, "http://localhost", aliceMnemonic, kycPublicKey)

		dir := t.TempDir()
		configPath := filepath.Join(dir, "/.env")

		err := os.WriteFile(configPath, []byte(config), 0644)
		assert.NoError(t, err)

		_, err = ReadConfFile(configPath)
		assert.Error(t, err)
	})

	t.Run("test empty substrate url", func(t *testing.T) {
		config := fmt.Sprintf(`
		URL=
		MNEMONIC=%s
		KYC_PUBLIC_KEY=%s
		ACTIVATION_AMOUNT=0
		`, aliceMnemonic, kycPublicKey)

		dir := t.TempDir()
		configPath := filepath.Join(dir, "/.env")

		err := os.WriteFile(configPath, []byte(config), 0644)
		assert.NoError(t, err)

		_, err = ReadConfFile(configPath)
		assert.Error(t, err)
	})

	t.Run("test empty kyc public key", func(t *testing.T) {
		config := fmt.Sprintf(`
		URL=%s
		MNEMONIC=%s
		KYC_PUBLIC_KEY=
		ACTIVATION_AMOUNT=0
		`, substrateURL, aliceMnemonic)

		dir := t.TempDir()
		configPath := filepath.Join(dir, "/.env")

		err := os.WriteFile(configPath, []byte(config), 0644)
		assert.NoError(t, err)

		_, err = ReadConfFile(configPath)
		assert.Error(t, err)
	})
}
