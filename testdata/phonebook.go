package testdata

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/sergeyfast/zenrpc"
	"sync"
)

// SerialPeopleAccess is middleware for seiral access to PhoneBook methods
func SerialPeopleAccess(pbNamespace string) zenrpc.MiddlewareFunc {
	var lock sync.RWMutex
	return func(h zenrpc.InvokeFunc) zenrpc.InvokeFunc {
		return func(ctx context.Context, method string, params json.RawMessage) zenrpc.Response {
			if zenrpc.NamespaceFromContext(ctx) == pbNamespace {
				switch method {
				case RPC.PhoneBook.Get, RPC.PhoneBook.ById:
					lock.RLock()
					defer lock.RUnlock()
				case RPC.PhoneBook.Delete, RPC.PhoneBook.Save, RPC.PhoneBook.Remove:
					lock.Lock()
					defer lock.Unlock()
				}
			}

			return h(ctx, method, params)
		}
	}
}

// People is sample db.
var People = map[uint64]*Person{
	1: {
		ID:        1,
		FirstName: "John",
		LastName:  "Doe",
		Phone:     "+1-800-142-31-22",
		Mobile:    []string{"m1", "m2", "m3"},
		Deleted:   false,
		Addresses: []Address{
			{Street: "Main street", City: "Main City"},
			{Street: "Default street", City: "Default City"},
		},
	},
	2: {
		ID:         2,
		FirstName:  "Ivan",
		LastName:   "Ivanov",
		Phone:      "+7-900-131-53-94",
		Deleted:    true,
		AltAddress: &Address{Street: "Main street", City: "Main City"},
	},
}

// Person in base model for phone book
type Person struct {
	// ID is Unique Identifier for person
	ID                  uint64
	FirstName, LastName string
	// Phone is main phone
	Phone     string
	WorkPhone *string
	Mobile    []string

	// Deleted is flag for
	Deleted bool

	// Addresses Could be nil or len() == 0.
	Addresses  []Address
	AltAddress *Address `json:"address"`
}

type PersonSearch struct {
	// ByName is filter for searching person by first name or last name.
	ByName    *string
	ByType    *string
	ByPhone   string
	ByAddress *Address
}

type Address struct {
	Street string
	City   string
}

type PhoneBook struct {
	DB map[uint64]*Person
	id uint64
} //zenrpc

// Get returns all people from DB.
//zenrpc:page:0 current page
//zenrpc:count:50 page size
func (pb PhoneBook) Get(search PersonSearch, page, count *int) (res []*Person) {
	for _, p := range pb.DB {
		res = append(res, p)
	}

	return
}

// ValidateSearch returns given search as result.
//zenrpc:search search object
func (pb PhoneBook) ValidateSearch(search *PersonSearch) *PersonSearch {
	return search
}

// ById returns Person from DB.
//zenrpc:id person id
//zenrpc:404 person was not found
func (pb PhoneBook) ById(id uint64) (*Person, *zenrpc.Error) {
	if p, ok := pb.DB[id]; ok {
		return p, nil
	}

	return nil, zenrpc.NewStringError(404, "person was not found")
}

// Delete marks person as deleted.
//zenrpc:id person id
//zenrpc:success operation result
func (pb PhoneBook) Delete(id uint64) (success bool, error error) {
	if p, ok := pb.DB[id]; ok {
		p.Deleted = true
		return true, nil
	}

	return false, errors.New("person was not found")
}

// Removes deletes person from DB.
//zenrpc:id person id
//zenrpc:success operation result
func (pb PhoneBook) Remove(id uint64) (success bool, error error) {
	if _, ok := pb.DB[id]; ok {
		delete(pb.DB, id)
		return true, nil
	}

	return false, errors.New("person was not found")
}

// Save saves person to DB.
//zenrpc:replace:false update person if exist
//zenrpc:400 	invalid request
//zenrpc:401 	use replace=true
func (pb *PhoneBook) Save(p Person, replace *bool) (id uint64, err *zenrpc.Error) {
	// validate
	if p.FirstName == "" || p.LastName == "" {
		return 0, zenrpc.NewStringError(400, "first name or last name is empty")
	}

	_, ok := pb.DB[p.ID]
	if ok && *replace == false {
		return 0, zenrpc.NewStringError(401, "")
	}

	pb.id++
	p.ID = pb.id
	pb.DB[p.ID] = &p

	return pb.id, nil
}
