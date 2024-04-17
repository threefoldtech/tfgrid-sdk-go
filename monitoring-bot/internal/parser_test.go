// Package internal contains all logic for monitoring service
package internal

import (
	"testing"
)

func TestParsers(t *testing.T) {
	t.Run("test_no_file", func(t *testing.T) {
		_, err := ReadFile("env.env")

		if err == nil {
			t.Errorf("expected error reading env.env")
		}
	})

	t.Run("test_valid_file", func(t *testing.T) {
		_, err := ReadFile("monitor.go")

		if err != nil {
			t.Errorf("expected no error, %v", err)
		}
	})

	t.Run("test_wrong_env_no_test_mnemonic", func(t *testing.T) {
		envContent := `
			TESTNET_MNEMONIC=
			MAINNET_MNEMONIC=mnemonic
			DEVNET_MNEMONIC=mnemonic
			QANET_MNEMONIC=mnemonic
			BOT_TOKEN=TOKEN
			CHAT_ID=ID
			MINS=1
		`

		_, err := ParseEnv(envContent)

		if err == nil {
			t.Errorf("expected error, no TESTNET_MNEMONIC")
		}
	})

	t.Run("test_wrong_env_no_main_mnemonic", func(t *testing.T) {
		envContent := `
			TESTNET_MNEMONIC=mnemonic
			MAINNET_MNEMONIC=
			DEVNET_MNEMONIC=mnemonic
			QANET_MNEMONIC=mnemonic
			BOT_TOKEN=TOKEN
			CHAT_ID=ID
			MINS=1
		`

		_, err := ParseEnv(envContent)

		if err == nil {
			t.Errorf("expected error, no MAINNET_MNEMONIC")
		}
	})

	t.Run("test_wrong_env_no_qa_mnemonic", func(t *testing.T) {
		envContent := `
			TESTNET_MNEMONIC=mnemonic
			MAINNET_MNEMONIC=mnemonic
			DEVNET_MNEMONIC=mnemonic
			QANET_MNEMONIC=
			BOT_TOKEN=TOKEN
			CHAT_ID=ID
			MINS=1
		`

		_, err := ParseEnv(envContent)

		if err == nil {
			t.Errorf("expected error, no QANET_MNEMONIC")
		}
	})

	t.Run("test_wrong_env_no_token", func(t *testing.T) {
		envContent := `
			TESTNET_MNEMONIC=mnemonic
			MAINNET_MNEMONIC=mnemonic
			DEVNET_MNEMONIC=mnemonic
			QANET_MNEMONIC=mnemonic
			BOT_TOKEN=
			CHAT_ID=ID
			MINS=1
		`

		_, err := ParseEnv(envContent)

		if err == nil {
			t.Errorf("expected error, BOT_TOKEN is missing")
		}
	})

	t.Run("test_wrong_env_no_id", func(t *testing.T) {
		envContent := `
			TESTNET_MNEMONIC=mnemonic
			MAINNET_MNEMONIC=mnemonic
			DEVNET_MNEMONIC=mnemonic
			QANET_MNEMONIC=mnemonic
			BOT_TOKEN=token
			CHAT_ID=
			MINS=1
		`

		_, err := ParseEnv(envContent)

		if err == nil {
			t.Errorf("expected error, CHAT_ID is missing")
		}
	})

	t.Run("test_wrong_env_0_mins", func(t *testing.T) {
		envContent := `
			TESTNET_MNEMONIC=mnemonic
			MAINNET_MNEMONIC=mnemonic
			DEVNET_MNEMONIC=mnemonic
			QANET_MNEMONIC=mnemonic
			BOT_TOKEN=token
			CHAT_ID=id
			MINS=0
		`

		_, err := ParseEnv(envContent)

		if err == nil {
			t.Errorf("expected error, MINS is 0")
		}
	})

	t.Run("test_wrong_env_string_mins", func(t *testing.T) {
		envContent := `
			TESTNET_MNEMONIC=mnemonic
			MAINNET_MNEMONIC=mnemonic
			BOT_TOKEN=token
			CHAT_ID=id
			MINS=min
		`

		_, err := ParseEnv(envContent)

		if err == nil {
			t.Errorf("expected error, MINS is string")
		}
	})

	t.Run("test_wrong_env_key", func(t *testing.T) {
		envContent := `
			key=key
			TESTNET_MNEMONIC=mnemonic
			MAINNET_MNEMONIC=mnemonic
			BOT_TOKEN=token
			CHAT_ID=id
			MINS=10
		`
		_, err := ParseEnv(envContent)

		if err == nil {
			t.Errorf("expected error, key is invalid")
		}
	})

	t.Run("test_valid_env", func(t *testing.T) {
		envContent := `
			TESTNET_MNEMONIC=mnemonic
			MAINNET_MNEMONIC=mnemonic
			DEVNET_MNEMONIC=mnemonic
			QANET_MNEMONIC=mnemonic
			DEV_FARM_NAME=Freefarm
			QA_FARM_NAME=Freefarm
			MAIN_FARM_NAME=Freefarm
			TEST_FARM_NAME=FreeFarm
			BOT_TOKEN=token
			CHAT_ID=id
			MINS=10
		`
		_, err := ParseEnv(envContent)

		if err != nil {
			t.Errorf("parsing should be successful")
		}
	})

	t.Run("test_invalid_env", func(t *testing.T) {
		envContent := `mnemonic`

		_, err := ParseEnv(envContent)

		if err == nil {
			t.Errorf("parsing should fail")
		}
	})

	t.Run("test_valid_json", func(t *testing.T) {
		content := `
		{ 
			"mainnet": [ { "name": "name", "address": "address", "threshold": 1} ],
			"testnet": [ { "name": "name-test", "address": "address", "threshold": 1} ], 
			"devnet": [ { "name": "name-dev", "address": "address", "threshold": 1} ],
			"qanet": [ { "name": "name-qa", "address": "address", "threshold": 1} ]   
		}
		`
		_, err := ParseJSONIntoWallets([]byte(content))

		if err != nil {
			t.Errorf("parsing should be successful")
		}
	})

	t.Run("test_invalid_json", func(t *testing.T) {
		content := `[]`
		_, err := ParseJSONIntoWallets([]byte(content))

		if err == nil {
			t.Errorf("parsing should fail")
		}
	})
}
