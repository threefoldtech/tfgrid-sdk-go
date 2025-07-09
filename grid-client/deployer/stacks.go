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
			"wss://tfchain.dev.threefold.me/ws",
			"wss://tfchain.dev.threefold.me:443",
		},
		QaNetwork: {
			"wss://tfchain.qa.grid.tf/ws",
			"wss://tfchain.qa.grid.tf:443",
			"wss://tfchain.qa.threefold.me/ws",
			"wss://tfchain.qa.threefold.me:443",
		},
		TestNetwork: {
			"wss://tfchain.test.grid.tf/ws",
			"wss://tfchain.test.grid.tf:443",
			"wss://tfchain.test.threefold.me/ws",
			"wss://tfchain.test.threefold.me:443",
		},
		MainNetwork: {
			"wss://tfchain.grid.tf/ws",
			"wss://tfchain.grid.tf:443",
			"wss://tfchain.threefold.me/ws",
			"wss://tfchain.threefold.me:443",
		},
	}

	// ProxyURLs are proxy urls
	ProxyURLs = map[string][]string{
		DevNetwork: {
			"https://gridproxy.dev.grid.tf/",
			"https://gridproxy.dev.threefold.me/",
		},
		TestNetwork: {
			"https://gridproxy.test.grid.tf/",
			"https://gridproxy.test.threefold.me/",
		},
		QaNetwork: {
			"https://gridproxy.qa.grid.tf/",
			"https://gridproxy.qa.threefold.me/",
		},
		MainNetwork: {
			"https://gridproxy.grid.tf/",
			"https://gridproxy.threefold.me/",
		},
	}

	// GraphQlURLs for graphql urls
	GraphQlURLs = map[string][]string{
		DevNetwork: {
			"https://graphql.dev.grid.tf/graphql",
			"https://graphql.dev.threefold.me/graphql",
		},
		TestNetwork: {
			"https://graphql.test.grid.tf/graphql",
			"https://graphql.test.threefold.me/graphql",
		},
		QaNetwork: {
			"https://graphql.qa.grid.tf/graphql",
			"https://graphql.qa.threefold.me/graphql",
		},
		MainNetwork: {
			"https://graphql.grid.tf/graphql",
			"https://graphql.threefold.me/graphql",
		},
	}

	// RelayURLs relay urls
	RelayURLs = map[string][]string{
		DevNetwork: {
			"wss://relay.dev.grid.tf",
			"wss://relay.dev.threefold.me",
		},
		TestNetwork: {
			"wss://relay.test.grid.tf",
			"wss://relay.test.threefold.me",
		},
		QaNetwork: {
			"wss://relay.qa.grid.tf",
			"wss://relay.qa.threefold.me",
		},
		MainNetwork: {
			"wss://relay.grid.tf",
			"wss://relay.threefold.me",
		},
	}

	KycURLs = map[string]string{
		DevNetwork:  "https://kyc.dev.grid.tf",
		TestNetwork: "https://kyc.test.grid.tf",
		QaNetwork:   "https://kyc.qa.grid.tf",
		MainNetwork: "https://kyc.grid.tf",
	}

	SentryDSN = map[string]string{
		DevNetwork:  "https://af8a73b8282edc62c5b8bfa22da50acb@dev.sentry.grid.tf/4",
		TestNetwork: "https://af8a73b8282edc62c5b8bfa22da50acb@dev.sentry.grid.tf/4",
		QaNetwork:   "https://af8a73b8282edc62c5b8bfa22da50acb@dev.sentry.grid.tf/4",
		MainNetwork: "https://b16d2b5fcfbc87234bc180a4c574b45f@sentry.grid.tf/3",
	}
)
