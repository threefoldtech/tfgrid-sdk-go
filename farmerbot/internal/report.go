package internal

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/rs/zerolog/log"
)

// NodeReport is a report for some node info
type NodeReport struct {
	ID                           uint32        `json:"id"`
	State                        string        `json:"state"`
	HasActiveRentContract        bool          `json:"rented"`
	HasActiveContracts           bool          `json:"has_active_contracts"`
	Dedicated                    bool          `json:"dedicated"`
	PublicConfig                 bool          `json:"public_config"`
	CRUUsagePercentage           uint8         `json:"cru_usage_percentage"`
	SRUUsagePercentage           uint8         `json:"sru_usage_percentage"`
	MRUUsagePercentage           uint8         `json:"mru_usage_percentage"`
	HRUUsagePercentage           uint8         `json:"hru_usage_percentage"`
	TimesRandomWakeUps           int           `json:"random_wakeups"`
	SincePowerStateChanged       time.Duration `json:"since_power_state_changed"`
	SinceLastTimeAwake           time.Duration `json:"since_last_time_awake"`
	LastTimePeriodicWakeUp       time.Time     `json:"last_time_periodic_wakeup"`
	UntilClaimedResourcesTimeout time.Duration `json:"until_claimed_resources_timeout"`
}

func createNodeReport(n node) NodeReport {
	nodeID := uint32(n.ID)

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

	used, total := calculateResourceUsage(map[uint32]node{nodeID: n})

	var hruUsage uint8
	if total.hru != 0 {
		hruUsage += uint8(100 * used.hru / total.hru)
	}

	return NodeReport{
		ID:                           nodeID,
		State:                        state,
		HasActiveRentContract:        n.hasActiveRentContract,
		HasActiveContracts:           n.hasActiveContracts,
		Dedicated:                    n.dedicated,
		PublicConfig:                 n.PublicConfig.HasValue,
		CRUUsagePercentage:           uint8(100 * used.cru / total.cru),
		SRUUsagePercentage:           uint8(100 * used.sru / total.sru),
		MRUUsagePercentage:           uint8(100 * used.mru / total.mru),
		HRUUsagePercentage:           hruUsage,
		TimesRandomWakeUps:           n.timesRandomWakeUps,
		SincePowerStateChanged:       sincePowerStateChanged,
		SinceLastTimeAwake:           sinceLastTimeAwake,
		LastTimePeriodicWakeUp:       n.lastTimePeriodicWakeUp,
		UntilClaimedResourcesTimeout: untilClaimedResourcesTimeout,
	}
}

func (f *FarmerBot) report() string {
	t := table.NewWriter()
	// t.SetOutputMirror(os.Stdout)

	t.AppendHeader(table.Row{
		"ID",
		"State",
		"Rented",
		"Dedicated",
		"public config",
		"Usage",
		"Active node contracts",
		"Random wake-ups",
		"Periodic wake-up",
		"last time state changed",
		"last time awake",
		"Claimed resources timeout",
	})

	for _, node := range f.nodes {
		nodeReport := createNodeReport(node)

		periodicWakeup := "-"
		// if the node wakes up today
		if nodeReport.LastTimePeriodicWakeUp.Day() == time.Now().Day() {
			periodicWakeupTime, err := json.Marshal(wakeUpDate(nodeReport.LastTimePeriodicWakeUp))
			if err != nil {
				log.Error().Err(err).Uint32("nodeID", nodeReport.ID).Msg("failed to marshal wake up time")
			}
			periodicWakeup = string(periodicWakeupTime)
		}

		t.AppendRow([]interface{}{
			nodeReport.ID,
			nodeReport.State,
			nodeReport.HasActiveRentContract,
			nodeReport.Dedicated,
			nodeReport.PublicConfig,
			fmt.Sprintf("CRU:%d%%, SRU:%d%%,\nMRU:%d%%, HRU:%d%%", nodeReport.CRUUsagePercentage, nodeReport.SRUUsagePercentage, nodeReport.MRUUsagePercentage, nodeReport.HRUUsagePercentage),
			nodeReport.TimesRandomWakeUps,
			periodicWakeup,
			nodeReport.SincePowerStateChanged,
			nodeReport.SinceLastTimeAwake,
			nodeReport.UntilClaimedResourcesTimeout,
		})
		t.AppendSeparator()
	}

	t.SetStyle(table.StyleLight)
	return t.Render()
}
