package zenrpc

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type Printer interface {
	Printf(string, ...interface{})
}

// ServeHTTP process JSON-RPC 2.0 requests via HTTP.
// http://www.simple-is-better.org/json-rpc/transport_http.html
func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// check for smd parameter and server settings and write schema if all conditions met,
	if _, ok := r.URL.Query()["smd"]; ok && s.options.ExposeSMD && r.Method == http.MethodGet {
		b, _ := json.Marshal(s.SMD())
		w.Write(b)
		return
	}

	// check for content-type and POST method.
	if !s.options.DisableTransportChecks {
		if !strings.HasPrefix(r.Header.Get("Content-Type"), contentTypeJSON) {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		} else if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		} else if r.Method != http.MethodPost {
			// skip rpc calls
			return
		}
	}

	// ok, method is POST and content-type is application/json, process body
	b, err := ioutil.ReadAll(r.Body)
	var data interface{}

	if err != nil {
		s.printf("read request body failed with err=%v", err)
		data = NewResponseError(nil, ParseError, "", nil)
	} else {
		data = s.process(newRequestContext(r.Context(), r), b)
	}

	// if responses is empty -> all requests are notifications -> exit immediately
	if data == nil {
		return
	}

	// set headers
	w.Header().Set("Content-Type", contentTypeJSON)
	if s.options.AllowCORS {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}

	// marshals data and write it to client.
	if resp, err := json.Marshal(data); err != nil {
		s.printf("marshal json response failed with err=%v", err)
		w.WriteHeader(http.StatusInternalServerError)
	} else if _, err := w.Write(resp); err != nil {
		s.printf("write response failed with err=%v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	return
}

// ServeWS processes JSON-RPC 2.0 requests via Gorilla WebSocket.
// https://github.com/gorilla/websocket/blob/master/examples/echo/
func (s Server) ServeWS(w http.ResponseWriter, r *http.Request) {
	c, err := s.options.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.printf("upgrade connection failed with err=%v", err)
		return
	}
	defer c.Close()
	
	r = r.WithContext(context.WithValue(r.Context(), "__ws_connect", c))

	for {
		mt, message, err := c.ReadMessage()

		// normal closure
		if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
			break
		}
		// abnormal closure
		if err != nil {
			s.printf("read message failed with err=%v", err)
			break
		}

		data, err := json.Marshal(s.process(newRequestContext(r.Context(), r), message))
		if err != nil {
			s.printf("marshal json response failed with err=%v", err)
			c.WriteControl(websocket.CloseInternalServerErr, nil, time.Time{})
			break
		}

		if err = c.WriteMessage(mt, data); err != nil {
			s.printf("write response failed with err=%v", err)
			c.WriteControl(websocket.CloseInternalServerErr, nil, time.Time{})
			break
		}
	}
}

// SMDBoxHandler is a handler for SMDBox web app.
func SMDBoxHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>SMD Box</title>
    <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/latest/css/bootstrap.min.css">
<link href="https://cdn.jsdelivr.net/gh/mikhail-eremin/smd-box@latest/dist/app.css" rel="stylesheet"></head>
<body>
<div id="json-rpc-root"></div>
<script type="text/javascript" src="https://cdn.jsdelivr.net/gh/mikhail-eremin/smd-box@latest/dist/app.js"></script></body>
</html>
	`))
}
