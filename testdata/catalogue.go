package testdata

import (
	"github.com/semrush/zenrpc"
)

type Group struct {
	Id    int    `json:"id"`
	Title string `json:"title"`
}

type Campaign struct {
	Id     int     `json:"id"`
	Groups []Group `json:"group"`
}

type CatalogueService struct{ zenrpc.Service }

func (s CatalogueService) First(groups []Group) (bool, error) {
	return true, nil
}

func (s CatalogueService) Second(campaigns []Campaign) (bool, error) {
	return true, nil
}

//go:generate zenrpc
