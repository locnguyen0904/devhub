// Package httpx holds the error contract shared by every module: services
// return domain errors, and this package is the single place that turns them
// into HTTP responses.
package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// Error codes returned to clients. The frontend switches on these; it must
// never parse Message. Documented in docs/03-api.md §1.
const (
	CodeInvalidRequest     = "invalid_request"
	CodeMethodNotAllowed   = "method_not_allowed"
	CodeUnauthenticated    = "unauthenticated"
	CodeForbidden          = "forbidden"
	CodeNotFound           = "not_found"
	CodeConflict           = "conflict"
	CodeValidationFailed   = "validation_failed"
	CodeRateLimited        = "rate_limited"
	CodeServiceUnavailable = "service_unavailable"
	CodeInternal           = "internal_error"
)

// statusForCode maps a domain code to its HTTP status. An unknown code is a
// programming mistake rather than a client error, so it becomes a 500.
func statusForCode(code string) int {
	switch code {
	case CodeInvalidRequest:
		return http.StatusBadRequest
	case CodeMethodNotAllowed:
		return http.StatusMethodNotAllowed
	case CodeUnauthenticated:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict:
		return http.StatusConflict
	case CodeValidationFailed:
		return http.StatusUnprocessableEntity
	case CodeRateLimited:
		return http.StatusTooManyRequests
	case CodeServiceUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// Error is the domain error every service returns. It carries no HTTP types,
// so service.go and repository.go never import huma.
type Error struct {
	Code    string
	Message string            // safe to show a user
	Fields  map[string]string // per-field messages when Code is validation_failed
	Err     error             // wrapped cause, logged but never sent to the client
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.Err }

// New builds a domain error. Prefer the named helpers below where one fits.
func New(code, message string, cause error) *Error {
	return &Error{Code: code, Message: message, Err: cause}
}

// NotFound reports a resource that does not exist, or that the caller is not
// allowed to know exists.
func NotFound(message string) *Error {
	return &Error{Code: CodeNotFound, Message: message}
}

// Forbidden reports an authenticated caller acting outside their permissions.
func Forbidden(message string) *Error {
	return &Error{Code: CodeForbidden, Message: message}
}

// Conflict reports a request that clashes with existing state.
func Conflict(message string) *Error {
	return &Error{Code: CodeConflict, Message: message}
}

// Invalid reports per-field validation failures.
func Invalid(message string, fields map[string]string) *Error {
	return &Error{Code: CodeValidationFailed, Message: message, Fields: fields}
}

// Unavailable reports a dependency the API needs but cannot reach.
func Unavailable(message string, cause error) *Error {
	return &Error{Code: CodeServiceUnavailable, Message: message, Err: cause}
}

// ToHuma converts a domain error into the response huma sends. Handlers call
// this and nothing else; it is the only bridge between the two vocabularies.
//
// An error that is not a *Error is a bug rather than an expected outcome, so it
// becomes a generic 500 and the cause stays in the logs.
func ToHuma(err error) error {
	var domain *Error
	if !errors.As(err, &domain) {
		return NewStatusError(http.StatusInternalServerError, CodeInternal,
			"Something went wrong, please try again", nil)
	}

	return NewStatusError(statusForCode(domain.Code), domain.Code, domain.Message, domain.Fields)
}

// ErrorModel is the body of every failed response. The wrapping "error" object
// keeps failures distinguishable from successful payloads by shape alone.
type ErrorModel struct {
	status int
	Err    ErrorBody `json:"error"`
}

// ErrorBody carries the machine-readable code plus a message safe to display.
// The request id travels in the X-Request-ID response header, not here, because
// huma builds error bodies without access to the request context.
type ErrorBody struct {
	Code    string            `json:"code"    doc:"Stable error code for clients to switch on" example:"not_found"`
	Message string            `json:"message" doc:"Human-readable message safe to display"`
	Fields  map[string]string `json:"fields,omitempty" doc:"Per-field errors when code is validation_failed"`
}

func (e *ErrorModel) Error() string { return e.Err.Message }

// GetStatus satisfies huma.StatusError.
func (e *ErrorModel) GetStatus() int { return e.status }

// NewStatusError builds the huma-facing error value.
func NewStatusError(status int, code, message string, fields map[string]string) *ErrorModel {
	return &ErrorModel{status: status, Err: ErrorBody{Code: code, Message: message, Fields: fields}}
}

// WriteError writes an error body directly. Router-level failures such as an
// unmatched path never reach a huma operation, so without this they would fall
// back to chi's plain-text default and clients would see two response shapes.
func WriteError(w http.ResponseWriter, code, message string) {
	status := statusForCode(code)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	// The header is already written; a failed encode can only be logged by the
	// caller's middleware, so there is nothing useful to do with the error here.
	_ = json.NewEncoder(w).Encode(NewStatusError(status, code, message, nil))
}

// UseErrorModel replaces huma's default error body with ours, so validation
// failures huma raises on its own come back in the same shape as ours.
//
// Assigning a package-level var is huma's documented extension point. Call this
// once from main before registering any operation.
func UseErrorModel() {
	huma.NewError = func(status int, msg string, errs ...error) huma.StatusError {
		code := codeForStatus(status)

		var fields map[string]string
		for _, e := range errs {
			var detail *huma.ErrorDetail
			if errors.As(e, &detail) {
				if fields == nil {
					fields = make(map[string]string, len(errs))
				}
				fields[detail.Location] = detail.Message
			}
		}
		return NewStatusError(status, code, msg, fields)
	}
}

// codeForStatus is the inverse of statusForCode, written as its own switch so
// two codes sharing a status can never make the reverse direction ambiguous.
func codeForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return CodeInvalidRequest
	case http.StatusMethodNotAllowed:
		return CodeMethodNotAllowed
	case http.StatusUnauthorized:
		return CodeUnauthenticated
	case http.StatusForbidden:
		return CodeForbidden
	case http.StatusNotFound:
		return CodeNotFound
	case http.StatusConflict:
		return CodeConflict
	case http.StatusUnprocessableEntity:
		return CodeValidationFailed
	case http.StatusTooManyRequests:
		return CodeRateLimited
	case http.StatusServiceUnavailable:
		return CodeServiceUnavailable
	default:
		return CodeInternal
	}
}
