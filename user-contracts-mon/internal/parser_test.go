package monitor

import (
	"testing"
)

func TestParsers(t *testing.T) {
	t.Run("test_no_file", func(t *testing.T) {
		_, err := parseFile(".env")

		if err == nil {
			t.Errorf("expected error reading .env")
		}
	})

	t.Run("test_valid_file", func(t *testing.T) {
		_, err := parseFile("parser.go")
		if err != nil {
			t.Errorf("expected no error, %v", err)
		}
	})

	t.Run("test_wrong_env_missing_fields", func(t *testing.T) {
		envMap := ` 
			NETWORK=  "dev",
			INTERVAL= "2",
            `
		_, err := parseMonitor(envMap)

		if err == nil {
			t.Errorf("expected error, missing fields")
		}
	})

	t.Run("test_wrong_env_invalid_interval", func(t *testing.T) {
		envContent := `
			BOT_TOKEN=
            NETWORK= "test"
		`

		_, err := parseMonitor(envContent)

		if err == nil {
			t.Errorf("expected error, invalid interval")
		}
	})

	t.Run("test_valid_env", func(t *testing.T) {
		envContent := `
			BOT_TOKEN= "token"
            NETWORK=   "network"
            MNEMONIC=  "mnemonic"
            INTERVAL=  "3"
		`
		_, err := parseMonitor(envContent)
		if err != nil {
			t.Errorf("parsing should be successful")
		}
	})
}
