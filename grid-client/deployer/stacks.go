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
			"wss://tfchain.dev.threefold.me/ws",
		},
		QaNetwork: {
			"wss://tfchain.qa.grid.tf/ws",
			"wss://tfchain.qa.threefold.me/ws",
		},
		TestNetwork: {
			"wss://tfchain.test.grid.tf/ws",
			"wss://tfchain.test.threefold.me/ws",
		},
		MainNetwork: {
			"wss://tfchain.grid.tf/ws",
			"wss://tfchain.be.grid.tf/ws",
			"wss://tfchain.grid.threefold.me/ws",
			"wss://tfchain.sg.grid.tf/ws",
			"wss://tfchain.us.grid.tf/ws",
			"wss://tfchain.grid.threefold.io/ws",
		},
	}

	// ProxyURLs are proxy urls
	ProxyURLs = map[string][]string{
		DevNetwork: {
			"https://gridproxy.dev.grid.tf/",
			"https://gridproxy.dev.threefold.me/",
			"https://gridproxy.dev.ninja.tf/",
		},
		TestNetwork: {
			"https://gridproxy.test.grid.tf/",
			"https://gridproxy.test.threefold.me/",
		},
		QaNetwork: {
			"https://gridproxy.qa.grid.tf/",
			"https://gridproxy.qa.threefold.me/",
			"https://gridproxy.qa.ninja.tf/",
		},
		MainNetwork: {
			"https://gridproxy.grid.tf/",
			"https://gridproxy.be.grid.tf/",
			"https://gridproxy.grid.threefold.me/",
			"https://gridproxy.sg.grid.tf/",
			"https://gridproxy.us.grid.tf/",
			"https://gridproxy.grid.threefold.io/",
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
			"https://graphql.be.grid.tf/graphql",
			"https://graphql.grid.threefold.me/graphql",
			"https://graphql.sg.grid.tf/graphql",
			"https://graphql.us.grid.tf/graphql",
			"https://graphql.grid.threefold.io/graphql",
		},
	}

	// RelayURLs relay urls
	RelayURLs = map[string][]string{
		DevNetwork: {
			"wss://relay.dev.grid.tf",
			// "wss://relay.dev.threefold.me",
		},
		TestNetwork: {
			"wss://relay.test.grid.tf",
			// "wss://relay.test.threefold.me",
		},
		QaNetwork: {
			"wss://relay.qa.grid.tf",
			// "wss://relay.qa.threefold.me",
			// "wss://relay.qa.ninja.tf",
		},
		MainNetwork: {
			"wss://relay.grid.tf",
			// "wss://relay.be.grid.tf",
			// "wss://relay.grid.threefold.me",
			// "wss://relay.sg.grid.tf",
			// "wss://relay.us.grid.tf",
			// "wss://relay.grid.threefold.io",
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
