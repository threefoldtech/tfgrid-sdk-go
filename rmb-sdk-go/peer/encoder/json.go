package encoder

import "encoding/json"

type jsonEncoder struct{}

func (e *jsonEncoder) Schema() string {
	return JSONSchema
}

func (e *jsonEncoder) Encode(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (e *jsonEncoder) Decode(data []byte, out interface{}) error {
	return json.Unmarshal(data, out)
}

// NewJSONEncoder returns a JSON encoder.
func NewJSONEncoder() Encoder {
	return &jsonEncoder{}
}
