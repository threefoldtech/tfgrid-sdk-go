package monitor

import (
	"os"
	"strings"

	env "github.com/hashicorp/go-envparse"
)

func parseConfig(envPath string) (map[string]string, error) {
	envMap := make(map[string]string)
	envContent, err := os.ReadFile(envPath)
	if err != nil {
		return envMap, err
	}

	envMap, err = env.Parse(strings.NewReader(string(envContent)))
	if err != nil {
		return envMap, err
	}
	return envMap, nil
}
