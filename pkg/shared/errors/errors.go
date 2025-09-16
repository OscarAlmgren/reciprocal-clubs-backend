package errors

import (
	"errors"
	"fmt"
)

// ErrorCode represents a type of application error
 type ErrorCode string

const (
	ErrNotFound        ErrorCode = "NOT_FOUND"
	ErrInvalidInput    ErrorCode = "INVALID_INPUT"
	ErrUnauthorized    ErrorCode = "UNAUTHORIZED"
	ErrForbidden       ErrorCode = "FORBIDDEN"
	ErrConflict        ErrorCode = "CONFLICT"
	ErrInternal        ErrorCode = "INTERNAL"
	ErrUnavailable     ErrorCode = "UNAVAILABLE"
	ErrTimeout         ErrorCode = "TIMEOUT"
)

// AppError is a structured application error
 type AppError struct {
	Code    ErrorCode
	Message string
	Err     error
	Fields  map[string]interface{}
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.Err }

// New creates a new application error
 func New(code ErrorCode, msg string, fields map[string]interface{}, err error) *AppError {
	return &AppError{Code: code, Message: msg, Err: err, Fields: fields}
}

// Helpers
 func NotFound(msg string, fields map[string]interface{}) *AppError {
	return New(ErrNotFound, msg, fields, nil)
}

func InvalidInput(msg string, fields map[string]interface{}, err error) *AppError {
	return New(ErrInvalidInput, msg, fields, err)
}

func Unauthorized(msg string, fields map[string]interface{}) *AppError {
	return New(ErrUnauthorized, msg, fields, nil)
}

func Forbidden(msg string, fields map[string]interface{}) *AppError {
	return New(ErrForbidden, msg, fields, nil)
}

func Conflict(msg string, fields map[string]interface{}) *AppError {
	return New(ErrConflict, msg, fields, nil)
}

func Internal(msg string, fields map[string]interface{}, err error) *AppError {
	return New(ErrInternal, msg, fields, err)
}

func Unavailable(msg string, fields map[string]interface{}, err error) *AppError {
	return New(ErrUnavailable, msg, fields, err)
}

func Timeout(msg string, fields map[string]interface{}, err error) *AppError {
	return New(ErrTimeout, msg, fields, err)
}

// Is checks if target error matches provided code
 func Is(err error, code ErrorCode) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}

// Wrap adds context to an error with fields and code, preserving the original error
 func Wrap(err error, code ErrorCode, msg string, fields map[string]interface{}) *AppError {
	if err == nil {
		return nil
	}
	return &AppError{Code: code, Message: msg, Err: err, Fields: fields}
}
