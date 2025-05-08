package handlerutil

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound          = errors.New("record not found")
	ErrForbidden         = errors.New("forbidden")
	ErrCredentialInvalid = errors.New("invalid username or password")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrInternalServer    = errors.New("internal server error")
	ErrInvalidUUID       = errors.New("failed to parse UUID")
)

type InvalidUUIDError struct {
	UUID    string
	Message string
}

func (e InvalidUUIDError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.UUID != "" {
		return fmt.Sprintf("failed to parse UUID '%s'", e.UUID)
	}
	return ErrInvalidUUID.Error()
}

func (e InvalidUUIDError) Is(target error) bool {
	return errors.Is(target, ErrInvalidUUID)
}

func NewInvalidUUIDError(uuid, message string) InvalidUUIDError {
	return InvalidUUIDError{
		UUID:    uuid,
		Message: message,
	}
}

type NotFoundError struct {
	Table   string
	Key     string
	Value   string
	Message string
}

func (e NotFoundError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Key != "" && e.Value != "" {
		return fmt.Sprintf("unable to find %s with %s '%s'", e.Table, e.Key, e.Value)
	}
	return ErrNotFound.Error()
}

func (e NotFoundError) Is(target error) bool {
	return errors.Is(target, ErrNotFound)
}

func NewNotFoundError(table, key, value, message string) NotFoundError {
	return NotFoundError{
		Table:   table,
		Key:     key,
		Value:   value,
		Message: message,
	}
}
