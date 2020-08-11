package subarithservice

import (
	"context"
	"errors"
	"github.com/semrush/zenrpc/v2"
	"github.com/semrush/zenrpc/v2/testdata/model"
	"math"
)

type SubArithService struct{} //zenrpc

// Sum sums two digits and returns error with error code as result and IP from context.
func (as SubArithService) Sum(ctx context.Context, a, b int) (bool, *zenrpc.Error) {
	r, _ := zenrpc.RequestFromContext(ctx)

	return true, zenrpc.NewStringError(a+b, r.Host)
}

func (as SubArithService) Positive() (bool, *zenrpc.Error) {
	return true, nil
}

func (SubArithService) ReturnPointFromSamePackage(p Point) Point {
	// some optimistic operations
	return Point{}
}

func (SubArithService) GetPoints() []model.Point {
	return []model.Point{}
}

func (SubArithService) GetPointsFromSamePackage() []Point {
	return []Point{}
}

func (SubArithService) DoSomethingWithPoint(p model.Point) model.Point {
	// some optimistic operations
	return p
}

// Multiply multiples two digits and returns result.
func (as SubArithService) Multiply(a, b int) int {
	return a * b
}

// CheckError throws error is isErr true.
//zenrpc:500 test error
func (SubArithService) CheckError(isErr bool) error {
	if isErr {
		return errors.New("test")
	}

	return nil
}

// CheckError throws zenrpc error is isErr true.
//zenrpc:500 test error
func (SubArithService) CheckZenRPCError(isErr bool) *zenrpc.Error {
	if isErr {
		return zenrpc.NewStringError(500, "test")
	}

	return nil
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
func (as *SubArithService) Divide(a, b int) (quo *Quotient, err error) {
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
//zenrpc:exp=2 	exponent could be empty
func (as *SubArithService) Pow(base float64, exp *float64) float64 {
	return math.Pow(base, *exp)
}

// PI returns math.Pi.
func (SubArithService) Pi() float64 {
	return math.Pi
}

// SumArray returns sum all items from array
//zenrpc:array=[]float64{1,2,4}
func (as *SubArithService) SumArray(array *[]float64) float64 {
	var sum float64

	for _, i := range *array {
		sum += i
	}
	return sum
}

//go:generate zenrpc
