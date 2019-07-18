package jsonbus

import (
	"github.com/adamcolton/bus"
)

var (
	serializer   = bus.SerializeTypeID32Func(Serialize).SerializeTypeID32
	deserializer = bus.DeserializeTypeID32Func(Deserialize).NewTypeID32Deserializer
)

// NewSender creates a TypeID32Sender from the serial channel it should
// send to using the Serialize function.
func NewSender(ch chan<- []byte) bus.TypeID32Sender {
	return bus.TypeID32SerialSender{
		&bus.SerialSender{
			Ch:        ch,
			Serialize: serializer,
		},
	}
}

// NewMultiSender creates TypeID32MultiSender using the Serialize function.
func NewMultiSender() bus.TypeID32MultiSender {
	return bus.NewTypeID32MultiSender(serializer)
}

// NewReceiver creates a TypeID32Receiver using the Deserialize function.
func NewReceiver(in <-chan []byte, out chan<- interface{}) bus.TypeID32Receiver {
	return bus.TypeID32SerialReceiver{
		&bus.SerialReceiver{
			In:           in,
			Out:          out,
			Deserializer: deserializer(),
		},
	}
}

// NewListener will create a listener on the in channel using the Deserialize
// function.
func NewListener(in <-chan []byte, errHandler func(error)) bus.Listener {
	return bus.NewSerialListener(in, deserializer(), errHandler)
}

// RunListener will create a listener on the in channel using the Deserialize
// function, register any handlers passed in and run the listener in a Go
// routine.
func RunListener(in <-chan []byte, errHandler func(error), handlers ...interface{}) (bus.Listener, error) {
	return bus.RunSerialListener(in, deserializer(), errHandler, handlers...)
}
