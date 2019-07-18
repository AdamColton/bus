package bus

import (
	"reflect"
)

// Sender handles the operations to place a message on a bus. For instance, it
// may contain the logic to serialize the message.
type Sender interface {
	Send(msg interface{}) error
}

// MultiSender will send a message to multiple busses at once. This can reduce
// duplication of work. For instance, if a message needs to be serialized, it
// will only be serialized once.
type MultiSender interface {
	Send(msg interface{}, ids ...string) error
	Add(key string, to interface{}) error
	Delete(key string)
}

// Serializer takes an interface and returns the serialization as a byte slice.
type Serializer func(i interface{}) ([]byte, error)

// Deserializer knows how to deserialize all types registered with the
// RegisterType method. It can take a byte slice and return an interface
// containing an instance to the value represented by the serialized data.
type Deserializer interface {
	Deserialize([]byte) (interface{}, error)
	RegisterType(zeroValue interface{}) error
}

// Receiver receives data from a bus transaltes it to a value and retransmits
// the value on an interface bus. For example, it may receive data as a byte
// slice, deserialize to a value and retransmit the value.
type Receiver interface {
	Run()
	Stop()
	RegisterType(zeroValue interface{}) error
	SetOut(out chan<- interface{})
	SetErrorHandler(func(error))
}

// ListenerMuxer takes values off an interface channel and multiplexes them out
// to the correct handlers for the given type.
type ListenerMuxer interface {
	RegisterMuxHandler(handler interface{}) (reflect.Type, error)
	Run()
	Stop()
	HandleError(error)
	SetErrorHandler(func(error))
	SetIn(<-chan interface{})
}

// Listener combines a Receiver and a ListenerMuxer to take in data from a
// bus, convert the data to an interface value and multiplex them out to the
// correct handlers.
type Listener interface {
	ListenerMuxer
	RegisterHandler(handler interface{}) error
	RegisterType(zeroValue interface{}) error
}
