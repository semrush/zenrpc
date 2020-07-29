package zenrpc_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/semrush/zenrpc/v2"
	"github.com/semrush/zenrpc/v2/testdata"
)

var rpc = zenrpc.NewServer(zenrpc.Options{BatchMaxLen: 5, AllowCORS: true})

func init() {
	rpc.Register("arith", &testdata.ArithService{})
	rpc.Register("", &testdata.ArithService{})
	//rpc.Use(zenrpc.Logger(log.New(os.Stderr, "", log.LstdFlags)))
}

func TestServer_SMD(t *testing.T) {
	r := rpc.SMD()
	if b, err := json.Marshal(r); err != nil {
		t.Fatal(err)
	} else if !bytes.Contains(b, []byte("default")) {
		t.Error(string(b))
	}
}
