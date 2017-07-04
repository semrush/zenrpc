package zenrpc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/sergeyfast/zenrpc"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
)

type ArithService struct{ zenrpc.Service }

// Invoke is as generated code from zenrpc cmd
func (as ArithService) Invoke(ctx context.Context, method string, params json.RawMessage) zenrpc.Response {
	resp := zenrpc.Response{}

	switch method {
	case "divide":
		var args = struct {
			A int `json:"a"`
			B int `json:"b"`
		}{}

		if err := json.Unmarshal(params, &args); err != nil {
			return zenrpc.NewResponseError(nil, zenrpc.InvalidParams, err.Error(), nil)
		}

		// todo set default values
		resp.Set(as.Divide(args.A, args.B))
	case "sum":
		var args = struct {
			A int `json:"a"`
			B int `json:"b"`
		}{}

		if err := json.Unmarshal(params, &args); err != nil {
			return zenrpc.NewResponseError(nil, zenrpc.InvalidParams, err.Error(), nil)
		}

		resp.Set(as.Sum(ctx, args.A, args.B))
	case "multiply":
		var args = struct {
			A int `json:"a"`
			B int `json:"b"`
		}{}

		if err := json.Unmarshal(params, &args); err != nil {
			return zenrpc.NewResponseError(nil, zenrpc.InvalidParams, err.Error(), nil)
		}

		resp.Set(as.Multiply(args.A, args.B))
	case "pow":
		var args = struct {
			Base float64  `json:"base"`
			Exp  *float64 `json:"exp"`
		}{}

		if err := json.Unmarshal(params, &args); err != nil {
			return zenrpc.NewResponseError(nil, zenrpc.InvalidParams, err.Error(), nil)
		}

		//zenrpc:exp:2
		if args.Exp == nil {
			var f float64 = 2
			args.Exp = &f
		}

		resp.Set(as.Pow(args.Base, args.Exp))
	default:
		resp = zenrpc.NewResponseError(nil, zenrpc.MethodNotFound, "", nil)
	}

	return resp
}

// Sum sums two digits and returns error with error code as result and IP from context.
func (as *ArithService) Sum(ctx context.Context, a, b int) (bool, *zenrpc.Error) {
	return true, zenrpc.NewStringError(a+b, ctx.Value("IP").(string))
}

// Multiply multiples two digits and returns result.
func (as *ArithService) Multiply(a, b int) int {
	return a * b
}

type Quotient struct {
	Quo, Rem int
}

// Divide divides two numbers.
// Possible error codes are:
// zenrpc:401 		we do not serve 1
// zenrpc:-32603	divide by zero
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
//zenrpc:exp:2
func (as *ArithService) Pow(base float64, exp *float64) float64 {
	return math.Pow(base, *exp)
}

var rpc = zenrpc.NewServer()

func init() {
	rpc.Register("arith", &ArithService{})
	rpc.Register("", &ArithService{})
}

func TestServer_ServeHTTP(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(rpc.ServeHTTP))
	defer ts.Close()

	var tc = []struct {
		in, out string
	}{
		{
			in:  `{"jsonrpc": "2.0", "method": "arith.divide", "params": { "a": 1, "b": 24 }, "id": 1 }`,
			out: `{"jsonrpc":"2.0","id":1,"result":{"Quo":0,"Rem":1}}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "arith.divide", "params": { "a": 1, "b": 0 }, "id": 1 }`,
			out: `{"jsonrpc":"2.0","id":1,"error":{"code":-32603,"message":"divide by zero"}}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "Arith.Divide", "params": { "a": 1, "b": 1 }, "id": "1" }`,
			out: `{"jsonrpc":"2.0","id":"1","error":{"code":401,"message":"we do not serve 1"}}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "arith.multiply", "params": { "a": 3, "b": 2 }, "id": 0 }`,
			out: `{"jsonrpc":"2.0","id":0,"result":6}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "multiply", "params": { "a": 4, "b": 2 }, "id": 0 }`,
			out: `{"jsonrpc":"2.0","id":0,"result":8}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "arith.pow", "params": { "base": 3, "exp": 3 }, "id": 0 }`,
			out: `{"jsonrpc":"2.0","id":0,"result":27}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "arith.pow", "params": { "base": 3 }, "id": 0 }`,
			out: `{"jsonrpc":"2.0","id":0,"result":9}`},
	}

	for _, c := range tc {
		res, err := http.Post(ts.URL, "application/json", bytes.NewBufferString(c.in))
		if err != nil {
			log.Fatal(err)
		}

		resp, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			log.Fatal(err)
		}

		if string(resp) != c.out {
			t.Errorf("Input: %s\n got %s expected %s", c.in, resp, c.out)
		}
	}
}

func TestServer_ServeHTTPWithErrors(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(rpc.ServeHTTP))
	defer ts.Close()

	var tc = []struct {
		in, out string
	}{
		{
			in:  `{"jsonrpc": "2.0", "method": "multiple1" }`,
			out: `{"jsonrpc":"2.0","id":null,"error":{"code":-32601,"message":"Method not found"}}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "test.multiple1", "id": 1 }`,
			out: `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`},
		{
			in:  `{"jsonrpc": "2.0", "method": "foobar, "params": "bar", "baz]`,
			out: `{"jsonrpc":"2.0","id":null,"error":{"code":-32700,"message":"Parse error"}}`},
		{
			in:  `{"jsonrpc": "2.0", "params": { "a": 1, "b": 0 }, "id": 1 }`,
			out: `{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"Invalid Request"}}`},
		{
			in:  `{"jsonrpc": "2.0", "method": 1, "params": "bar"}`,
			out: `{"jsonrpc":"2.0","id":null,"error":{"code":-32700,"message":"Parse error"}}`,
			// in spec: {"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null}
		},
	}

	for _, c := range tc {
		res, err := http.Post(ts.URL, "application/json", bytes.NewBufferString(c.in))
		if err != nil {
			log.Fatal(err)
		}

		resp, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			log.Fatal(err)
		}

		if string(resp) != c.out {
			t.Errorf("Input: %s\n got %s expected %s", c.in, resp, c.out)
		}
	}
}
