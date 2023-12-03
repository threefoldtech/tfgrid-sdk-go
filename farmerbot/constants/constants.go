package constants

import (
	"time"
)

const (
	//TimeoutRMBResponse a timeout for rmb response
	TimeoutRMBResponse = time.Second * 120 // in seconds

	//TimeoutUpdate a timeout for farmerbot updates
	TimeoutUpdate = time.Minute * 5

	//TimeoutPowerStateChange a timeout for changing nodes power
	TimeoutPowerStateChange = time.Minute * 30

	// PeriodicWakeUpDuration is the duration for periodic wake ups
	PeriodicWakeUpDuration = time.Minute * 30

	DefaultRandomWakeUpsAMonth = 10

	// DefaultPeriodicWakeUPLimit default number of nodes will be woken up every 5 minutes during a periodic wake up
	DefaultPeriodicWakeUPLimit = uint8(1)

	// DefaultCPUProvision number
	DefaultCPUProvision = float32(2)

	//DefaultWakeUpThreshold default threshold to wake up a new node
	DefaultWakeUpThreshold = uint8(80)
	//MinWakeUpThreshold min threshold to wake up a new node
	MinWakeUpThreshold = uint8(50)
	//MaxWakeUpThreshold max threshold to wake up a new node
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

// RelayURLS relay urls
var RelayURLS = map[string]string{
	DevNetwork:  "wss://relay.dev.grid.tf",
	TestNetwork: "wss://relay.test.grid.tf",
	QaNetwork:   "wss://relay.qa.grid.tf",
	MainNetwork: "wss://relay.grid.tf",
}
