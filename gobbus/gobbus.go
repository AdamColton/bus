package gobbus

import (
	"encoding/gob"
	"io"
)

// Serialize wraps the gob Encoder to encode an interface to an io.Writer.
func Serialize(w io.Writer, i interface{}) error {
	return gob.NewEncoder(w).Encode(i)
}

// Deserialize wraps the gob Decoder to read data off an io.Reader and
// deserialize it to an interface.
func Deserialize(r io.Reader, i interface{}) error {
	return gob.NewDecoder(r).Decode(i)
}
