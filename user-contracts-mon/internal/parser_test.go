package monitor

import (
	"strings"
	"testing"
)

func TestParsers(t *testing.T) {
	t.Run("test_no_file", func(t *testing.T) {
		_, err := readFile("env")

		if err == nil {
			t.Errorf("expected error reading .env")
		}
	})

	t.Run("test_valid_file", func(t *testing.T) {
		_, err := readFile("parser.go")
		if err != nil {
			t.Errorf("expected no error, %v", err)
		}
	})

	t.Run("test_wrong_env_missing_fields", func(t *testing.T) {
		env := ` 
			NETWORK=  "dev",
			INTERVAL= "2",
            `
		_, err := ParseConfig(strings.NewReader(env))

		if err == nil {
			t.Errorf("expected error, missing fields")
		}
	})

	t.Run("test_wrong_env_invalid_interval", func(t *testing.T) {
		env := `
			BOT_TOKEN=
            NETWORK= "test"
		`

		_, err := ParseConfig(strings.NewReader(env))

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
		_, err := ParseConfig(strings.NewReader(env))
		if err != nil {
			t.Errorf("parsing should be successful")
		}
	})
}
