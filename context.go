package zenrpc

import (
	"net"
	"net/http"
	"strings"
	"sync"
)

type Context interface {
	// Request returns `*http.Request`.
	Request() *http.Request

	// Response returns `*http.Response`.
	Response() *http.Response

	// RealIP returns the client's network address based on `X-Forwarded-For`
	// or `X-Real-IP` request header.
	// The behavior can be configured using `Echo#IPExtractor`.
	RealIP() string

	// Cookie returns the named cookie provided in the request.
	Cookie(name string) (*http.Cookie, error)

	// SetCookie adds a `Set-Cookie` header in HTTP response.
	SetCookie(cookie *http.Cookie)

	// Cookies returns the HTTP cookies sent with the request.
	Cookies() []*http.Cookie

	// Get retrieves data from the context.
	Get(key string) interface{}

	// Set saves data in the context.
	Set(key string, value interface{})
}

type basicContext struct {
	request  *http.Request
	response *http.Response
	lock     sync.RWMutex
	store    map[string]interface{}
}

func (c *basicContext) Request() *http.Request {
	return c.request
}

func (c *basicContext) Response() *http.Response {
	return c.response
}

func (c *basicContext) RealIP() string {
	if ip := c.request.Header.Get("X-Forwarded-For"); ip != "" {
		i := strings.IndexAny(ip, ", ")
		if i > 0 {
			return ip[:i]
		}
		return ip
	}
	if ip := c.request.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	ra, _, _ := net.SplitHostPort(c.request.RemoteAddr)
	return ra
}

func (c *basicContext) Cookie(name string) (*http.Cookie, error) {
	return c.request.Cookie(name)
}

func (c *basicContext) SetCookie(cookie *http.Cookie) {
	panic("implement me")
}

func (c *basicContext) Cookies() []*http.Cookie {
	panic("implement me")
}

func (c *basicContext) Get(key string) interface{} {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.store[key]
}

func (c *basicContext) Set(key string, value interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.store[key] = value
}

func newContext(request *http.Request, response *http.Response) *basicContext {
	return &basicContext{
		request:  request,
		response: response,
		store:    make(map[string]interface{}),
	}
}
