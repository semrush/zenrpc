package zenrpc_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/s3rj1k/zenrpc"
	"github.com/s3rj1k/zenrpc/testdata"
)

// ArithService description goes here.
type ArithService struct{ zenrpc.Service }

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
