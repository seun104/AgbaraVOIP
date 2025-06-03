package domain

import "errors"

var (
	ErrNotFound = errors.New("requested item not found")
	ErrConflict = errors.New("item already exists or conflict with existing item")
	// Add other common domain errors here as needed
	ErrAuthentication       = errors.New("authentication failed")
	ErrAuthorization      = errors.New("authorization failed")
	ErrInvalidInput       = errors.New("invalid input")
	ErrOperationFailed    = errors.New("operation failed")
	ErrResourceExhausted  = errors.New("resource exhausted")
	ErrExternalService    = errors.New("external service error")
	ErrNotImplemented     = errors.New("not implemented")
)
