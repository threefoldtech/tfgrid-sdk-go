package encoder

import (
	"fmt"
)

// Encoder interface for encoding data
type Encoder interface {
	Schema() string
	Encode(interface{}) ([]byte, error)
	Decode([]byte, interface{}) error
}

type Schema string

// supported schemas
const (
	JSONSchema    = "application/json"
	DefaultSchema = JSONSchema
)

// NewEncoder returns an encoder based on the given schema
func NewEncoder(schema Schema) (Encoder, error) {
	switch schema {
	case JSONSchema:
		return newJSONEncoder(), nil
	default:
		return nil, fmt.Errorf("invalid encoder schema %q", schema)
	}
}
