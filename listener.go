package bus

import (
	"reflect"
)

type listener struct {
	Receiver
	ListenerMuxer
}

// NewListener creates a Listener from a Receiver and a ListenerMuxer. It
// connects the Out channel on the Receiver to the In channel on the
// ListenerMuxer. If errHandler is not null it will be set on both the Receiver
// and ListenerMuxer.
func NewListener(r Receiver, lm ListenerMuxer, errHandler func(error)) Listener {
	ch := make(chan interface{})
	r.SetOut(ch)
	lm.SetIn(ch)
	lm.SetErrorHandler(errHandler)
	r.SetErrorHandler(errHandler)
	return &listener{
		Receiver:      r,
		ListenerMuxer: lm,
	}
}

// RunNewListener creates a Listener from a Receiver and a ListenerMuxer. It
// connects the Out channel on the Receiver to the In channel on the
// ListenerMuxer. If errHandler is not null it will be set on both the Receiver
// and ListenerMuxer. It registers the supplied handers and starts running both
// the Receiver and ListenerMuxer in a Go routine.
func RunNewListener(r Receiver, lm ListenerMuxer, errHandler func(error), handlers ...interface{}) (Listener, error) {
	l := NewListener(r, lm, errHandler)
	for _, h := range handlers {
		if err := l.RegisterHandler(h); err != nil {
			return nil, err
		}
	}
	go l.Run()
	return l, nil
}

// NewSerialListener creates a Listener that reads from the in channel,
// deserializes to a value and passes the value to a ListenerMuxer.
func NewSerialListener(in <-chan []byte, d Deserializer, errHandler func(error)) Listener {
	r := &SerialReceiver{
		In:           in,
		Deserializer: d,
	}
	return NewListener(r, NewListenerMux(nil), errHandler)
}

// RunSerialListener creates a Listener that reads from the in channel,
// deserializes to a value and passes the value to a ListenerMuxer. It regsiters
// the handlers and starts both the Receiver and ListenerMuxer running in a Go
// routine.
func RunSerialListener(in <-chan []byte, d Deserializer, errHandler func(error), handlers ...interface{}) (Listener, error) {
	r := &SerialReceiver{
		In:           in,
		Deserializer: d,
	}
	return RunNewListener(r, NewListenerMux(nil), errHandler, handlers...)
}

// Run both the Receiver and the ListenerMuxer.
func (l *listener) Run() {
	go l.Receiver.Run()
	l.ListenerMuxer.Run()
}

// Stop both the Receiver and the ListenerMuxer.
func (l *listener) Stop() {
	go l.Receiver.Stop()
	go l.ListenerMuxer.Stop()
}

// RegisterHandler with the underlying ListenerMuxer and the argument to the
// handler is registered with the Receiver.
func (l *listener) RegisterHandler(handler interface{}) error {
	t, err := l.RegisterMuxHandler(handler)
	if err != nil {
		return err
	}
	zeroVal := reflect.New(t).Elem().Interface()
	return l.RegisterType(zeroVal)
}

// SetErrorHandler on both the underlying ListenerMuxer and Receiver.
func (l *listener) SetErrorHandler(errHandler func(error)) {
	l.ListenerMuxer.SetErrorHandler(errHandler)
	l.Receiver.SetErrorHandler(errHandler)
}
