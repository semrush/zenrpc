package testdata

import (
	"context"
	"errors"
	"github.com/sergeyfast/zenrpc"
	"math"
)

type ArithService struct{ zenrpc.Service }

// Sum sums two digits and returns error with error code as result and IP from context.
func (as ArithService) Sum(ctx context.Context, a, b int) (bool, *zenrpc.Error) {
	r, _ := zenrpc.RequestFromContext(ctx)

	return true, zenrpc.NewStringError(a+b, r.Host)
}

func (as ArithService) Positive() (bool, *zenrpc.Error) {
	return true, nil
}

// Multiply multiples two digits and returns result.
func (as ArithService) Multiply(a, b int) int {
	return a * b
}

// Quotient docs
type Quotient struct {
	// Quo docs
	Quo int

	// Rem docs
	Rem int `json:"rem"`
}

// Divide divides two numbers.
//zenrpc:a			the a
//zenrpc:b 			the b
//zenrpc:quo		result is Quotient, should be named var
//zenrpc:401 		we do not serve 1
//zenrpc:-32603		divide by zero
func (as *ArithService) Divide(a, b int) (quo *Quotient, err error) {
	if b == 0 {
		return nil, errors.New("divide by zero")
	} else if b == 1 {
		return nil, zenrpc.NewError(401, errors.New("we do not serve 1"))
	}

	return &Quotient{
		Quo: a / b,
		Rem: a % b,
	}, nil
}

// Pow returns x**y, the base-x exponential of y. If Exp is not set then default value is 2.
//zenrpc:exp:2 	exponent could be empty
func (as *ArithService) Pow(base float64, exp *float64) float64 {
	return math.Pow(base, *exp)
}

// PI returns math.Pi.
func (ArithService) Pi() float64 {
	return math.Pi
}

// SumArray returns sum all items from array
//zenrpc:array:[]float64{1,2,4}
func (as *ArithService) SumArray(array *[]float64) float64 {
	var sum float64

	for _, i := range *array {
		sum += i
	}
	return sum
}

//go:generate zenrpc
