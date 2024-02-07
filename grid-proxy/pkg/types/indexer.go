package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type NetworkTestResult struct {
	NodeTwinId    uint32  `json:"node_twin_id" gorm:"unique;not null"`
	UploadSpeed   float64 `json:"upload_speed"`   // in bit/sec
	DownloadSpeed float64 `json:"download_speed"` // in bit/sec
}

type PerfResult struct {
	NodeTwinId uint32              `json:"node_twin_id"`
	Result     []NetworkTestResult `json:"result"`
}

type DmiInfo struct {
	NodeTwinId uint32     `json:"node_twin_id,omitempty" gorm:"unique;not null"`
	BIOS       BIOS       `json:"bios" gorm:"type:jsonb"`
	Baseboard  Baseboard  `json:"baseboard" gorm:"type:jsonb"`
	Processor  Processors `json:"processor" gorm:"type:jsonb"`
	Memory     Memories   `json:"memory" gorm:"type:jsonb"`
}

type BIOS struct {
	Vendor  string `json:"vendor"`
	Version string `json:"version"`
}

type Baseboard struct {
	Manufacturer string `json:"manufacturer"`
	ProductName  string `json:"product_name"`
}

type Processor struct {
	Version     string `json:"version"`
	ThreadCount string `json:"thread_count"`
}
type Processors []Processor

type Memory struct {
	Manufacturer string `json:"manufacturer"`
	Type         string `json:"type"`
}
type Memories []Memory

/*
GORM directly maps the structs to tables. These structs can contain fields with basic Go types,
pointers or aliases of these types, or even custom types, as long as they implement the Scanner
and Valuer interfaces from the database/sql package.

Notes:
	- For simple types like the BIOS struct, we can directly implement the Scan/Value methods.
		However, for types like []Processor, we need to create an alias, Processors,
		so we have method receivers.
*/

func (c Processors) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func (c *Processors) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("Invalid data type for Processor")
	}
	return json.Unmarshal(bytes, &c)
}

func (c Memories) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func (c *Memories) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("Invalid data type for Processor")
	}
	return json.Unmarshal(bytes, &c)
}

func (c *BIOS) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func (c *BIOS) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("Invalid data type for Processor")
	}
	return json.Unmarshal(bytes, &c)
}

func (c *Baseboard) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func (c *Baseboard) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("Invalid data type for Processor")
	}
	return json.Unmarshal(bytes, &c)
}
