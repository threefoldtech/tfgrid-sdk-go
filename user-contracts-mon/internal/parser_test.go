package monitor

import (
	"os"
	"path"
	"testing"
)

func TestParsers(t *testing.T) {
	tempDir := t.TempDir()
	testEnv := path.Join(tempDir, "test.env")

	_, err := os.Create(testEnv)
	if err != nil {
		t.Errorf("failed to create test file")
	}

	t.Run("test_no_file", func(t *testing.T) {
		_, err := ParseConfig("env")

		if err == nil {
			t.Errorf("expected error reading .env")
		}
	})

	t.Run("test_wrong_env_missing_bot_token", func(t *testing.T) {
		env := ` 
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
            INTERVAL=  "3"
		`
		want := Config{
			botToken: "token",
			interval: 3,
		}

		err = os.WriteFile(testEnv, []byte(env), 0667)
		if err != nil {
			t.Errorf("failed to write to test file")
		}

		got, err := ParseConfig(testEnv)
		if err != nil {
			t.Errorf("parsing should be successful")
		}

		if want != got {
			t.Errorf("Expected: %v\nbut got: %v", want, got)
		}
	})
}
