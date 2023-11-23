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
	DefaultPeriodicWakeUPLimit = 1

	// DefaultCPUProvision number
	DefaultCPUProvision = 2

	//DefaultWakeUpThreshold default threshold to wake up a new node
	DefaultWakeUpThreshold = uint8(80)
	//MinWakeUpThreshold min threshold to wake up a new node
	MinWakeUpThreshold = uint8(50)
	//MaxWakeUpThreshold max threshold to wake up a new node
	MaxWakeUpThreshold = uint8(80)
)

const (
	mainNetwork string = "main"
	testNetwork string = "test"
	devNetwork  string = "dev"
	qaNetwork   string = "qa"
)

// SubstrateURLs for substrate urls
var SubstrateURLs = map[string][]string{
	testNetwork: {"wss://tfchain.test.grid.tf/ws", "wss://tfchain.test.grid.tf:443"},
	mainNetwork: {"wss://tfchain.grid.tf/ws", "wss://tfchain.grid.tf:443"},
	devNetwork:  {"wss://tfchain.dev.grid.tf/ws", "wss://tfchain.dev.grid.tf:443"},
	qaNetwork:   {"wss://tfchain.qa.grid.tf/ws", "wss://tfchain.qa.grid.tf:443"},
}

// RelayURLS relay urls
var RelayURLS = map[string]string{
	devNetwork:  "wss://relay.dev.grid.tf",
	testNetwork: "wss://relay.test.grid.tf",
	qaNetwork:   "wss://relay.qa.grid.tf",
	mainNetwork: "wss://relay.grid.tf",
}
