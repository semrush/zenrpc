package subarithservice

import (
	"github.com/semrush/zenrpc/v2/testdata/objects"
)

type Point struct {
	objects.AbstractObject
	A, B int // coordinate
	C    int `json:"-"`
}
