package jsonbus

import (
	"encoding/json"
	"io"
)

// Serialize wraps the json Encoder to encode an interface to an io.Writer.
func Serialize(w io.Writer, i interface{}) error {
	return json.NewEncoder(w).Encode(i)
}

// Deserialize wraps the json Decoder to read data off an io.Reader and
// deserialize it to an interface.
func Deserialize(r io.Reader, i interface{}) error {
	return json.NewDecoder(r).Decode(i)
}
