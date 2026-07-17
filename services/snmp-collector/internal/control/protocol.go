// Package control implements the local Unix NDJSON status/control protocol.
package control

import "time"

const (
	// ProtocolVersion is the only supported control protocol version.
	ProtocolVersion = 1
	// MaxRequestBytes caps a single request frame.
	MaxRequestBytes = 256 * 1024
	// MaxResponseBytes caps a single response frame.
	MaxResponseBytes = 1024 * 1024
	// DefaultRequestTimeout is the per-request server timeout.
	DefaultRequestTimeout = 30 * time.Second
	// ConfirmTTL is how long prepare tokens remain valid.
	ConfirmTTL = 60 * time.Second
)

// Stable error codes for the control protocol.
const (
	CodeUnsupportedVersion = "UNSUPPORTED_VERSION"
	CodeInvalidRequest     = "INVALID_REQUEST"
	CodeMethodNotFound     = "METHOD_NOT_FOUND"
	CodeConfirmExpired     = "CONFIRM_EXPIRED"
	CodeRevisionMismatch   = "REVISION_MISMATCH"
	CodeValidationFailed   = "VALIDATION_FAILED"
	CodeConfigReloadFailed = "CONFIG_RELOAD_FAILED"
	CodeNotFound           = "NOT_FOUND"
	CodeInternal           = "INTERNAL"
)

// Request is a versioned NDJSON control request.
type Request struct {
	Version int            `json:"version"`
	ID      string         `json:"id"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params"`
}

// Response is a versioned NDJSON control response.
type Response struct {
	Version int            `json:"version"`
	ID      string         `json:"id"`
	OK      bool           `json:"ok"`
	Result  map[string]any `json:"result,omitempty"`
	Error   *ErrorBody     `json:"error,omitempty"`
}

// ErrorBody is a stable protocol error.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ProtocolError is a typed handler error with a stable code.
type ProtocolError struct {
	Code    string
	Message string
}

func (e *ProtocolError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message == "" {
		return e.Code
	}
	return e.Code + ": " + e.Message
}

func newProtoError(code, message string) *ProtocolError {
	return &ProtocolError{Code: code, Message: message}
}
