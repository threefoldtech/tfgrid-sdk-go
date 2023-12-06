package internal

import (
	"time"
)

const (
	//timeoutRMBResponse a timeout for rmb response
	timeoutRMBResponse = time.Second * 120 // in seconds

	//timeoutUpdate a timeout for farmerbot updates
	timeoutUpdate = time.Minute * 5

	//timeoutPowerStateChange a timeout for changing nodes power
	timeoutPowerStateChange = time.Minute * 30

	// periodicWakeUpDuration is the duration for periodic wake ups
	periodicWakeUpDuration = time.Minute * 30

	defaultRandomWakeUpsAMonth = 10

	// defaultPeriodicWakeUPLimit default number of nodes will be woken up every 5 minutes during a periodic wake up
	defaultPeriodicWakeUPLimit = uint8(1)

	// defaultCPUProvision number
	defaultCPUProvision = float32(2)

	// defaultWakeUpThreshold default threshold to wake up a new node
	defaultWakeUpThreshold = uint8(80)
	// minWakeUpThreshold min threshold to wake up a new node
	minWakeUpThreshold = uint8(50)
	// MaxWakeUpThreshold max threshold to wake up a new node
	MaxWakeUpThreshold = uint8(80)
)

const (
	MainNetwork string = "main"
	TestNetwork string = "test"
	DevNetwork  string = "dev"
	QaNetwork   string = "qa"
)

// SubstrateURLs for substrate urls
var SubstrateURLs = map[string][]string{
	TestNetwork: {"wss://tfchain.test.grid.tf/ws", "wss://tfchain.test.grid.tf:443"},
	MainNetwork: {"wss://tfchain.grid.tf/ws", "wss://tfchain.grid.tf:443"},
	DevNetwork:  {"wss://tfchain.dev.grid.tf/ws", "wss://tfchain.dev.grid.tf:443"},
	QaNetwork:   {"wss://tfchain.qa.grid.tf/ws", "wss://tfchain.qa.grid.tf:443"},
}

// relayURLS relay urls
var relayURLS = map[string]string{
	DevNetwork:  "wss://relay.dev.grid.tf",
	TestNetwork: "wss://relay.test.grid.tf",
	QaNetwork:   "wss://relay.qa.grid.tf",
	MainNetwork: "wss://relay.grid.tf",
}
