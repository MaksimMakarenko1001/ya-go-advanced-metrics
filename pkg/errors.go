package pkg

import (
	"fmt"
	"net/http"
)

// ErrInternalServer represents a generic internal server error.
var ErrInternalServer = &Error{
	Message: "Internal error",
	Code:    "INTERNAL_SERVER_ERROR",
	Status:  http.StatusInternalServerError,
}

// ErrNotFound represents a not found error.
var ErrNotFound = &Error{
	Message: "Not found",
	Code:    "NOT_FOUND",
	Status:  http.StatusNotFound,
}

// ErrBadRequest represents a bad request error.
var ErrBadRequest = &Error{
	Message: "Bad request",
	Code:    "BAD_REQUEST",
	Status:  http.StatusBadRequest,
}

// ErrNotFound represents a forbidden error.
var ErrForbidden = &Error{
	Message: "Forbidden",
	Code:    "FORBIDDEN",
	Status:  http.StatusForbidden,
}

// allowStatusError defines allowed HTTP status codes for errors.
var allowStatusError = map[int]struct{}{
	http.StatusInternalServerError: {},
	http.StatusNotFound:            {},
	http.StatusBadRequest:          {},
	http.StatusForbidden:           {},
}

// ErrorCode represents a unique error code identifier.
type ErrorCode string

// Error represents an application error with HTTP status code support.
type Error struct {
	Message string
	Code    ErrorCode
	Status  int
	Info    string
}

// Error returns the string representation of the error.
func (e *Error) Error() string {
	if e == nil || e.Code == "" {
		return ""
	}

	if e.Info == "" {
		return fmt.Sprintf("[%s] %s", e.Code, e.Message)
	}

	return fmt.Sprintf("[%s] %s (%s)", e.Code, e.Message, e.Info)

}

// HTTPStatus returns the HTTP status code for the error.
func (e *Error) HTTPStatus() int {
	if e == nil {
		return http.StatusOK
	}
	if _, ok := allowStatusError[e.Status]; !ok {
		return http.StatusInternalServerError
	}
	return e.Status
}

// SetInfo creates a new error with additional information.
func (e *Error) SetInfo(s string) *Error {
	return &Error{
		Message: e.Message,
		Code:    e.Code,
		Status:  e.Status,
		Info:    s,
	}
}

// SetInfof creates a new error with formatted additional information.
func (e *Error) SetInfof(s string, v ...any) *Error {
	return e.SetInfo(fmt.Sprintf(s, v...))
}
