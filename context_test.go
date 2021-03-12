package zenrpc

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStorage(t *testing.T) {
	c := newContext(nil, nil)
	c.Set("name", "John Doe")
	assert.Equal(t, "John Doe", c.Get("name"))
}

func TestRequest(t *testing.T) {
	r := &http.Request{
		Host: "example.com",
	}
	c := newContext(r, nil)
	assert.Equal(t, r, c.Request())
}

func TestResponse(t *testing.T) {
	r := &http.Response{
		StatusCode: http.StatusOK,
	}
	c := newContext(nil, r)
	assert.Equal(t, r, c.Response())
}
