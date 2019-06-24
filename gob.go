package bus

import (
	"encoding/gob"
	"io"
)

// SerializeGob takes an interface that fulfills MessageType and returns a
// serialization that begins with the message type uint32 followed by the gob
// serialization. It fulfills the Serialize field on SerialBusSender.
func SerializeGob(i interface{}) ([]byte, error) {
	return SerializeMessage(i, func(w io.Writer, i interface{}) error {
		return gob.NewEncoder(w).Encode(i)
	})
}

// NewDeserializeGob returns a Deserializer that can read serializations that
// begin with a message type uint32 followed by the gob serialization.
func NewDeserializeGob() Deserializer {
	return NewMessageDeserializer(func(r io.Reader, i interface{}) error {
		return gob.NewDecoder(r).Decode(i)
	})
}

// GobBusSender will serialize message using SerializeGob and place them on a
// serial channel.
type GobBusSender struct {
	sbs *SerialBusSender
}

// NewGobBusSender creates a GobBusSender from the serial channel it should send
// to.
func NewGobBusSender(ch chan<- []byte) *GobBusSender {
	return &GobBusSender{
		sbs: &SerialBusSender{
			Ch:        ch,
			Serialize: SerializeGob,
		},
	}
}

// Send a message that will be serialized with SerializeGob.
func (gb *GobBusSender) Send(msg MessageType) error {
	return gb.sbs.Send(msg)
}

type GobMultiBusSender struct {
	smbs *SerialMultiBusSender
}

func NewGobMultiBusSender() *GobMultiBusSender {
	return &GobMultiBusSender{
		smbs: &SerialMultiBusSender{
			Chans:     make(map[string]chan<- []byte),
			Serialize: SerializeGob,
		},
	}
}

func (gmb *GobMultiBusSender) Add(key string, ch chan<- []byte) {
	gmb.smbs.Chans[key] = ch
}

func (gmb *GobMultiBusSender) Delete(key string) {
	delete(gmb.smbs.Chans, key)
}

func (gmb *GobMultiBusSender) Send(msg MessageType, keys ...string) error {
	return gmb.smbs.Send(msg, keys...)
}

// GobBusReceiver can recieve messages serialized with SerializeGob.
type GobBusReceiver struct {
	r *SerialBusReceiver
}

// NewGobBusReceiver creates a GobBusReceiver that will listen for serialized
// messages on bCh, deserialize them with a Gob Deserializer and place the
// deserialized interfaces on iCh.
func NewGobBusReceiver(bCh <-chan []byte, iCh chan<- interface{}) *GobBusReceiver {
	r := newGobSerialBusReceiver(bCh)
	r.Out = iCh
	return &GobBusReceiver{
		r: r,
	}
}

// Register a MessageType with the Deserializer
func (gb *GobBusReceiver) Register(msg MessageType) {
	gb.r.Register(msg)
}

// Run the GobBusReceiver
func (gb *GobBusReceiver) Run() {
	gb.r.Run()
}

// Run the GobBusReceiver
func (gb *GobBusReceiver) Stop() {
	gb.r.Stop()
}

// GobListener will listen for messages serialized with SerializeGob and pass
// them to the correct handler.
type GobListener struct {
	l *SerialBusListener
}

func newGobSerialBusReceiver(in <-chan []byte) *SerialBusReceiver {
	return &SerialBusReceiver{
		In:           in,
		Deserializer: NewDeserializeGob(),
	}
}

// NewGobListener will create a listener on the in channel.
func NewGobListener(in <-chan []byte, errHandler func(error)) *GobListener {
	return &GobListener{
		l: NewSerialBusListener(newGobSerialBusReceiver(in), errHandler),
	}
}

// RunGobListener will create a listener on the in channel, register any
// handlers passed in and run the listener in a Go routine.
func RunGobListener(in <-chan []byte, errHandler func(error), handlers ...interface{}) (*GobListener, error) {
	sbl, err := RunNewSerialBusListener(newGobSerialBusReceiver(in), errHandler, handlers...)
	if err != nil {
		return nil, err
	}
	return &GobListener{
		l: sbl,
	}, nil
}

// Run the GobListener
func (jl *GobListener) Run() { jl.l.Run() }

// Stop the GobListener if it is running.
func (jl *GobListener) Stop() { jl.l.Stop() }

// Register a handler with GobListener.
func (jl *GobListener) Register(handler interface{}) error {
	return jl.l.Register(handler)
}
