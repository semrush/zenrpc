package zenrpc

import (
	"encoding/json"
)

const (
	// Invalid JSON was received by the server.
	// An error occurred on the server while parsing the JSON text.
	ParseError = -32700

	// The JSON sent is not a valid Request object.
	InvalidRequest = -32600

	// The method does not exist / is not available.
	MethodNotFound = -32601

	// Invalid method parameter(s).
	InvalidParams = -32602

	// Internal JSON-RPC error.
	InternalError = -32603

	// Reserved for implementation-defined server-errors.
	ServerError = -32000
)

var errorMessages = map[int]string{
	ParseError:     "Parse error",
	InvalidRequest: "Invalid Request",
	MethodNotFound: "Method not found",
	InvalidParams:  "Invalid params",
	InternalError:  "Internal error",
	ServerError:    "Server error",
}

// Json structure for json-rpc request to server. See:
// http://www.jsonrpc.org/specification#request_object
//easyjson:json
type Request struct {
	// A String specifying the version of the JSON-RPC protocol. MUST be exactly "2.0".
	Version string `json:"jsonrpc"`

	// An identifier established by the Client that MUST contain a String, Number, or NULL value if included.
	// If it is not included it is assumed to be a notification.
	// The value SHOULD normally not be Null [1] and Numbers SHOULD NOT contain fractional parts
	Id json.RawMessage `json:"id"`

	// A String containing the name of the method to be invoked.
	// Method names that begin with the word rpc followed by a period character (U+002E or ASCII 46)
	// are reserved for rpc-internal methods and extensions and MUST NOT be used for anything else.
	Method string `json:"method"`

	// A Structured value that holds the parameter values to be used during the invocation of the method.
	// This member MAY be omitted.
	Params json.RawMessage `json:"params"`

	// Namespace holds
	Namespace string `json:"-"`
}

// Json structure for json-rpc response from server. See:
// http://www.jsonrpc.org/specification#response_object
//easyjson:json
type Response struct {
	// A String specifying the version of the JSON-RPC protocol. MUST be exactly "2.0".
	Version string `json:"jsonrpc"`

	// This member is REQUIRED.
	// It MUST be the same as the value of the id member in the Request Object.
	// If there was an error in detecting the id in the Request object (e.g. Parse error/Invalid Request), it MUST be Null.
	Id json.RawMessage `json:"id"`

	// This member is REQUIRED on success.
	// This member MUST NOT exist if there was an error invoking the method.
	// The value of this member is determined by the method invoked on the Server.
	Result json.RawMessage `json:"result,omitempty"`

	// This member is REQUIRED on error.
	// This member MUST NOT exist if there was no error triggered during invocation.
	// The value for this member MUST be an Object as defined in section 5.1.
	Error *Error `json:"error,omitempty"`
}

func (r Response) JSON() []byte {
	// TODO handle error
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

	// A String providing a short description of the error.
	// The message SHOULD be limited to a concise single sentence.
	Message string `json:"message"`

	// A Primitive or Structured value that contains additional information about the error.
	// This may be omitted.
	// The value of this member is defined by the Server (e.g. detailed error information, nested errors etc.).
	Data interface{} `json:"data,omitempty"`

	Err error
}

func (Error) Error() string {
	panic("me	")
}

func NewResponseError(id json.RawMessage, code int, message string, data interface{}) Response {
	return Response{
		Id: id,
		Error: &Error{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}
