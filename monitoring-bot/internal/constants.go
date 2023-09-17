// Package internal contains all logic for monitoring service
package internal

const (
	mainNetwork network = "mainnet"
	testNetwork network = "testnet"
	devNetwork  network = "devnet"
	qaNetwork   network = "qanet"

	tftIssuerAddress    = "GA47YZA3PKFUZMPLQ3B5F2E3CJIB57TGGU7SPCQT2WAEYKN766PWIMB3"
	bridgeTestTFTAmount = 10
)

var networks = []network{devNetwork, qaNetwork, testNetwork, mainNetwork}

var BridgeAddresses = map[network]string{
	devNetwork:  "GDHJP6TF3UXYXTNEZ2P36J5FH7W4BJJQ4AYYAXC66I2Q2AH5B6O6BCFG",
	qaNetwork:   "GAQH7XXFBRWXT2SBK6AHPOLXDCLXVFAKFSOJIRMRNCDINWKHGI6UYVKM",
	testNetwork: "GA2CWNBUHX7NZ3B5GR4I23FMU7VY5RPA77IUJTIXTTTGKYSKDSV6LUA4",
	mainNetwork: "GBNOTAYUMXVO5QDYWYO2SOCOYIJ3XFIP65GKOQN7H65ZZSO6BK4SLWSC",
}

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
