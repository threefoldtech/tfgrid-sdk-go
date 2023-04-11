// Package internal contains all logic for monitoring service
package internal

import (
	"fmt"
	"os"
	"testing"

	client "github.com/threefoldtech/substrate-client"
)

func TestMonitor(t *testing.T) {
	//json
	jsonFile, err := os.CreateTemp("", "*.json")

	if err != nil {
		t.Errorf("failed with error, %v", err)
	}

	defer jsonFile.Close()
	defer os.Remove(jsonFile.Name())

	data := []byte(`{ 
		"mainnet": [ { "name": "name", "address": "address", "threshold": 1} ],
		"testnet": [ { "name": "name-test", "address": "address", "threshold": 1} ] 
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

	data = []byte(`TESTNET_MNEMONIC=mnemonic
	MAINNET_MNEMONIC=mnemonic
	DEVNET_MNEMONIC=mnemonic
	QANET_MNEMONIC=mnemonic
	BOT_TOKEN=token
	CHAT_ID=id
	MINS=10`)
	if _, err := envFile.Write(data); err != nil {
		t.Error(err)
	}

	//managers
	substrate := map[network]client.Manager{}

	substrate[mainNetwork] = client.NewManager(SubstrateURLs[mainNetwork]...)
	substrate[testNetwork] = client.NewManager(SubstrateURLs[testNetwork]...)

	t.Run("test_invalid_monitor_env", func(t *testing.T) {
		_, err := NewMonitor("env", jsonFile.Name())

		if err == nil {
			t.Errorf("monitor should fail, wrong env")
		}
	})

	t.Run("test_invalid_monitor_json", func(t *testing.T) {

		_, err := NewMonitor(envFile.Name(), "wallets")

		if err == nil {
			t.Errorf("monitor should fail, wrong json")
		}
	})

	t.Run("test_valid_monitor", func(t *testing.T) {

		_, err := NewMonitor(envFile.Name(), jsonFile.Name())

		if err != nil {
			t.Errorf("monitor should be successful")
		}
	})

	t.Run("test_invalid_monitor_token", func(t *testing.T) {

		monitor, err := NewMonitor(envFile.Name(), jsonFile.Name())
		if err != nil {
			t.Errorf("monitor should be successful")
		}

		wallet := wallet{"", 1, ""}

		monitor.env.botToken = ""
		err = monitor.sendMessage(substrate[testNetwork], wallet)
		if err == nil {
			t.Errorf("sending a message should fail")
		}
	})

	t.Run("test_send_message_low_threshold", func(t *testing.T) {

		monitor, err := NewMonitor(envFile.Name(), jsonFile.Name())

		if err != nil {
			t.Errorf("monitor should be successful")
		}

		wallet := wallet{"", 1, ""}

		err = monitor.sendMessage(substrate[testNetwork], wallet)
		if err == nil {
			t.Errorf("no message should be sent")
		}
	})

	t.Run("test_telegram_url", func(t *testing.T) {

		monitor, err := NewMonitor(envFile.Name(), jsonFile.Name())
		monitor.env.botToken = "token"
		want := "https://api.telegram.org/bottoken"

		if err != nil {
			t.Errorf("monitor should be successful")
		}

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

	t.Run("test_invalid_monitor_wrong_env", func(t *testing.T) {
		//env
		envFile, err := os.CreateTemp("", "*.env")
		if err != nil {
			t.Errorf("failed with error, %v", err)
		}

		defer envFile.Close()
		defer os.Remove(envFile.Name())

		data = []byte(`TESTNET_MNEMONIC=mnemonic`)
		if _, err := envFile.Write(data); err != nil {
			t.Error(err)
		}

		_, err = NewMonitor(envFile.Name(), jsonFileOK.Name())

		if err == nil {
			t.Errorf("monitor should fail, wrong env")
		}
	})

	t.Run("test_invalid_monitor_wrong_json", func(t *testing.T) {
		//json
		jsonFile, err := os.CreateTemp("", "*.json")

		if err != nil {
			t.Errorf("failed with error, %v", err)
		}

		defer jsonFile.Close()
		defer os.Remove(jsonFile.Name())

		data := []byte(`[]`)
		if _, err := jsonFile.Write(data); err != nil {
			t.Error(err)
		}

		_, err = NewMonitor(envFileOk.Name(), jsonFile.Name())

		if err == nil {
			t.Errorf("monitor should fail, wrong wallets")
		}
	})
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
		"mainnet": [ { "name": "name", "address": "address", "threshold": 1} ],
		"testnet": [ { "name": "name-test", "address": "address", "threshold": 1} ] 
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

	data = []byte(`TESTNET_MNEMONIC=mnemonic
	MAINNET_MNEMONIC=mnemonic
	DEVNET_MNEMONIC=mnemonic
	QANET_MNEMONIC=mnemonic
	BOT_TOKEN=token
	CHAT_ID=id
	MINS=10`)
	if _, err := envFile.Write(data); err != nil {
		t.Error(err)
	}

	t.Run("test_failed_system_versions", func(t *testing.T) {
		mon, err := NewMonitor(envFile.Name(), jsonFile.Name())

		if err != nil {
			fmt.Printf("ver: %v\n", err)
			t.Errorf("monitor should be successful")
		}

		_, err = mon.systemVersion()

		if err != nil {
			t.Errorf("getting system versions failed")
		}
	})
}
