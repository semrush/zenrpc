package multiple_in_one

import "github.com/devimteam/zenrpc"

type (
	//zenrpc
	API struct {
		ServiceA //zenrpc
		ServiceB //zenrpc
		zenrpc.Service
	}

	//zenrpc:embedded
	ServiceA struct{}

	//zenrpc:embedded
	ServiceB struct{}
)

func (ServiceA) MethodA() error {
	return nil
}

func (ServiceB) MethodB() string {
	return ""
}

func init() {
	f := API{}
	f.MethodA()
}

//go:generate zenrpc -endpoint-format=snake
