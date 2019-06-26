package bus

import (
	"errors"
	"reflect"
	"strings"
	"sync"
)

// ListenerMux takes objects off a channel and passes them into the handlers
// for that type.
type ListenerMux struct {
	in         <-chan interface{}
	handlers   map[reflect.Type][]reflect.Value
	mux        sync.Mutex
	stop       chan bool
	ErrHandler func(error)
}

// Listener provides an interface to register handlers
type Listener interface {
	Register(handlerFunc interface{}) error
}

// NewListenerMux creates a ListenerMux for the in bus channel.
func NewListenerMux(in <-chan interface{}, errHandler func(error)) *ListenerMux {
	return &ListenerMux{
		in:         in,
		handlers:   make(map[reflect.Type][]reflect.Value),
		ErrHandler: errHandler,
	}
}

// RunNewListenerMux creates a ListenerMux for the in bus channel,
// registers any handlers passed in and runs the ListenerMux in a Go routine.
func RunNewListenerMux(in <-chan interface{}, errHandler func(error), handlers ...interface{}) (*ListenerMux, error) {
	lm := NewListenerMux(in, errHandler)
	for _, l := range handlers {
		if _, err := lm.Register(l); err != nil {
			return nil, err
		}
	}
	go lm.Run()
	return lm, nil
}

// Run the ListenerMux
func (lm *ListenerMux) Run() {
	lm.mux.Lock()
	lm.stop = make(chan bool)
	for lm.stop != nil {
		select {
		case i := <-lm.in:
			go lm.handle(i)
		case <-lm.stop:
			lm.stop = nil
		}
	}

	lm.mux.Unlock()
}

// Stop the ListenerMux if it's running
func (lm *ListenerMux) Stop() {
	if lm.stop != nil {
		go func() { lm.stop <- true }()
	}
}

func (lm *ListenerMux) handle(i interface{}) {
	v := []reflect.Value{reflect.ValueOf(i)}
	for _, l := range lm.handlers[v[0].Type()] {
		out := l.Call(v)
		if l := len(out); l > 0 {
			err, ok := out[l-1].Interface().(error)
			if ok && err != nil && lm.ErrHandler != nil {
				lm.ErrHandler(err)
			}
		}
	}
}

// Register a handler with ListenerMux. It will return the argument type for
// the handler.
func (lm *ListenerMux) Register(handler interface{}) (reflect.Type, error) {
	v := reflect.ValueOf(handler)

	switch v.Kind() {
	case reflect.Func:
		return lm.registerFunc(v)
	case reflect.Chan:
		return lm.registerChan(v)
	}
	return nil, errors.New("Register requires a func or a channel")
}

func (lm *ListenerMux) registerFunc(v reflect.Value) (reflect.Type, error) {
	t := v.Type()
	if t.NumIn() != 1 {
		return nil, errors.New("Can only register a func with exactly one argument: " + t.String())
	}
	argType := t.In(0)
	lm.handlers[argType] = append(lm.handlers[argType], v)
	return argType, nil
}

func (lm *ListenerMux) registerChan(v reflect.Value) (reflect.Type, error) {
	t := v.Type()
	argType := t.Elem()
	fn := func(i interface{}) {
		v.Send(reflect.ValueOf(i))
	}
	lm.handlers[argType] = append(lm.handlers[argType], reflect.ValueOf(fn))
	return argType, nil
}

// RegisterHandlerType is a bit of reflection magic. It takes a object and
// iterates over it's methods. Any methods that start with "Handler" will be
// registered with the ListenerMux. If there is a method named "ErrHandler" and
// the ListenerMux's ErrHandler field is nil, the field will be set to the
// method. A slice containing the arugments types of the handlers is returned.
func (lm *ListenerMux) RegisterHandlerType(obj interface{}) ([]reflect.Type, error) {
	v := reflect.ValueOf(obj)
	t := v.Type()
	ms := v.NumMethod()
	var ts []reflect.Type
	for i := 0; i < ms; i++ {
		tm := t.Method(i)
		if strings.HasPrefix(tm.Name, "Handle") {
			at, err := lm.registerFunc(v.Method(i))
			if err != nil {
				return ts, err
			}
			ts = append(ts, at)
		} else if tm.Name == "ErrHandler" && lm.ErrHandler == nil {
			vm := v.Method(i)
			lm.ErrHandler = func(err error) {
				vm.Call([]reflect.Value{reflect.ValueOf(err)})
			}
		}
	}
	return ts, nil
}
