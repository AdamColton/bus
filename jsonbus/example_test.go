package jsonbus_test

import (
	"fmt"
	"github.com/adamcolton/bus"
	"github.com/adamcolton/bus/jsonbus"
)

func RunSender(ch chan<- []byte) {
	s := jsonbus.NewSender(ch)
	s.Send(&person{
		Name: "Adam",
	})
	s.Send(strSlice{"what", "a", "place", "for", "a", "snark"})
	s.Send(&person{
		Name: "Stephen",
	})
	s.Send(signal{})
}

func RunReceiver(ch <-chan []byte) {
	ho := &handlerObj{
		done: make(chan bool),
	}
	l, err := jsonbus.RunListener(ch, nil)
	if err != nil {
		panic(err)
	}
	bus.RegisterHandlerType(l, ho)
	<-ho.done
}

func Example() {
	ch := make(chan []byte)
	go RunSender(ch)
	RunReceiver(ch)
	// Output:
	// Got Person: Adam
	// Got String Slice: [what a place for a snark]
	// Got Person: Stephen
	// Got Signal - closing
}

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

type handlerObj struct {
	done chan bool
}

func (ho *handlerObj) HandleSignal(s signal) {
	fmt.Println("Got Signal - closing")
	ho.done <- true
}

func (ho *handlerObj) HandleStrSlice(s strSlice) {
	fmt.Println("Got String Slice:", s)
}

func (ho *handlerObj) HandlePerson(p *person) {
	fmt.Println("Got Person:", p.Name)
}
