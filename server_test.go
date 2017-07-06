package zenrpc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"fmt"
	"github.com/sergeyfast/zenrpc"
	"github.com/sergeyfast/zenrpc/smd"
)

// ArithService description goes here.
type ArithService struct{ zenrpc.Service }

func (as ArithService) SMD() smd.ServiceInfo {
	return smd.ServiceInfo{
		Description: "",
		Methods: map[string]smd.Service{
			"Divide": {
				Description: "Divide divides two numbers.",
				Parameters: []smd.JSONSchema{
					{Name: "a", Optional: false, Description: "the a", Type: smd.Integer},
					{Name: "b", Optional: false, Description: "the b", Type: smd.Integer},
				},
				Returns: smd.JSONSchema{
					Type:        smd.Object,
					Description: "result is Quotient, should be named var", // or Quotient docs if return desc
					Optional:    true,
					Properties: map[string]smd.Property{
						"Quo": {Type: smd.Integer, Description: "Quo docs"},
						"rem": {Type: smd.Integer, Description: "rem docs"},
					},
				},
			},
			"Sum": {
				Description: "Sum sums two digits and returns error with error code as result and IP from context.",
				Parameters: []smd.JSONSchema{
					{Name: "a", Optional: false, Type: smd.Integer},
					{Name: "b", Optional: false, Type: smd.Integer},
				},
				Returns: smd.JSONSchema{
					Type: smd.Boolean,
				},
			},
			"Multiply": {
				Description: "Multiply multiples two digits and returns result.",
				Parameters: []smd.JSONSchema{
					{Name: "a", Optional: false, Type: smd.Integer},
					{Name: "b", Optional: false, Type: smd.Integer},
				},
				Returns: smd.JSONSchema{
					Type: smd.Integer,
				},
			},
			"Pow": {
				Description: "Pow returns x**y, the base-x exponential of y. If Exp is not set then default value is 2.",
				Parameters: []smd.JSONSchema{
					{Name: "base", Optional: false, Type: smd.Float},
					{Name: "exp", Optional: true, Type: smd.Float, Default: smd.RawMessageString("2"), Description: "exponent could be empty"},
				},
				Returns: smd.JSONSchema{
					Type: smd.Float,
				},
			},
		},
	}
}

