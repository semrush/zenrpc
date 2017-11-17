package zenrpc

import (
	"encoding/json"
)

const (
	// ParseError is error code defined by JSON-RPC 2.0 spec.
	// Invalid JSON was received by the server.
	// An error occurred on the server while parsing the JSON text.
	ParseError = -32700

	// InvalidRequest is error code defined by JSON-RPC 2.0 spec.
	// The JSON sent is not as valid Request object.
	InvalidRequest = -32600

	// MethodNotFound is error code defined by JSON-RPC 2.0 spec.
	// The method does not exist / is not available.
	MethodNotFound = -32601

	// InvalidParams is error code defined by JSON-RPC 2.0 spec.
	// Invalid method parameter(s).
	InvalidParams = -32602

	// InternalError is error code defined by JSON-RPC 2.0 spec.
	// Internal JSON-RPC error.
	InternalError = -32603

	// ServerError is error code defined by JSON-RPC 2.0 spec.
	// Reserved for implementation-defined server-errors.
	ServerError = -32000

	// Version is only supported JSON-RPC Version.
	Version = "2.0"
)

var errorMessages = map[int]string{
	ParseError:     "Parse error",
	InvalidRequest: "Invalid Request",
	MethodNotFound: "Method not found",
	InvalidParams:  "Invalid params",
	InternalError:  "Internal error",
	ServerError:    "Server error",
}

// ErrorMsg returns error as text for default JSON-RPC errors.
func ErrorMsg(code int) string {
	return errorMessages[code]
}

// Request is a json structure for json-rpc request to server. See:
// http://www.jsonrpc.org/specification#request_object
//easyjson:json
type Request struct {
	// A String specifying the version of the JSON-RPC protocol. MUST be exactly "2.0".
	Version string `json:"jsonrpc"`

	// An identifier established by the Client that MUST contain as String, Number, or NULL value if included.
	// If it is not included it is assumed to be as notification.
	// The value SHOULD normally not be Null [1] and Numbers SHOULD NOT contain fractional parts.
	ID *json.RawMessage `json:"id"`

	// A String containing the name of the method to be invoked.
	// Method names that begin with the word rpc followed by as period character (U+002E or ASCII 46)
	// are reserved for rpc-internal methods and extensions and MUST NOT be used for anything else.
	Method string `json:"method"`

	// A Structured value that holds the parameter values to be used during the invocation of the method.
	// This member MAY be omitted.
	Params json.RawMessage `json:"params"`

	// Namespace holds namespace. Not in spec, for internal needs.
	Namespace string `json:"-"`
}

// Response is json structure for json-rpc response from server. See:
// http://www.jsonrpc.org/specification#response_object
//easyjson:json
type Response struct {
	// A String specifying the version of the JSON-RPC protocol. MUST be exactly "2.0".
	Version string `json:"jsonrpc"`

	// This member is REQUIRED.
	// It MUST be the same as the value of the id member in the Request Object.
	// If there was an error in detecting the id in the Request object (e.g. Parse error/Invalid Request), it MUST be Null.
	ID *json.RawMessage `json:"id"`

	// This member is REQUIRED on success.
	// This member MUST NOT exist if there was an error invoking the method.
	// The value of this member is determined by the method invoked on the Server.
	Result *json.RawMessage `json:"result,omitempty"`

	// This member is REQUIRED on error.
	// This member MUST NOT exist if there was no error triggered during invocation.
	// The value for this member MUST be an Object as defined in section 5.1.
	Error *Error `json:"error,omitempty"`
}

// JSON is temporary method that silences error during json marshalling.
func (r Response) JSON() []byte {
	// TODO process error
	b, _ := json.Marshal(r)
	return b
}

// Error object used in response if function call errored. See:
// http://www.jsonrpc.org/specification#error_object
//easyjson:json
type Error struct {
	// A Number that indicates the error type that occurred.
	// This MUST be an integer.
	Code int `json:"code"`

	// A String providing as short description of the error.
	// The message SHOULD be limited to as concise single sentence.
	Message string `json:"message"`

	// A Primitive or Structured value that contains additional information about the error.
	// This may be omitted.
	// The value of this member is defined by the Server (e.g. detailed error information, nested errors etc.).
	Data interface{} `json:"data,omitempty"`

	Err error `json:"-"`
}

// NewStringError makes a JSON-RPC with given code and message.
func NewStringError(code int, message string) *Error {
	return &Error{Code: code, Message: message}
}

// NewError makes a JSON-RPC error with given code and standard error.
func NewError(code int, err error) *Error {
	e := &Error{Code: code, Err: err}
	e.Message = e.Error()
	return e
}

// Error returns first filled value from Err, Message or default text for JSON-RPC error.
func (e Error) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}

	if e.Message != "" {
		return e.Message
	}

	return ErrorMsg(e.Code)
}

// NewResponseError returns new Response with Error object.
func NewResponseError(id *json.RawMessage, code int, message string, data interface{}) Response {
	if message == "" {
		message = ErrorMsg(code)
	}

	return Response{
		Version: Version,
		ID:      id,
		Error: &Error{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

// Set sets result and error if needed.
func (r *Response) Set(v interface{}, er ...error) {
	r.Version = Version
	var err error

	if e, ok := v.(error); ok && e != nil {
		er = []error{e}
		v = nil
	}
	// check for nil *zenrpc.Error
	// TODO(sergeyfast): add ability to return other error types
	if len(er) > 0 && er[0] != nil {
		err = er[0]
		if e, ok := err.(*Error); ok && e == nil {
			err = nil
		}
	}

	// set first error if occurred
	if err != nil {
		if e, ok := err.(*Error); ok {
			r.Error = e
		} else {
			r.Error = NewError(InternalError, err)
		}

		return
	}

	// set result or error on marshal
	if res, err := json.Marshal(v); err != nil {
		r.Error = NewError(ServerError, err)
	} else {
		rm := json.RawMessage(res)
		r.Result = &rm
	}
}
