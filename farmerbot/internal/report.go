package internal

import (
	"os"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
)

// NodeReport is a report for some node info
type NodeReport struct {
	ID                           uint32        `json:"id"`
	State                        string        `json:"state"`
	HasActiveRentContract        bool          `json:"rented"`
	Dedicated                    bool          `json:"dedicated"`
	PublicConfig                 bool          `json:"public_config"`
	Used                         bool          `json:"used"`
	TimesRandomWakeUps           int           `json:"random_wakeups"`
	SincePowerStateChanged       time.Duration `json:"since_power_state_changed"`
	SinceLastTimeAwake           time.Duration `json:"since_last_time_awake"`
	UntilClaimedResourcesTimeout time.Duration `json:"until_claimed_resources_timeout"`
}

func createNodeReport(n node) NodeReport {
	var state string
	switch n.powerState {
	case on:
		state = "ON"
	case off:
		state = "OFF"
	case shuttingDown:
		state = "Shutting down"
	case wakingUp:
		state = "Waking up"
	}

	var untilClaimedResourcesTimeout time.Duration
	if time.Until(n.timeoutClaimedResources) > 0 {
		untilClaimedResourcesTimeout = time.Until(n.timeoutClaimedResources)
	}

	var sincePowerStateChanged time.Duration
	if !n.lastTimePowerStateChanged.IsZero() {
		sincePowerStateChanged = time.Since(n.lastTimePowerStateChanged)
	}

	var sinceLastTimeAwake time.Duration
	if !n.lastTimeAwake.IsZero() {
		sinceLastTimeAwake = time.Since(n.lastTimeAwake)
	}

	return NodeReport{
		ID:                           uint32(n.ID),
		State:                        state,
		HasActiveRentContract:        n.hasActiveRentContract,
		Dedicated:                    n.dedicated,
		PublicConfig:                 n.PublicConfig.HasValue,
		Used:                         !n.isUnused(),
		TimesRandomWakeUps:           n.timesRandomWakeUps,
		SincePowerStateChanged:       sincePowerStateChanged,
		SinceLastTimeAwake:           sinceLastTimeAwake,
		UntilClaimedResourcesTimeout: untilClaimedResourcesTimeout,
	}
}

func (f *FarmerBot) report() {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	t.AppendHeader(table.Row{
		"ID",
		"State",
		"Rented",
		"Dedicated",
		"public config",
		"Used",
		"Random wake-ups",
		"last time state changed",
		"last time awake",
		"Claimed resources timeout",
	})

	for _, node := range f.nodes {
		nodeReport := createNodeReport(node)

		t.AppendRow([]interface{}{
			nodeReport.ID,
			nodeReport.State,
			nodeReport.HasActiveRentContract,
			nodeReport.Dedicated,
			nodeReport.PublicConfig,
			nodeReport.Used,
			nodeReport.TimesRandomWakeUps,
			nodeReport.SincePowerStateChanged,
			nodeReport.SinceLastTimeAwake,
			nodeReport.UntilClaimedResourcesTimeout,
		})
		t.AppendSeparator()
	}

	t.SetStyle(table.StyleLight)
	t.Render()
}
