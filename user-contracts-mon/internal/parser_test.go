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

	_, err := os.Create(testEnv)
	if err != nil {
		t.Errorf("failed to create test file")
	}

	t.Run("test_wrong_env_missing_mnemonics", func(t *testing.T) {
		env := ` 
			NETWORK=  "dev",
			INTERVAL= "2",
            BOT_TOKEN="token",
            `
		err = os.WriteFile(testEnv, []byte(env), 0667)
		if err != nil {
			t.Errorf("failed to write to test file")
		}

		_, err = ParseConfig(testEnv)
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
		err = os.WriteFile(testEnv, []byte(env), 0667)
		if err != nil {
			t.Errorf("failed to write to test file")
		}

		_, err = ParseConfig(testEnv)
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
		err = os.WriteFile(testEnv, []byte(env), 0667)
		if err != nil {
			t.Errorf("failed to write to test file")
		}

		_, err = ParseConfig(testEnv)

		if err == nil {
			t.Errorf("expected error, missing fields")
		}
	})
	t.Run("test_wrong_env_invalid_interval", func(t *testing.T) {
		env := `
			BOT_TOKEN=
            NETWORK= "test"
		`

		err = os.WriteFile(testEnv, []byte(env), 0667)
		if err != nil {
			t.Errorf("failed to write to test file")
		}

		_, err = ParseConfig(testEnv)
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
		err = os.WriteFile(testEnv, []byte(env), 0667)
		if err != nil {
			t.Errorf("failed to write to test file")
		}

		_, err := ParseConfig(testEnv)
		if err != nil {
			t.Errorf("parsing should be successful")
		}
	})
	defer os.Remove(testEnv)
}
