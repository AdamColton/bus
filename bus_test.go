package bus

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"strconv"
	"strings"
	"testing"
)

type Foo struct {
	Name string
}

func (*Foo) MessageType() uint32 {
	return 123
}

func TestBus(t *testing.T) {
	strCh := make(chan string)
	// hdlr will first put Foo.Name on the string channel then return an error
	// which will be picked up by the ErrHandler and will also be placed on the
	// string channel.
	fooHdlr := func(f *Foo) error {
		strCh <- f.Name
		return errors.New("test error")
	}
	errHdlr := func(err error) {
		strCh <- err.Error()
	}
	fooChan := make(chan *Foo)

	ifcCh := make(chan interface{})
	ml, err := RunNewListenerMux(ifcCh, nil, fooHdlr, fooChan)
	assert.NoError(t, err)
	ml.ErrHandler = errHdlr

	ifcCh <- &Foo{Name: "this is a test"}

	assert.Equal(t, "this is a test", <-strCh)
	assert.Equal(t, "test error", <-strCh)
	assert.Equal(t, "this is a test", (<-fooChan).Name)

	ml.Stop()
}

func TestJSON(t *testing.T) {
	bCh := make(chan []byte)
	iCh := make(chan interface{})

	s := NewJSONBusSender(bCh)
	r := NewJSONBusReceiver(bCh, iCh)
	r.Register(strSlice(nil))
	go r.Run()

	go s.Send(strSlice{"this", "is", "a", "test"})
	assert.Equal(t, strSlice{"this", "is", "a", "test"}, <-iCh)

	r.Stop()
}

func TestJSONMultiBusSender(t *testing.T) {
	type ch struct {
		b chan []byte
		i chan interface{}
		r *JSONBusReceiver
	}

	sender := NewJSONMultiBusSender()
	chs := make([]*ch, 5)
	for i := range chs {
		iCh := make(chan interface{})
		bCh := make(chan []byte)
		r := NewJSONBusReceiver(bCh, iCh)
		chs[i] = &ch{
			b: bCh,
			i: iCh,
			r: r,
		}
		r.Register(strSlice(nil))
		go r.Run()
		sender.Add(strconv.Itoa(i), bCh)
	}

	msg := strSlice{"this", "is", "a", "test"}
	sender.Send(msg, "0")
	assert.Equal(t, msg, <-chs[0].i)

	msg = strSlice{"twas", "brillig"}
	sender.Send(msg, "3", "1", "4")
	assert.Equal(t, msg, <-chs[4].i)
	assert.Equal(t, msg, <-chs[1].i)
	assert.Equal(t, msg, <-chs[3].i)

	msg = strSlice{"calling", "all", "channels"}
	sender.Send(msg)
	for _, c := range chs {
		assert.Equal(t, msg, <-c.i)
	}

	for _, c := range chs {
		c.r.Stop()
	}
}

func TestGobMultiBusSender(t *testing.T) {
	type ch struct {
		b chan []byte
		i chan interface{}
		r *GobBusReceiver
	}

	sender := NewGobMultiBusSender()
	chs := make([]*ch, 5)
	for i := range chs {
		iCh := make(chan interface{})
		bCh := make(chan []byte)
		r := NewGobBusReceiver(bCh, iCh)
		chs[i] = &ch{
			b: bCh,
			i: iCh,
			r: r,
		}
		r.Register(strSlice(nil))
		go r.Run()
		sender.Add(strconv.Itoa(i), bCh)
	}

	msg := strSlice{"this", "is", "a", "test"}
	sender.Send(msg, "0")
	assert.Equal(t, msg, <-chs[0].i)

	msg = strSlice{"twas", "brillig"}
	sender.Send(msg, "3", "1", "4")
	assert.Equal(t, msg, <-chs[4].i)
	assert.Equal(t, msg, <-chs[1].i)
	assert.Equal(t, msg, <-chs[3].i)

	msg = strSlice{"calling", "all", "channels"}
	sender.Send(msg)
	for _, c := range chs {
		assert.Equal(t, msg, <-c.i)
	}

	for _, c := range chs {
		c.r.Stop()
	}
}

func TestGob(t *testing.T) {
	bCh := make(chan []byte)
	iCh := make(chan interface{})

	s := NewGobBusSender(bCh)
	r := NewGobBusReceiver(bCh, iCh)
	r.Register(strSlice(nil))
	go r.Run()

	go s.Send(strSlice{"this", "is", "a", "test"})
	assert.Equal(t, strSlice{"this", "is", "a", "test"}, <-iCh)
}

type signal struct{}

func (signal) MessageType() uint32 {
	return 456
}

type strSlice []string

func (strSlice) MessageType() uint32 {
	return 789
}

func TestListeners(t *testing.T) {
	strCh := make(chan string)

	tests := map[string]struct {
		handler  interface{}
		send     MessageType
		expected string
	}{
		"*Foo": {
			handler: func(f *Foo) {
				strCh <- f.Name
			},
			send:     &Foo{Name: "Foo test"},
			expected: "Foo test",
		},
		"signal": {
			handler: func(s signal) {
				strCh <- "signal test"
			},
			send:     signal{},
			expected: "signal test",
		},
		"string slice": {
			handler: func(s strSlice) {
				strCh <- strings.Join(s, "|")
			},
			send:     strSlice{"a", "b", "c"},
			expected: "a|b|c",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			bCh := make(chan []byte)
			js := NewJSONBusSender(bCh)
			_, err := RunJSONListener(bCh, nil, tc.handler)
			assert.NoError(t, err)

			go js.Send(tc.send)
			assert.Equal(t, tc.expected, <-strCh)

			bCh = make(chan []byte)
			gs := NewGobBusSender(bCh)
			_, err = RunGobListener(bCh, nil, tc.handler)
			assert.NoError(t, err)

			go gs.Send(tc.send)
			assert.Equal(t, tc.expected, <-strCh)
		})
	}
}

func TestUint32RoundTrip(t *testing.T) {
	var u uint32 = 1234
	assert.Equal(t, u, sliceToUint32(uint32ToSlice(u)))
}

type handlerObj chan string

func (ho handlerObj) HandleSignal(s signal) {
	ho <- "signal"
}

func (ho handlerObj) HandleStrSlice(s strSlice) {
	ho <- strings.Join(s, "|")
}

func (ho handlerObj) HandleFoo(f *Foo) {
	ho <- f.Name
}

func (ho handlerObj) ErrHandler(err error) {
	ho <- "Error: " + err.Error()
}

func (ho handlerObj) HandleFooErr(f *Foo) error {
	return errors.New(f.Name)
}

func TestRegisterHandlers(t *testing.T) {
	ho := make(handlerObj)
	bCh := make(chan []byte)
	js := NewJSONBusSender(bCh)
	l, err := RunJSONListener(bCh, nil)
	assert.NoError(t, err)
	l.RegisterHandlerType(ho)

	js.Send(signal{})
	assert.Equal(t, "signal", <-ho)

	js.Send(strSlice{"3", "1", "4"})
	assert.Equal(t, "3|1|4", <-ho)

	js.Send(&Foo{Name: "RegisterHandlers"})
	assert.Equal(t, "RegisterHandlers", <-ho)
	assert.Equal(t, "Error: RegisterHandlers", <-ho)

}
