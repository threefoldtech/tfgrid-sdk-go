package encoder

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
