package testdata

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sergeyfast/zenrpc"
)

func TestServer_ServeHTTP(t *testing.T) {
	var rpc = zenrpc.NewServer(zenrpc.Options{})
	rpc.Register("arith", &ArithService{})
	rpc.Register("", &ArithService{})

	ts := httptest.NewServer(rpc)
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
