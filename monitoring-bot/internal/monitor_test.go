// Package internal contains all logic for monitoring service
package internal

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/cosmos/go-bip39"
	"github.com/stretchr/testify/assert"
	client "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
)

var entropy, _ = bip39.NewEntropy(256)
var mnemonic, _ = bip39.NewMnemonic(entropy)

func TestMonitor(t *testing.T) {
	//json
	jsonFile, err := os.CreateTemp("", "*.json")

	if err != nil {
		t.Errorf("failed with error, %v", err)
	}

	defer jsonFile.Close()
	defer os.Remove(jsonFile.Name())

	data := []byte(`{ 
		"mainnet": [ { "name": "name", "address": "5ECu6QxQ8eQmAjDtKQE6UVgzRFrdmYH1VvUiiK4UyxhkJ469", "threshold": 1} ],
		"testnet": [ { "name": "name-test", "address": "5GLQdUZ3tyeashZteV2nYYiJ6TdXKxEPhiBtoyWcb8jFuwVq", "threshold": 1} ] 
	}`)
	if _, err := jsonFile.Write(data); err != nil {
		t.Error(err)
	}

	//env
	envFile, err := os.CreateTemp("", "*.env")
	if err != nil {
		t.Errorf("failed with error, %v", err)
	}

	defer envFile.Close()
	defer os.Remove(envFile.Name())

	data = []byte(fmt.Sprintf(`TESTNET_MNEMONIC=%s
	MAINNET_MNEMONIC=%s
	DEVNET_MNEMONIC=%s
	QANET_MNEMONIC=%s
	BOT_TOKEN=token
	CHAT_ID=id
	MINS=10`, mnemonic, mnemonic, mnemonic, mnemonic))
	if _, err := envFile.Write(data); err != nil {
		t.Error(err)
	}

	//managers
	substrate := map[network]client.Manager{}

	substrate[mainNetwork] = client.NewManager(SubstrateURLs[mainNetwork]...)
	substrate[testNetwork] = client.NewManager(SubstrateURLs[testNetwork]...)

	envContent, err := ReadFile(envFile.Name())
	assert.NoError(t, err)

	env, err := ParseEnv(string(envContent))
	assert.NoError(t, err)

	walletsContent, err := ReadFile(jsonFile.Name())
	assert.NoError(t, err)

	wallets, err := ParseJSONIntoWallets(walletsContent)
	assert.NoError(t, err)

	monitor, err := NewMonitor(env, wallets)
	assert.NoError(t, err)

	t.Run("test_invalid_monitor_token", func(t *testing.T) {
		wallet := wallet{"", 1, ""}

		monitor.env.botToken = ""
		err = monitor.monitorBalance(substrate[testNetwork], wallet)
		if err == nil {
			t.Errorf("sending a message should fail")
		}
	})

	t.Run("test_send_message_low_threshold", func(t *testing.T) {
		wallet := wallet{"", 1, ""}

		err = monitor.monitorBalance(substrate[testNetwork], wallet)
		if err == nil {
			t.Errorf("no message should be sent")
		}
	})

	t.Run("test_telegram_url", func(t *testing.T) {
		monitor.env.botToken = "token"
		want := "https://api.telegram.org/bottoken"

		telegramURL := monitor.getTelegramURL()
		if telegramURL != want {
			t.Errorf("telegram wrong url")
		}
	})
}

func TestWrongFilesContent(t *testing.T) {
	//json
	jsonFileOK, err := os.CreateTemp("", "*.json")

	if err != nil {
		t.Errorf("failed with error, %v", err)
	}

	defer jsonFileOK.Close()
	defer os.Remove(jsonFileOK.Name())

	data := []byte(`{ 
		"mainnet": []  
	}`)
	if _, err := jsonFileOK.Write(data); err != nil {
		t.Error(err)
	}

	//env
	envFileOk, err := os.CreateTemp("", "*.env")
	if err != nil {
		t.Errorf("failed with error, %v", err)
	}

	defer envFileOk.Close()
	defer os.Remove(envFileOk.Name())

	data = []byte(`TESTNET_MNEMONIC=mnemonic
	MAINNET_MNEMONIC=mnemonic
	BOT_TOKEN=token
	CHAT_ID=id
	MINS=10`)
	if _, err := envFileOk.Write(data); err != nil {
		t.Error(err)
	}
}

func TestZosVersion(t *testing.T) {
	//json
	jsonFile, err := os.CreateTemp("", "*.json")

	if err != nil {
		t.Errorf("failed with error, %v", err)
	}

	defer jsonFile.Close()
	defer os.Remove(jsonFile.Name())

	data := []byte(`{ 
		"mainnet": [ { "name": "name", "address": "5ECu6QxQ8eQmAjDtKQE6UVgzRFrdmYH1VvUiiK4UyxhkJ469", "threshold": 1} ],
		"testnet": [ { "name": "name-test", "address": "5GLQdUZ3tyeashZteV2nYYiJ6TdXKxEPhiBtoyWcb8jFuwVq", "threshold": 1} ] 
	}`)
	if _, err := jsonFile.Write(data); err != nil {
		t.Error(err)
	}

	//env
	envFile, err := os.CreateTemp("", "*.env")
	if err != nil {
		t.Errorf("failed with error, %v", err)
	}

	defer envFile.Close()
	defer os.Remove(envFile.Name())

	data = []byte(fmt.Sprintf(`TESTNET_MNEMONIC=%s
	MAINNET_MNEMONIC=%s
	DEVNET_MNEMONIC=%s
	QANET_MNEMONIC=%s
	BOT_TOKEN=token
	CHAT_ID=id
	MINS=10`, mnemonic, mnemonic, mnemonic, mnemonic))
	if _, err := envFile.Write(data); err != nil {
		t.Error(err)
	}

	t.Run("test_failed_system_versions", func(t *testing.T) {
		envContent, err := ReadFile(envFile.Name())
		assert.NoError(t, err)

		env, err := ParseEnv(string(envContent))
		assert.NoError(t, err)

		walletsContent, err := ReadFile(jsonFile.Name())
		assert.NoError(t, err)

		wallets, err := ParseJSONIntoWallets(walletsContent)
		assert.NoError(t, err)

		monitor, err := NewMonitor(env, wallets)
		assert.NoError(t, err)

		versions, working, failed := monitor.systemVersion(context.Background())
		assert.Empty(t, versions)
		assert.Empty(t, working)
		assert.Empty(t, failed)
	})
}
