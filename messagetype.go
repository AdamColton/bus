package bus

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"unsafe"
)

// MessageType identifies a type by a uint32. The uint32 size was chosen becuase
// it should allow for plenty of message types, but uses little overhead.
type MessageType interface {
	MessageType() uint32
}

// MessageSender provides an interface for for sending anything that fulfills
// MessageType
type MessageSender interface {
	Send(MessageType) error
}

func uint32ToSlice(u uint32) []byte {
	a := *(*[4]byte)(unsafe.Pointer(&u))
	return a[:]
}

func sliceToUint32(b []byte) uint32 {
	if len(b) < 4 {
		return 0
	}
	return *(*uint32)(unsafe.Pointer(&b[0]))
}

// SerializeMessage prepends the MessageType uint32 to a slice then append the
// serialized value. It fulfills the Serialize field on SerialBusSender and
// allows the MessageType prefixing strategy to be reused for different
// serialization types.
func SerializeMessage(i interface{}, fn func(io.Writer, interface{}) error) ([]byte, error) {
	msg, ok := i.(MessageType)
	if !ok {
		return nil, errors.New("SerializeMessage requires interface to be MessageType")
	}

	buf := bytes.NewBuffer(nil)
	buf.Write(uint32ToSlice(msg.MessageType()))
	err := fn(buf, msg)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// MessageDeserializer fulfills Deserializer and uses the MessageType prefixing
// strategy.
type MessageDeserializer struct {
	types map[uint32]reflect.Type
	fn    func(io.Reader, interface{}) error
}

// NewMessageDeserializer creates a MessageDeserializer from a deserializing
// func.
func NewMessageDeserializer(fn func(io.Reader, interface{}) error) *MessageDeserializer {
	return &MessageDeserializer{
		types: make(map[uint32]reflect.Type),
		fn:    fn,
	}
}

// Register a type with the Deserializer. Fulfills the Deserializer interface.
func (d *MessageDeserializer) Register(i interface{}) error {
	msg, ok := i.(MessageType)
	if !ok {
		if i == nil {
			return errors.New("MessageDeserializer.Register) cannot register nil interface")
		}
		return errors.New("MessageDeserializer.Register) " + reflect.TypeOf(i).Name() + " does not fulfill MessageType")
	}
	d.types[msg.MessageType()] = reflect.TypeOf(msg)
	return nil
}

// Deserialize a message. Fulfills the Deserialize interface.
func (d *MessageDeserializer) Deserialize(b []byte) (interface{}, error) {
	if len(b) < 4 {
		return nil, errors.New("Message too short")
	}

	rt := d.types[sliceToUint32(b)]
	if rt == nil {
		return nil, errors.New("No type registered")
	}
	v := reflect.New(rt)
	i := v.Interface()

	err := d.fn(bytes.NewReader(b[4:]), i)
	if err != nil {
		return nil, err
	}
	return reflect.ValueOf(i).Elem().Interface(), nil
}
