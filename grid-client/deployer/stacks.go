package deployer

var (
	DevNetwork  = "dev"
	QaNetwork   = "qa"
	TestNetwork = "test"
	MainNetwork = "main"

	// SubstrateURLs are substrate urls
	SubstrateURLs = map[string][]string{
		DevNetwork: {
			"wss://tfchain.dev.grid.tf/ws",
			"wss://tfchain.dev.grid.tf:443",
			"wss://tfchain.02.dev.grid.tf/ws",
			"wss://tfchain.02.dev.grid.tf:443",
		},
		QaNetwork: {
			"wss://tfchain.qa.grid.tf/ws",
			"wss://tfchain.qa.grid.tf:443",
			"wss://tfchain.02.qa.grid.tf/ws",
			"wss://tfchain.02.qa.grid.tf:443",
		},
		TestNetwork: {
			"wss://tfchain.test.grid.tf/ws",
			"wss://tfchain.test.grid.tf:443",
			"wss://tfchain.02.test.grid.tf/ws",
			"wss://tfchain.02.test.grid.tf:443",
		},
		MainNetwork: {
			"wss://tfchain.grid.tf/ws",
			"wss://tfchain.grid.tf:443",
			"wss://tfchain.02.grid.tf/ws",
			"wss://tfchain.02.grid.tf:443",
		},
	}

	// ProxyURLs are proxy urls
	ProxyURLs = map[string][]string{
		DevNetwork: {
			"https://gridproxy.dev.grid.tf/",
			"https://gridproxy.02.dev.grid.tf/",
		},
		TestNetwork: {
			"https://gridproxy.test.grid.tf/",
			"https://gridproxy.02.test.grid.tf/",
		},
		QaNetwork: {
			"https://gridproxy.qa.grid.tf/",
			"https://gridproxy.02.qa.grid.tf/",
		},
		MainNetwork: {
			"https://gridproxy.grid.tf/",
			"https://gridproxy.02.grid.tf/",
		},
	}

	// GraphQlURLs for graphql urls
	GraphQlURLs = map[string][]string{
		DevNetwork: {
			"https://graphql.dev.grid.tf/graphql",
			"https://graphql.02.dev.grid.tf/graphql",
		},
		TestNetwork: {
			"https://graphql.test.grid.tf/graphql",
			"https://graphql.02.test.grid.tf/graphql",
		},
		QaNetwork: {
			"https://graphql.qa.grid.tf/graphql",
			"https://graphql.02.qa.grid.tf/graphql",
		},
		MainNetwork: {
			"https://graphql.grid.tf/graphql",
			"https://graphql.02.grid.tf/graphql",
		},
	}

	// RelayURLs relay urls
	RelayURLs = map[string][]string{
		DevNetwork: {
			"wss://relay.dev.grid.tf",
			"wss://relay.02.dev.grid.tf",
		},
		TestNetwork: {
			"wss://relay.test.grid.tf",
			"wss://relay.02.test.grid.tf",
		},
		QaNetwork: {
			"wss://relay.qa.grid.tf",
			"wss://relay.02.qa.grid.tf",
		},
		MainNetwork: {
			"wss://relay.grid.tf",
			"wss://relay.02.grid.tf",
		},
	}
)
