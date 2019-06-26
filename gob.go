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

// GobMultiBusSender allows a single message to be sent to multiple channels
type GobMultiBusSender struct {
	smbs *SerialMultiBusSender
}

// NewGobMultiBusSender creates a GobMultiBusSender
func NewGobMultiBusSender() *GobMultiBusSender {
	return &GobMultiBusSender{
		smbs: &SerialMultiBusSender{
			Chans:     make(map[string]chan<- []byte),
			Serialize: SerializeGob,
		},
	}
}

// Add a channel by id.
func (gmb *GobMultiBusSender) Add(id string, ch chan<- []byte) {
	gmb.smbs.Chans[id] = ch
}

// Delete a channel by id.
func (gmb *GobMultiBusSender) Delete(id string) {
	delete(gmb.smbs.Chans, id)
}

// Send a message to all of the given ids. If no ids are given, the message will
// be sent to all channels.
func (gmb *GobMultiBusSender) Send(msg MessageType, id ...string) error {
	return gmb.smbs.Send(msg, id...)
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

// Stop the GobBusReceiver
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
func (gl *GobListener) Run() { gl.l.Run() }

// Stop the GobListener if it is running.
func (gl *GobListener) Stop() { gl.l.Stop() }

// Register a handler with GobListener.
func (gl *GobListener) Register(handler interface{}) error {
	return gl.l.Register(handler)
}

// RegisterHandlerType registers a Handler Object, see
// ListenerMux.RegisterHandlerType for more information.
func (gl *GobListener) RegisterHandlerType(obj interface{}) {
	gl.l.RegisterHandlerType(obj)
}
