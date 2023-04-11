// Package internal contains all logic for monitoring service
package internal

const (
	mainNetwork network = "mainnet"
	testNetwork network = "testnet"
	devNetwork  network = "devnet"
	qaNetwork   network = "qanet"
)

var networks = []network{devNetwork, qaNetwork, testNetwork, mainNetwork}

// SubstrateURLs for substrate urls
var SubstrateURLs = map[network][]string{
	testNetwork: {"wss://tfchain.test.grid.tf/ws", "wss://tfchain.test.grid.tf:443"},
	mainNetwork: {"wss://tfchain.grid.tf/ws", "wss://tfchain.grid.tf:443"},
	devNetwork:  {"wss://tfchain.dev.grid.tf/ws", "wss://tfchain.dev.grid.tf:443"},
	qaNetwork:   {"wss://tfchain.qa.grid.tf/ws", "wss://tfchain.qa.grid.tf:443"},
}

// ProxyUrls for proxy urls
var ProxyUrls = map[network]string{
	testNetwork: "https://gridproxy.test.grid.tf/",
	mainNetwork: "https://gridproxy.grid.tf/",
	devNetwork:  "https://gridproxy.dev.grid.tf/",
	qaNetwork:   "https://gridproxy.qa.grid.tf/",
}

// RelayURLS for relay urls
var RelayURLS = map[network]string{
	devNetwork:  "wss://relay.dev.grid.tf",
	testNetwork: "wss://relay.test.grid.tf",
	qaNetwork:   "wss://relay.qa.grid.tf",
	mainNetwork: "wss://relay.grid.tf",
}
