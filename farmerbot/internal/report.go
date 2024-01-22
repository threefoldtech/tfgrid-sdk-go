package internal

import (
	"os"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
)

func (f *FarmerBot) report() {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	t.AppendHeader(table.Row{
		"ID",
		"State",
		"RentContract",
		"Dedicated",
		"public config",
		"Used",
		"Claimed resources timeout",
		"Random wake-ups",
		"last time state changed",
		"last time awake",
	})

	for _, node := range f.nodes {
		var state string
		switch node.powerState {
		case on:
			state = "ON"
		case off:
			state = "OFF"
		case shuttingDown:
			state = "Shutting down"
		case wakingUp:
			state = "Waking up"
		}

		var timeoutClaimedResources time.Duration
		if time.Until(node.timeoutClaimedResources) > 0 {
			timeoutClaimedResources = time.Until(node.timeoutClaimedResources)
		}

		var lastTimePowerStateChanged time.Duration
		if !node.lastTimePowerStateChanged.IsZero() {
			lastTimePowerStateChanged = time.Since(node.lastTimePowerStateChanged)
		}

		var lastTimeAwake time.Duration
		if !node.lastTimeAwake.IsZero() {
			lastTimeAwake = time.Since(node.lastTimeAwake)
		}

		t.AppendRow([]interface{}{
			node.ID,
			state,
			node.hasActiveRentContract,
			node.dedicated,
			node.PublicConfig.HasValue,
			!node.isUnused(),
			timeoutClaimedResources,
			node.timesRandomWakeUps,
			lastTimePowerStateChanged,
			lastTimeAwake,
		})
		t.AppendSeparator()
	}

	t.SetStyle(table.StyleLight)
	t.Render()
}
