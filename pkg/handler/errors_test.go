package handlerutil

import (
	"errors"
	"testing"
)

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  ValidationError
		want string
	}{
		{
			name: "Should return custom message when provided",
			err: ValidationError{
				Field:   "email",
				Value:   "invalid-email",
				Message: "Email format is invalid",
			},
			want: "Email format is invalid",
		},
		{
			name: "Should return field error message when only field is set",
			err: ValidationError{
				Field: "username",
				Value: "test",
			},
			want: "validation failed for field 'username'",
		},
		{
			name: "Should return default error when all fields are empty",
			err:  ValidationError{},
			want: "validation error",
		},
		{
			name: "Should return message when errors list is provided",
			err: ValidationError{
				Message: "Multiple validation errors occurred",
				Errors: []string{
					"email: invalid format",
					"password: too short",
				},
			},
			want: "Multiple validation errors occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("ValidationError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidationError_Is(t *testing.T) {
	tests := []struct {
		name   string
		err    ValidationError
		target error
		want   bool
	}{
		{
			name:   "Should match ErrValidation",
			err:    ValidationError{Message: "test"},
			target: ErrValidation,
			want:   true,
		},
		{
			name:   "Should not match ErrNotFound",
			err:    ValidationError{Message: "test"},
			target: ErrNotFound,
			want:   false,
		},
		{
			name:   "Should not match custom error",
			err:    ValidationError{Message: "test"},
			target: errors.New("custom error"),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Is(tt.target); got != tt.want {
				t.Errorf("ValidationError.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewValidationError(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		value   interface{}
		message string
		want    ValidationError
	}{
		{
			name:    "Should create validation error with string value",
			field:   "email",
			value:   "invalid-email",
			message: "Email format is invalid",
			want: ValidationError{
				Field:   "email",
				Value:   "invalid-email",
				Message: "Email format is invalid",
			},
		},
		{
			name:    "Should create validation error with numeric value",
			field:   "age",
			value:   15,
			message: "Age must be at least 18",
			want: ValidationError{
				Field:   "age",
				Value:   15,
				Message: "Age must be at least 18",
			},
		},
		{
			name:    "Should create validation error with nil value",
			field:   "username",
			value:   nil,
			message: "Username is required",
			want: ValidationError{
				Field:   "username",
				Value:   nil,
				Message: "Username is required",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewValidationError(tt.field, tt.value, tt.message)
			if got.Field != tt.want.Field {
				t.Errorf("NewValidationError().Field = %v, want %v", got.Field, tt.want.Field)
			}
			if got.Value != tt.want.Value {
				t.Errorf("NewValidationError().Value = %v, want %v", got.Value, tt.want.Value)
			}
			if got.Message != tt.want.Message {
				t.Errorf("NewValidationError().Message = %v, want %v", got.Message, tt.want.Message)
			}
		})
	}
}

func TestNewValidationErrorWithErrors(t *testing.T) {
	tests := []struct {
		name    string
		message string
		errs    []string
		want    ValidationError
	}{
		{
			name:    "Should create validation error with multiple errors",
			message: "Multiple validation errors occurred",
			errs: []string{
				"email: Email format is invalid",
				"password: Password must be at least 8 characters",
				"username: Username is already taken",
			},
			want: ValidationError{
				Message: "Multiple validation errors occurred",
				Errors: []string{
					"email: Email format is invalid",
					"password: Password must be at least 8 characters",
					"username: Username is already taken",
				},
			},
		},
		{
			name:    "Should create validation error with empty errors list",
			message: "Validation failed",
			errs:    []string{},
			want: ValidationError{
				Message: "Validation failed",
				Errors:  []string{},
			},
		},
		{
			name:    "Should create validation error with single error",
			message: "Request validation failed",
			errs:    []string{"body: Request body is required"},
			want: ValidationError{
				Message: "Request validation failed",
				Errors:  []string{"body: Request body is required"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewValidationErrorWithErrors(tt.message, tt.errs)
			if got.Message != tt.want.Message {
				t.Errorf("NewValidationErrorWithErrors().Message = %v, want %v", got.Message, tt.want.Message)
			}
			if len(got.Errors) != len(tt.want.Errors) {
				t.Errorf("NewValidationErrorWithErrors().Errors length = %v, want %v", len(got.Errors), len(tt.want.Errors))
				return
			}
			for i := range got.Errors {
				if got.Errors[i] != tt.want.Errors[i] {
					t.Errorf("NewValidationErrorWithErrors().Errors[%d] = %v, want %v", i, got.Errors[i], tt.want.Errors[i])
				}
			}
		})
	}
}

func TestValidationError_WithErrorsIs(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{
			name:   "Should work with errors.Is for ValidationError",
			err:    NewValidationError("email", "test", "invalid"),
			target: ErrValidation,
			want:   true,
		},
		{
			name:   "Should work with errors.Is for wrapped ValidationError",
			err:    NewValidationErrorWithErrors("test", []string{"error1"}),
			target: ErrValidation,
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(tt.err, tt.target); got != tt.want {
				t.Errorf("errors.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidationError_ErrorInterface(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "Should implement error interface for ValidationError",
			err:  NewValidationError("email", "test", "invalid email"),
			want: "invalid email",
		},
		{
			name: "Should implement error interface for ValidationErrorWithErrors",
			err:  NewValidationErrorWithErrors("validation failed", []string{"error1", "error2"}),
			want: "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("error.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}
