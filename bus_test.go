package bus

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

type person struct {
	Name string
}

func (*person) TypeID32() uint32 {
	return 123
}

func TestBus(t *testing.T) {
	strCh := make(chan string)
	// hdlr will first put person.Name on the string channel then return an error
	// which will be picked up by the ErrHandler and will also be placed on the
	// string channel.
	personHdlr := func(f *person) error {
		strCh <- f.Name
		return errors.New("test error")
	}
	errHdlr := func(err error) {
		strCh <- err.Error()
	}
	personChan := make(chan *person)

	ifcCh := make(chan interface{})
	ml, err := RunNewListenerMux(ifcCh, nil, personHdlr, personChan)
	assert.NoError(t, err)
	ml.ErrHandler = errHdlr

	ifcCh <- &person{Name: "this is a test"}

	assert.Equal(t, "this is a test", <-strCh)
	assert.Equal(t, "test error", <-strCh)
	assert.Equal(t, "this is a test", (<-personChan).Name)

	ml.Stop()
}

func TestUint32RoundTrip(t *testing.T) {
	var u uint32 = 1234
	b := uint32ToSlice(u)
	assert.Equal(t, []byte{210, 4, 0, 0}, b)
	assert.Equal(t, u, sliceToUint32(b))
}
