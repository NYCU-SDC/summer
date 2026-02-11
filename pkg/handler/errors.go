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
	ErrValidation        = errors.New("validation error")
)

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

type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
	Errors  []string
}

func (e ValidationError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Field != "" {
		return fmt.Sprintf("validation failed for field '%s'", e.Field)
	}
	return ErrValidation.Error()
}

func (e ValidationError) Is(target error) bool {
	return errors.Is(target, ErrValidation)
}

func NewValidationError(field string, value interface{}, message string) ValidationError {
	return ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

func NewValidationErrorWithErrors(message string, errs []string) ValidationError {
	return ValidationError{
		Message: message,
		Errors:  errs,
	}
}