// Invoke is as generated code from zenrpc cmd
func (as ArithService) Invoke(ctx context.Context, method string, params json.RawMessage) zenrpc.Response {
	resp := zenrpc.Response{}

	switch method {
	case "print":

		// generate it
		var args = struct {
			Str string `json:"str"`
			Int int    `json:"int"`
			Obj struct {
				Str   string  `json:"str"`
				Float float32 `json:"float"`
				Array []int   `json:"array"`
			} `json:"obj"`
			Array []string       `json:"array"`
			Map   map[string]int `json:"map"`
		}{}

		if zenrpc.IsArray(params) {
			// generate it
			keys := []string{"str", "int", "obj", "array", "map"}

			// TODO refactor - think how to get rid of else
			if conv, err := zenrpc.ConvertToObject(keys, params); err != nil {
				return zenrpc.NewResponseError(nil, zenrpc.InvalidParams, err.Error(), nil)
			} else {
				params = conv
			}
		}

		if err := json.Unmarshal(params, &args); err != nil {
			return zenrpc.NewResponseError(nil, zenrpc.InvalidParams, err.Error(), nil)
		}

		// todo set default values
		resp.Set(fmt.Sprintf("%v", args))
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
	r, _ := zenrpc.RequestFromContext(ctx)

	return true, zenrpc.NewStringError(a+b, r.Host)
}

// Multiply multiples two digits and returns result.
func (as *ArithService) Multiply(a, b int) int {
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

var rpc = zenrpc.NewServer(zenrpc.Options{BatchMaxLen: 5})

func init() {
	rpc.Register("arith", &ArithService{})
	rpc.Register("", &ArithService{})
	//rpc.Use(zenrpc.Logger(log.New(os.Stderr, "", log.LstdFlags)))
}

func TestServer_ServeHTTPArray(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(rpc.ServeHTTP))
	defer ts.Close()

	out := `{"jsonrpc":"2.0","id":1,"result":"{test 1 {test nested 1.5 [1 2 3]} [el1 el2] map[key1:1]}"}`
	var tc = []struct {
		in, out string
	}{
		{
			in: `{"jsonrpc": "2.0", "method": "arith.print",
				   "params": [
				     "test",
				     1,
				     {"str": "test nested", "float": 1.5, "array": [1,2,3]},
				     ["el1", "el2"],
				     {"key1": 1}
				   ],
				   "id": 1 }`,
			out: out},
		{
			in: `{"jsonrpc": "2.0", "method": "arith.print",
				   "params": {
				     "str": "test",
				     "int": 1,
				     "obj": {"str": "test nested", "float": 1.5, "array": [1,2,3]},
				     "array": ["el1", "el2"],
				     "map": {"key1": 1}
				   },
				   "id": 1 }`,
			out: out},
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

func TestServer_ServeHTTP(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(rpc.ServeHTTP))
	defer ts.Close()

	var tc = []struct {
		in, out string
	}{
		{
			in:  `{"jsonrpc": "2.0", "method": "arith.divide", "params": { "a": 1, "b": 24 }, "id": 1 }`,
			out: `{"jsonrpc":"2.0","id":1,"result":{"Quo":0,"rem":1}}`},
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
			in:  `{"jsonrpc": "2.0", "method": "arith.sum", "params": { "a": 3, "b": 3 }, "id": 1 }`,
			out: `{"jsonrpc":"2.0","id":1,"error":{"code":6,"message":"` + ts.Listener.Addr().String() + `"}}`},
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

func TestServer_ServeHTTPNotifications(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(rpc.ServeHTTP))
	defer ts.Close()

	var tc = []struct {
		in, out string
	}{
		{
			in:  `{"jsonrpc": "2.0", "method": "arith.divide", "params": { "a": 1, "b": 24 }}`,
			out: ``},
		{
			// should be empty even with error
			in:  `{"jsonrpc": "2.0", "method": "arith.divide", "params": { "a": 1, "b": 0 }}`,
			out: ``},
		{
			// but parse errors should be displayed
			in:  `{"jsonrpc": "1.0", "method": "Arith.Divide", "params": { "a": 1, "b": 1 }`,
			out: `{"jsonrpc":"2.0","id":null,"error":{"code":-32700,"message":"Parse error"}}`},
		{
			// in batch requests notifications should not be listed in response
			in: `[{"jsonrpc": "2.0", "method": "arith.multiply", "params": { "a": 3, "b": 2 }, "id": 0 },
				   {"jsonrpc": "2.0", "method": "arith.pow", "params": { "base": 2, "exp": 2 } }]`,
			out: `[{"jsonrpc":"2.0","id":0,"result":6}]`},
		{
			// order doesn't matter
			in: `[{"jsonrpc": "2.0", "method": "arith.multiply", "params": { "a": 3, "b": 2 } },
				   {"jsonrpc": "2.0", "method": "arith.pow", "params": { "base": 2, "exp": 2 }, "id": 0 }]`,
			out: `[{"jsonrpc":"2.0","id":0,"result":4}]`},
		{
			// all notifications
			in: `[{"jsonrpc": "2.0", "method": "arith.multiply", "params": { "a": 3, "b": 2 } },
				   {"jsonrpc": "2.0", "method": "arith.pow", "params": { "base": 2, "exp": 2 }}]`,
			out: ``},
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

func TestServer_ServeHTTPBatch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(rpc.ServeHTTP))
	defer ts.Close()

	var tc = []struct {
		in  string
		out []string
	}{
		{
			// batch requests should process asynchronously, any order in responses accepted
			in: `[{"jsonrpc": "2.0", "method": "arith.multiply", "params": { "a": 3, "b": 2 }, "id": 0 },
				  {"jsonrpc": "2.0", "method": "arith.multiply", "params": { "a": 3, "b": 3 }, "id": 1 },
				  {"jsonrpc": "2.0", "method": "arith.pow", "params": { "a": 2, "b": 3 } },
				  {"jsonrpc": "2.0", "method": "arith.pow", "params": { "base": 2, "exp": 2 }, "id": 2 }]`,
			out: []string{
				`{"jsonrpc":"2.0","id":1,"result":9}`,
				`{"jsonrpc":"2.0","id":0,"result":6}`,
				`{"jsonrpc":"2.0","id":2,"result":4}`}},
		{
			// one of the requests errored
			in: `[{"jsonrpc": "2.0", "method": "arith.multiply1", "params": { "a": 3, "b": 2 }, "id": 0 },
				  {"jsonrpc": "2.0", "method": "arith.multiply", "params": { "a": 3, "b": 3 }, "id": 1 },
				  {"jsonrpc": "2.0", "method": "arith.pow", "params": { "a": 2, "b": 3 } },
				  {"jsonrpc": "2.0", "method": "arith.pow", "params": { "base": 2, "exp": 2 }, "id": 2 }]`,
			out: []string{
				`{"jsonrpc":"2.0","id":1,"result":9}`,
				`{"jsonrpc":"2.0","id":0,"error":{"code":-32601,"message":"Method not found"}}`,
				`{"jsonrpc":"2.0","id":2,"result":4}`}},
		{
			// to much batch requests
			in: `[{"jsonrpc": "2.0", "method": "arith.multiply1", "params": { "a": 3, "b": 2 }, "id": 0 },
				  {"jsonrpc": "2.0", "method": "arith.multiply", "params": { "a": 3, "b": 3 }, "id": 1 },
				  {"jsonrpc": "2.0", "method": "arith.pow", "params": { "a": 2, "b": 3 } },
				  {"jsonrpc": "2.0", "method": "arith.pow", "params": { "a": 2, "b": 3 } },
				  {"jsonrpc": "2.0", "method": "arith.pow", "params": { "a": 2, "b": 3 } },
				  {"jsonrpc": "2.0", "method": "arith.pow", "params": { "base": 2, "exp": 2 }, "id": 2 }]`,
			out: []string{
				`{"jsonrpc":"2.0","id":null,"error":{"code":-32600,"message":"Invalid Request","data":"max requests length in batch exceeded"}}`}},
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

		// checking if count of responses is correct
		if cnt := strings.Count(string(resp), `"jsonrpc":"2.0"`); len(c.out) != cnt {
			t.Errorf("Input: %s\n got %d in batch expected %d", c.in, cnt, len(c.out))
		}

		// checking every response variant to be in response
		for _, check := range c.out {
			if !strings.Contains(string(resp), check) {
				t.Errorf("Input: %s\n not found %s in batch %s", c.in, check, resp)
			}
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
			in:  `{"jsonrpc": "2.0", "method": "multiple1", "id": 1 }`,
			out: `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`},
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

func TestServer_SMD(t *testing.T) {
	r := rpc.SMD()
	if b, err := json.Marshal(r); err != nil {
		t.Fatal(err)
	} else if !bytes.Contains(b, []byte("Quo")) {
		t.Error(string(b))
	}
}
