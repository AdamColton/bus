package bus

import (
	"reflect"
	"sync"
)

// SerialBusSender combines the logic of serializing an object and placing it
// on a channel
type SerialBusSender struct {
	Ch        chan<- []byte
	Serialize func(i interface{}) ([]byte, error)
}

// Send takes a message, serializes it and places it on a channel.
func (sbs *SerialBusSender) Send(msg interface{}) error {
	b, err := sbs.Serialize(msg)
	if err != nil {
		return err
	}
	sbs.Ch <- b
	return nil
}

type SerialMultiBusSender struct {
	Chans     map[string]chan<- []byte
	Serialize func(i interface{}) ([]byte, error)
}

func (smbs *SerialMultiBusSender) Send(msg interface{}, keys ...string) error {
	b, err := smbs.Serialize(msg)
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		for _, ch := range smbs.Chans {
			ch <- b
		}
	} else {
		for _, key := range keys {
			if ch, found := smbs.Chans[key]; found {
				ch <- b
			}
		}
	}

	return nil
}

// Deserializer can deserialize a message of any type registered with it.
type Deserializer interface {
	Deserialize([]byte) (interface{}, error)
	Register(interface{}) error
}

// SerialBusReceiver takes serialized messages off a serial bus, deserializes
// them and places the deserialized objects on an object bus.
type SerialBusReceiver struct {
	In  <-chan []byte
	Out chan<- interface{}
	Deserializer
	ErrHandler func(error)
	mux        sync.Mutex
	stop       chan bool
}

// Run starts the SerialBusReceiver. It must be running to receive messages.
func (sbr *SerialBusReceiver) Run() {
	sbr.mux.Lock()
	sbr.stop = make(chan bool)
	for sbr.stop != nil {
		select {
		case <-sbr.stop:
			sbr.stop = nil
		case b := <-sbr.In:
			go sbr.handle(b)
		}
	}
	sbr.mux.Unlock()
}

// Stop the SerialBusReceiver
func (sbr *SerialBusReceiver) Stop() {
	if sbr.stop != nil {
		sbr.stop <- true
	}
}

func (sbr *SerialBusReceiver) handle(b []byte) {
	i, err := sbr.Deserialize(b)
	if err != nil {
		if sbr.ErrHandler != nil {
			sbr.ErrHandler(err)
		}
		return
	}
	sbr.Out <- i
}

// SerialBusListener combines a SerialBusReceiver with a BusListenerMux which
// takes the deserialized objects and passes them to their correct handlers.
type SerialBusListener struct {
	r *SerialBusReceiver
	l *ListenerMux
}

// NewSerialBusListener creates a SerialBusListener from a SerialBusReceiver.
// The Out channel on the SerialBusReceiver will be overridden.
func NewSerialBusListener(r *SerialBusReceiver, errHandler func(error)) *SerialBusListener {
	ch := make(chan interface{})
	r.Out = ch
	return &SerialBusListener{
		r: r,
		l: NewListenerMux(ch, errHandler),
	}
}

// RunNewSerialBusListener creates a SerialBusListener from a SerialBusReceiver,
// registers any handlers passed in and runs the SerialBusListener in a Go
// routine.
func RunNewSerialBusListener(r *SerialBusReceiver, errHandler func(error), handlers ...interface{}) (*SerialBusListener, error) {
	sbl := NewSerialBusListener(r, errHandler)
	for _, h := range handlers {
		if err := sbl.Register(h); err != nil {
			return nil, err
		}
	}
	go sbl.Run()
	return sbl, nil
}

// Run both the SerialBusReceiver and the BusListenerMux.
func (sbl *SerialBusListener) Run() {
	go sbl.r.Run()
	sbl.l.Run()
}

// Stop both the SerialBusReceiver and the BusListenerMux.
func (sbl *SerialBusListener) Stop() {
	go sbl.r.Stop()
	go sbl.l.Stop()
}

// Register a handler. The handler is registered with the BusListenerMux and the
// argument to the handler is registered with the SerialBusReceiver.
func (sbl *SerialBusListener) Register(handler interface{}) error {
	t, err := sbl.l.Register(handler)
	if err != nil {
		return err
	}
	i := reflect.New(t).Elem().Interface()
	return sbl.r.Register(i)
}
