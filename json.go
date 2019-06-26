package bus

import (
	"encoding/json"
	"io"
)

// SerializeJSON takes an interface that fulfills MessageType and returns a
// serialization that begins with the message type uint32 followed by the json
// serialization. It fulfills the Serialize field on SerialBusSender.
func SerializeJSON(i interface{}) ([]byte, error) {
	return SerializeMessage(i, func(w io.Writer, i interface{}) error {
		return json.NewEncoder(w).Encode(i)
	})
}

// NewDeserializeJSON returns a Deserializer that can read serializations that
// begin with a message type uint32 followed by the json serialization.
func NewDeserializeJSON() Deserializer {
	return NewMessageDeserializer(func(r io.Reader, i interface{}) error {
		return json.NewDecoder(r).Decode(i)
	})
}

// JSONBusSender will serialize message using SerializeJSON and place them on a
// serial channel.
type JSONBusSender struct {
	sbs *SerialBusSender
}

// NewJSONBusSender creates a JSONBusSender from the serial channel it should
// send to.
func NewJSONBusSender(ch chan<- []byte) *JSONBusSender {
	return &JSONBusSender{
		sbs: &SerialBusSender{
			Ch:        ch,
			Serialize: SerializeJSON,
		},
	}
}

// Send a message that will be serialized with SerializeJSON.
func (jb *JSONBusSender) Send(msg MessageType) error {
	return jb.sbs.Send(msg)
}

// JSONMultiBusSender allows a single message to be sent to multiple channels
type JSONMultiBusSender struct {
	smbs *SerialMultiBusSender
}

// NewJSONMultiBusSender creates a JSONMultiBusSender
func NewJSONMultiBusSender() *JSONMultiBusSender {
	return &JSONMultiBusSender{
		smbs: &SerialMultiBusSender{
			Chans:     make(map[string]chan<- []byte),
			Serialize: SerializeJSON,
		},
	}
}

// Add a channel by id.
func (jmb *JSONMultiBusSender) Add(key string, ch chan<- []byte) {
	jmb.smbs.Chans[key] = ch
}

// Delete a channel by id.
func (jmb *JSONMultiBusSender) Delete(key string) {
	delete(jmb.smbs.Chans, key)
}

// Send a message to all of the given ids. If no ids are given, the message will
// be sent to all channels.
func (jmb *JSONMultiBusSender) Send(msg MessageType, keys ...string) error {
	return jmb.smbs.Send(msg, keys...)
}

// JSONBusReceiver can recieve messages serialized with SerializeJSON.
type JSONBusReceiver struct {
	r *SerialBusReceiver
}

// NewJSONBusReceiver creates a JSONBusReceiver that will listen for serialized
// messages on bCh, deserialize them with a JSON Deserializer and place the
// deserialized interfaces on iCh.
func NewJSONBusReceiver(bCh <-chan []byte, iCh chan<- interface{}) *JSONBusReceiver {
	r := newJSONSerialBusReceiver(bCh)
	r.Out = iCh
	return &JSONBusReceiver{
		r: r,
	}
}

// Register a MessageType with the Deserializer
func (jb *JSONBusReceiver) Register(msg MessageType) {
	jb.r.Register(msg)
}

// Run the JSONBusReceiver
func (jb *JSONBusReceiver) Run() {
	jb.r.Run()
}

// Stop the JSONBusReceiver
func (jb *JSONBusReceiver) Stop() {
	jb.r.Stop()
}

// JSONListener will listen for messages serialized with SerializeJSON and pass
// them to the correct handler.
type JSONListener struct {
	l *SerialBusListener
}

func newJSONSerialBusReceiver(in <-chan []byte) *SerialBusReceiver {
	return &SerialBusReceiver{
		In:           in,
		Deserializer: NewDeserializeJSON(),
	}
}

// NewJSONListener will create a listener on the in channel.
func NewJSONListener(in <-chan []byte, errHandler func(error)) *JSONListener {
	return &JSONListener{
		l: NewSerialBusListener(newJSONSerialBusReceiver(in), errHandler),
	}
}

// RunJSONListener will create a listener on the in channel, register any
// handlers passed in and run the listener in a Go routine.
func RunJSONListener(in <-chan []byte, errHandler func(error), handlers ...interface{}) (*JSONListener, error) {
	sbl, err := RunNewSerialBusListener(newJSONSerialBusReceiver(in), errHandler, handlers...)
	if err != nil {
		return nil, err
	}
	return &JSONListener{
		l: sbl,
	}, nil
}

// Run the JSONListener
func (jl *JSONListener) Run() { jl.l.Run() }

// Stop the JSONListener if it is running.
func (jl *JSONListener) Stop() { jl.l.Stop() }

// Register a handler with JSONListener.
func (jl *JSONListener) Register(handler interface{}) error {
	return jl.l.Register(handler)
}

// RegisterHandlerType registers a Handler Object, see
// ListenerMux.RegisterHandlerType for more information.
func (jl *JSONListener) RegisterHandlerType(obj interface{}) {
	jl.l.RegisterHandlerType(obj)
}
