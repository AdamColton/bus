package bus

import (
	"reflect"
	"strings"
)

// RegisterMuxHandlerType is a bit of reflection magic. It takes a object and
// iterates over it's methods. Any methods that start with "Handler" will be
// registered with the ListenerMuxer. If there is a method named "ErrHandler"
// and the ListenerMuxer's ErrHandler field is nil, the field will be set to the
// method. A slice containing the arugments types of the handlers is returned.
func RegisterMuxHandlerType(lm ListenerMuxer, handlerType interface{}) ([]reflect.Type, error) {
	v := reflect.ValueOf(handlerType)
	t := v.Type()
	ms := v.NumMethod()
	var ts []reflect.Type
	for i := 0; i < ms; i++ {
		tm := t.Method(i)
		if strings.HasPrefix(tm.Name, "Handle") {
			at, err := lm.RegisterMuxHandler(v.Method(i).Interface())
			if err != nil {
				return ts, err
			}
			ts = append(ts, at)
		} else if tm.Name == "ErrHandler" {
			errHandler, ok := v.Method(i).Interface().(func(err error))
			if ok {
				lm.SetErrorHandler(errHandler)
			}
		}
	}
	return ts, nil
}

// RegisterHandlerType is a bit of reflection magic. Calls
// RegisterMuxHandlerType on the underlying ListenerMuxer and registers the
// returned types with the underlying Receiver.
func RegisterHandlerType(l Listener, handlerType interface{}) error {
	ts, err := RegisterMuxHandlerType(l, handlerType)
	if err != nil {
		return err
	}
	for _, t := range ts {
		i := reflect.New(t).Elem().Interface()
		err = l.RegisterType(i)
		if err != nil {
			return err
		}
	}
	return nil
}
