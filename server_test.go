package zenrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

type ArithService struct{ Service }

// Invoke is as generated code from zenrpc cmd
func (as ArithService) Invoke(ctx context.Context, method string, params json.RawMessage) Response {
	resp := Response{}

	switch method {
	case "divide":
		var args = struct {
			A int `json:"a"`
			B int `json:"b"`
		}{}

		if err := json.Unmarshal(params, &args); err != nil {
			return NewResponseError(nil, InvalidParams, err.Error(), nil)
		}

		// todo set default values
		resp.Set(as.Divide(args.A, args.B))
	case "sum":
		var args = struct {
			A int `json:"a"`
			B int `json:"b"`
		}{}

		if err := json.Unmarshal(params, &args); err != nil {
			return NewResponseError(nil, InvalidParams, err.Error(), nil)
		}

		resp.Set(as.Sum(ctx, args.A, args.B))
	case "multiply":
		var args = struct {
			A int `json:"a"`
			B int `json:"b"`
		}{}

		if err := json.Unmarshal(params, &args); err != nil {
			return NewResponseError(nil, InvalidParams, err.Error(), nil)
		}

		resp.Set(as.Multiply(args.A, args.B))
	default:
		resp = NewResponseError(nil, MethodNotFound, "", nil)
	}

	return resp
}

// Sum sums two digits and returns error with error code as result and IP from context.
func (as *ArithService) Sum(ctx context.Context, a, b int) (bool, *Error) {
	return true, NewStringError(a+b, ctx.Value("IP").(string))
}

// Multiply multiples two digits and returns result.
func (as *ArithService) Multiply(a, b int) int {
	return a * b
}

type Quotient struct {
	Quo, Rem int
}

func (as *ArithService) Divide(a, b int) (quo *Quotient, err error) {
	if b == 0 {
		return nil, errors.New("divide by zero")
	} else if b == 1 {
		return nil, NewError(401, errors.New("we do not serve 1"))
	}

	return &Quotient{
		Quo: a / b,
		Rem: a % b,
	}, nil
}

func TestServer_ServeHTTP(t *testing.T) {
	s := NewServer()
	s.Register("arith", &ArithService{})
	s.Register("", &ArithService{})

	ts := httptest.NewServer(http.HandlerFunc(s.ServeHTTP))
	defer ts.Close()

	v := bytes.NewBuffer([]byte(`{"jsonrpc": "2.0", "method": "arith.divide", "params": { "a": 1, "b": 24 }, "id": 1 }`))
	res, err := http.Post(ts.URL, "application/json", v)
	if err != nil {
		log.Fatal(err)
	}

	greeting, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s", greeting)
}
