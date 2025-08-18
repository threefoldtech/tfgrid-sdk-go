package deployer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type externalConfig struct {
	RelaysURLs    []string `json:"relays_urls"`
	SubstrateURLs []string `json:"substrate_urls"`
	GraphqlURLs   []string `json:"graphql_urls"`
	KYCURL        string   `json:"kyc_url"`
}

var (
	DefaultConfigBaseURL = "https://raw.githubusercontent.com/threefoldtech/zosbase/main/config"
)

func getNetworkFilename(network string) string {
	networkFileMap := map[string]string{
		"dev":  "development",
		"qa":   "qa",
		"test": "testing",
		"main": "production",
	}

	if filename, exists := networkFileMap[network]; exists {
		return filename
	}

	return network
}

func getConfigURL(network string) string {
	baseURL := strings.TrimSpace(os.Getenv("TFGRID_CONFIG_BASE_URL"))
	if baseURL == "" {
		baseURL = DefaultConfigBaseURL
	}

	baseURL = strings.TrimSuffix(baseURL, "/")

	filename := getNetworkFilename(network)

	return fmt.Sprintf("%s/%s.json", baseURL, filename)
}

func FetchRemoteConfig(network string) (externalConfig, error) {

	cfgURL := getConfigURL(network)

	httpClient := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, cfgURL, nil)
	if err != nil {
		return externalConfig{}, fmt.Errorf("create request: %w", err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return externalConfig{}, fmt.Errorf("fetch config: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return externalConfig{}, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, cfgURL)
	}
	var cfg externalConfig
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return externalConfig{}, fmt.Errorf("decode json: %w", err)
	}
	return cfg, nil
}
