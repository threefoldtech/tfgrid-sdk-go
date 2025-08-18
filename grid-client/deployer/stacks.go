package deployer

var (
	DevNetwork  = "dev"
	QaNetwork   = "qa"
	TestNetwork = "test"
	MainNetwork = "main"

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

	SentryDSN = map[string]string{
		DevNetwork:  "https://af8a73b8282edc62c5b8bfa22da50acb@dev.sentry.grid.tf/4",
		TestNetwork: "https://af8a73b8282edc62c5b8bfa22da50acb@dev.sentry.grid.tf/4",
		QaNetwork:   "https://af8a73b8282edc62c5b8bfa22da50acb@dev.sentry.grid.tf/4",
		MainNetwork: "https://b16d2b5fcfbc87234bc180a4c574b45f@sentry.grid.tf/3",
	}
)
