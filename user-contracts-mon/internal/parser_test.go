package monitor

import (
	"os"
	"testing"
)

func TestParsers(t *testing.T) {
	testEnv := "test.env"
	t.Run("test_no_file", func(t *testing.T) {
		_, err := ParseConfig("env")

		if err == nil {
			t.Errorf("expected error reading .env")
		}
	})

	os.Create(testEnv)

	t.Run("test_wrong_env_missing_mnemonics", func(t *testing.T) {
		env := ` 
			NETWORK=  "dev",
			INTERVAL= "2",
            BOT_TOKEN="token",
            `
		os.WriteFile(testEnv, []byte(env), 0667)
		_, err := ParseConfig(testEnv)

		if err == nil {
			t.Errorf("expected error, missing fields")
		}
	})

	t.Run("test_wrong_env_missing_network", func(t *testing.T) {
		env := ` 
			MNEMONIC=  "mnemonic",
			INTERVAL= "2",
            BOT_TOKEN="token",
            `
		os.WriteFile(testEnv, []byte(env), 0667)
		_, err := ParseConfig(testEnv)

		if err == nil {
			t.Errorf("expected error, missing fields")
		}
	})

	t.Run("test_wrong_env_missing_bot_token", func(t *testing.T) {
		env := ` 
			MNEMONIC=  "mnemonic",
            NETWORK="dev",
			INTERVAL= "2",
            `
		os.WriteFile(testEnv, []byte(env), 0667)
		_, err := ParseConfig(testEnv)

		if err == nil {
			t.Errorf("expected error, missing fields")
		}
	})
	t.Run("test_wrong_env_invalid_interval", func(t *testing.T) {
		env := `
			BOT_TOKEN=
            NETWORK= "test"
		`

		os.WriteFile(testEnv, []byte(env), 0667)
		_, err := ParseConfig(testEnv)

		if err == nil {
			t.Errorf("expected error, invalid interval")
		}
	})

	t.Run("test_valid_env", func(t *testing.T) {
		env := `
			BOT_TOKEN= "token"
            NETWORK=   "network"
            MNEMONIC=  "mnemonic"
            INTERVAL=  "3"
		`
		os.WriteFile(testEnv, []byte(env), 0667)
		_, err := ParseConfig(testEnv)
		if err != nil {
			t.Errorf("parsing should be successful")
		}
	})
	defer os.Remove(testEnv)
}
