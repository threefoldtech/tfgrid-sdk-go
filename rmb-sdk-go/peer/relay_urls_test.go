package peer

import (
	"reflect"
	"testing"
)

func TestValidateRelayURLs_DedupSort(t *testing.T) {
	inputs := []string{
		"ws://Relay.Grid.tf",           // same host different case, ws
		"wss://relay.grid.tf",          // same host wss
		"wss://relay.grid.tf:443",      // same host with default port -> ignored for dedup
		"ws://relay.grid.tf/some/path", // same host path -> ignored for dedup
		"wss://beta.grid.tf",           // distinct host
		"http://not-allowed.example",   // invalid scheme
		"://bad://url",                 // unparsable
		"wss://relay.dev.grid.tf:443",  // another host
		"ws://relay.dev.grid.tf",       // duplicate host, ws should be ignored in favor of wss entry
	}

	urls, err := validateRelayURLs(inputs)
	if err != nil {
		// should not error; we have valid entries
		t.Fatalf("unexpected error: %v", err)
	}

	// Expect unique hostnames (lowercased) sorted: [beta.grid.tf relay.dev.grid.tf relay.grid.tf]
	if len(urls) != 3 {
		t.Fatalf("expected 3 unique hosts, got %d", len(urls))
	}

	expectedHosts := []string{"beta.grid.tf", "relay.dev.grid.tf", "relay.grid.tf"}
	gotHosts := []string{urls[0].Hostname(), urls[1].Hostname(), urls[2].Hostname()}
	if !reflect.DeepEqual(expectedHosts, gotHosts) {
		t.Fatalf("hosts mismatch\nexpected: %v\n     got: %v", expectedHosts, gotHosts)
	}
}

func TestValidateRelayURLs_AllInvalid(t *testing.T) {
	inputs := []string{"http://a", "ftp://b", "::::"}
	_, err := validateRelayURLs(inputs)
	if err == nil {
		t.Fatalf("expected error for no valid relay URLs")
	}
	if err != ErrNoValidRelayURLs {
		t.Fatalf("expected ErrNoValidRelayURLs, got %v", err)
	}
}

func TestGetRelayConnections_HostsNormalizedAndAligned(t *testing.T) {
	inputs := []string{
		"ws://relay.grid.tf:443/path", // same host as below, different scheme/port/path
		"wss://relay.grid.tf",
		"wss://Relay.Dev.Grid.TF", // case-insensitive host
	}

	hosts, conns, err := getRelayConnections(inputs, nil, "sess", 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hosts) != len(conns) {
		t.Fatalf("hosts and connections length mismatch: %d vs %d", len(hosts), len(conns))
	}

	expectedHosts := []string{"relay.dev.grid.tf", "relay.grid.tf"}
	if !reflect.DeepEqual(expectedHosts, hosts) {
		t.Fatalf("normalized hosts mismatch\nexpected: %v\n     got: %v", expectedHosts, hosts)
	}
}

func TestValidateRelayURLs_DeterministicSort(t *testing.T) {
	inputs := []string{
		"wss://relay.grid.tf",
		"wss://relay.dev.grid.tf",
		"wss://beta.grid.tf",
	}

	urls, err := validateRelayURLs(inputs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedHosts := []string{"beta.grid.tf", "relay.dev.grid.tf", "relay.grid.tf"}
	gotHosts := []string{urls[0].Hostname(), urls[1].Hostname(), urls[2].Hostname()}
	if !reflect.DeepEqual(expectedHosts, gotHosts) {
		t.Fatalf("hosts mismatch\nexpected: %v\n     got: %v", expectedHosts, gotHosts)
	}
}
