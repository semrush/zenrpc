package zenrpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStorage(t *testing.T) {
	c := newContext(nil, nil)
	c.Set("name", "John Doe")
	assert.Equal(t, "John Doe", c.Get("name"))
}
