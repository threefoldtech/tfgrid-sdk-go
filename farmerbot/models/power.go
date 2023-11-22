package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// WakeUpDate is the date to wake up all nodes
type WakeUpDate time.Time

type PowerState uint8

const (
	ON = PowerState(iota)
	WakingUP
	OFF
	ShuttingDown
)

// Power represents power configuration
type Power struct {
	// PeriodicWakeUp WakeUpDate `json:"periodicWakeUp"`
	WakeUpThreshold     uint8      `json:"wake_up_threshold"`
	PeriodicWakeUpStart WakeUpDate `json:"periodic_wake_up_start"`
	PeriodicWakeUpLimit uint8      `json:"periodic_wake_up_limit"`
}

// UnmarshalJSON unmarshal the given JSON object into wakeUp date
func (d *WakeUpDate) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	t, err := time.Parse("03:04PM", s)
	if err != nil {
		return err
	}
	*d = WakeUpDate(t)
	return nil
}

// MarshalJSON marshals the wake up date
func (d WakeUpDate) MarshalJSON() ([]byte, error) {
	date := time.Time(d)

	dayTime := "AM"
	if date.Hour() >= 12 {
		dayTime = "PM"
		date = date.Add(time.Duration(-12) * time.Hour)
	}

	timeFormat := fmt.Sprintf("%02d:%02d%s", date.Hour(), date.Minute(), dayTime)
	return json.Marshal(timeFormat)
}

// PeriodicWakeUpTime returns periodic wake up date
func (d WakeUpDate) PeriodicWakeUpTime() time.Time {
	date := time.Time(d)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return today.Local().Add(time.Hour*time.Duration(date.Hour()) +
		time.Minute*time.Duration(date.Minute()) +
		0)
}
