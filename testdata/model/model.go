package model

type Point struct {
	X, Y int // coordinate
	Z    int `json:"-"`
}
