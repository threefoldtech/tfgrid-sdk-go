package models

import (
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
	WakeUpThreshold     uint8      `json:"wake_up_threshold" yaml:"wake_up_threshold" toml:"wake_up_threshold"`
	PeriodicWakeUpStart WakeUpDate `json:"periodic_wake_up_start" yaml:"periodic_wake_up_start" toml:"periodic_wake_up_start"`
	PeriodicWakeUpLimit uint8      `json:"periodic_wake_up_limit" yaml:"periodic_wake_up_limit" toml:"periodic_wake_up_limit"`
}

// UnmarshalJSON unmarshal the given JSON string into wakeUp date
func (d *WakeUpDate) UnmarshalJSON(b []byte) error {
	return d.Unmarshal(b)
}

// MarshalJSON marshals the wake up date
func (d WakeUpDate) MarshalJSON() ([]byte, error) {
	return d.Marshal()
}

// UnmarshalYAML unmarshal the given yaml string into wakeUp date
func (d *WakeUpDate) UnmarshalYAML(s string) error {
	return d.Unmarshal([]byte(s))
}

// MarshalYAML marshals the wake up date
func (d WakeUpDate) MarshalYAML() ([]byte, error) {
	return d.Marshal()
}

// UnmarshalText unmarshal the given TOML string into wakeUp date
func (d *WakeUpDate) UnmarshalText(b []byte) error {
	return d.Unmarshal(b)
}

// MarshalText marshals the wake up TOML date
func (d WakeUpDate) MarshalText() ([]byte, error) {
	return d.Marshal()
}

// Unmarshal unmarshal the given string into wakeUp date
func (d *WakeUpDate) Unmarshal(b []byte) error {
	s := strings.Trim(string(b), "\"")
	t, err := time.Parse("03:04PM", s)
	if err != nil {
		return err
	}
	*d = WakeUpDate(t)
	return nil
}

// Marshal marshals the wake up date
func (d WakeUpDate) Marshal() ([]byte, error) {
	date := time.Time(d)

	dayTime := "AM"
	if date.Hour() >= 12 {
		dayTime = "PM"
		date = date.Add(time.Duration(-12) * time.Hour)
	}

	timeFormat := fmt.Sprintf("%02d:%02d%s", date.Hour(), date.Minute(), dayTime)
	return []byte(timeFormat), nil
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
