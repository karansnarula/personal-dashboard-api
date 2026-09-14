package domain

import (
	"errors"
	"fmt"
)

// Sentinel errors. Repositories and services return these; the HTTP layer
// maps them to status codes.
var (
	ErrNotFound           = errors.New("not found")
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

// ValidationError describes a client mistake in the request payload. Its
// message is safe to return to the caller.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

// Validationf builds a *ValidationError.
func Validationf(format string, args ...any) error {
	return &ValidationError{Message: fmt.Sprintf(format, args...)}
}

// IsValidation reports whether err is (or wraps) a ValidationError.
func IsValidation(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}
