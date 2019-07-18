package jsonbus

import (
	"errors"
	"github.com/adamcolton/bus"
	"github.com/stretchr/testify/assert"
	"strconv"
	"strings"
	"testing"
)

type person struct {
	Name string
}

func (*person) TypeID32() uint32 {
	return 123
}

type strSlice []string

func (strSlice) TypeID32() uint32 {
	return 789
}

type signal struct{}

func (signal) TypeID32() uint32 {
	return 456
}

type handlerObj chan string

func (ho handlerObj) HandleSignal(s signal) {
	ho <- "signal"
}

func (ho handlerObj) HandleStrSlice(s strSlice) {
	ho <- strings.Join(s, "|")
}

func (ho handlerObj) HandleFoo(p *person) {
	ho <- p.Name
}

func (ho handlerObj) ErrHandler(err error) {
	ho <- "Error: " + err.Error()
}

func (ho handlerObj) HandleFooErr(p *person) error {
	return errors.New(p.Name)
}

func TestSendReceive(t *testing.T) {
	bCh := make(chan []byte)
	iCh := make(chan interface{})

	s := NewSender(bCh)
	r := NewReceiver(bCh, iCh)
	r.RegisterType(strSlice(nil))
	go r.Run()

	go s.Send(strSlice{"this", "is", "a", "test"})
	assert.Equal(t, strSlice{"this", "is", "a", "test"}, <-iCh)

	r.Stop()
}

func TestMultiSender(t *testing.T) {
	type ch struct {
		b chan []byte
		i chan interface{}
		r bus.TypeID32Receiver
	}

	sender := NewMultiSender()
	chs := make([]*ch, 5)
	for i := range chs {
		iCh := make(chan interface{})
		bCh := make(chan []byte)
		r := NewReceiver(bCh, iCh)
		chs[i] = &ch{
			b: bCh,
			i: iCh,
			r: r,
		}
		r.RegisterType(strSlice(nil))
		go r.Run()
		if !assert.NoError(t, sender.Add(strconv.Itoa(i), bCh)) {
			return
		}
	}

	msg := strSlice{"this", "is", "a", "test"}
	assert.NoError(t, sender.Send(msg, "0"))
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

func TestListeners(t *testing.T) {
	strCh := make(chan string)

	tests := map[string]struct {
		handler  interface{}
		send     bus.TypeIDer32
		expected string
	}{
		"*person": {
			handler: func(p *person) {
				strCh <- p.Name
			},
			send:     &person{Name: "person test"},
			expected: "person test",
		},
		"signal": {
			handler: func(s signal) {
				strCh <- "signal test"
			},
			send:     signal{},
			expected: "signal test",
		},
		"strSlice": {
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
			s := NewSender(bCh)
			l, err := RunListener(bCh, nil, tc.handler)
			assert.NoError(t, err)

			go s.Send(tc.send)
			assert.Equal(t, tc.expected, <-strCh)

			l.Stop()
		})
	}
}

func TestRegisterHandlers(t *testing.T) {
	ho := make(handlerObj)
	bCh := make(chan []byte)
	s := NewSender(bCh)
	l, err := RunListener(bCh, nil)
	assert.NoError(t, err)
	bus.RegisterHandlerType(l, ho)

	s.Send(signal{})
	assert.Equal(t, "signal", <-ho)

	s.Send(strSlice{"3", "1", "4"})
	assert.Equal(t, "3|1|4", <-ho)

	s.Send(&person{Name: "RegisterHandlers"})
	assert.Equal(t, "RegisterHandlers", <-ho)
	assert.Equal(t, "Error: RegisterHandlers", <-ho)
	l.Stop()
}
