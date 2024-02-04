package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type DmiInfo struct {
	NodeTwinId uint32         `json:"node_twin_id" gorm:"unique;not null"`
	BIOS       BIOS           `json:"bios" gorm:"type:jsonb"`
	Baseboard  BaseBoard      `json:"baseboard" gorm:"type:jsonb"`
	Processor  ProcessorArray `json:"processor" gorm:"type:jsonb"`
	Memory     MemoryArray    `json:"memory" gorm:"type:jsonb"`
}

func (HealthReport) TableName() string {
	return "dmi_info"
}

type BIOS struct {
	Vendor  string `json:"vendor"`
	Version string `json:"version"`
}

type BaseBoard struct {
	Manufacturer string `json:"manufacturer"`
	ProductName  string `json:"product_name"`
}

type Processor struct {
	Version     string `json:"version"`
	ThreadCount string `json:"thread_count"`
}

type Memory struct {
	Manufacturer string `json:"manufacturer"`
	Type         string `json:"type"`
}

type ProcessorArray []Processor

type MemoryArray []Memory

func (p ProcessorArray) Value() (driver.Value, error) {
	return json.Marshal(p)
}

func (p *ProcessorArray) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("Invalid data type for Processor")
	}
	return json.Unmarshal(bytes, &p)
}

func (p MemoryArray) Value() (driver.Value, error) {
	return json.Marshal(p)
}

func (p *MemoryArray) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("Invalid data type for Processor")
	}
	return json.Unmarshal(bytes, &p)
}
