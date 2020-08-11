package model

import "github.com/semrush/zenrpc/v2/testdata/objects"

type Point struct {
	objects.AbstractObject
	X, Y int // coordinate
	Z    int `json:"-"`
}
