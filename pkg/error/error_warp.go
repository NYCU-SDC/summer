package errutil

import (
	"errors"
	"runtime/debug"

	"go.uber.org/zap"
)

// ErrorType is a stable machine-readable error classification for logs.
//
// The values mirror canonical status-like categories so services can group
// unrelated Go error types under the same operational meaning.
type ErrorType string

const (
	// OK indicates that no error occurred.
	OK ErrorType = "OK"
	// CANCELLED indicates that an operation was cancelled before completion.
	CANCELLED ErrorType = "CANCELLED"
	// UNKNOWN indicates that the error category could not be determined.
	UNKNOWN ErrorType = "UNKNOWN"
	// INVALID_ARGUMENT indicates that the caller supplied invalid input.
	INVALID_ARGUMENT ErrorType = "INVALID_ARGUMENT"
	// DEADLINE_EXCEEDED indicates that an operation exceeded its time limit.
	DEADLINE_EXCEEDED ErrorType = "DEADLINE_EXCEEDED"
	// NOT_FOUND indicates that a requested resource does not exist.
	NOT_FOUND ErrorType = "NOT_FOUND"
	// ALREADY_EXISTS indicates that a resource already exists.
	ALREADY_EXISTS ErrorType = "ALREADY_EXISTS"
	// PERMISSION_DENIED indicates that the caller lacks permission.
	PERMISSION_DENIED ErrorType = "PERMISSION_DENIED"
	// UNAUTHENTICATED indicates that authentication is required or invalid.
	UNAUTHENTICATED ErrorType = "UNAUTHENTICATED"
	// RESOURCE_EXHAUSTED indicates that quota, capacity, or another limit was exhausted.
	RESOURCE_EXHAUSTED ErrorType = "RESOURCE_EXHAUSTED"
	// FAILED_PRECONDITION indicates that the system state does not allow the operation.
	FAILED_PRECONDITION ErrorType = "FAILED_PRECONDITION"
	// ABORTED indicates that an operation was aborted, often due to a conflict.
	ABORTED ErrorType = "ABORTED"
	// OUT_OF_RANGE indicates that an input is outside the allowed range.
	OUT_OF_RANGE ErrorType = "OUT_OF_RANGE"
	// UNIMPLEMENTED indicates that the requested operation is not implemented.
	UNIMPLEMENTED ErrorType = "UNIMPLEMENTED"
	// INTERNAL indicates an unexpected server-side failure.
	INTERNAL ErrorType = "INTERNAL"
	// UNAVAILABLE indicates that a dependency or service is temporarily unavailable.
	UNAVAILABLE ErrorType = "UNAVAILABLE"
	// DATA_LOSS indicates unrecoverable data corruption or loss.
	DATA_LOSS ErrorType = "DATA_LOSS"
)

// ErrorInfoKey defines common keys for structured metadata carried by errors.
//
// These keys are intended for InfoError maps and become error.info.<key> fields
// when emitted by ErrorFields or ErrorFieldsWithStacktrace.
type ErrorInfoKey string

const (
	// ErrorInfoReason describes why an error occurred or a branch was taken.
	ErrorInfoReason ErrorInfoKey = "reason"
	// ErrorInfoOperation names the operation that produced the error.
	ErrorInfoOperation ErrorInfoKey = "operation"
	// ErrorInfoRetryable records whether retrying the operation may succeed.
	ErrorInfoRetryable ErrorInfoKey = "retryable"
	// ErrorInfoField names the input or domain field related to the error.
	ErrorInfoField ErrorInfoKey = "field"
	// ErrorInfoUserID records the user ID related to the error.
	ErrorInfoUserID ErrorInfoKey = "user.id"
	// ErrorInfoRequestID records the request ID related to the error.
	ErrorInfoRequestID ErrorInfoKey = "request.id"
)

// InfoCarrier marks an error as carrying structured log metadata.
//
// ErrorFields and InfoFieldsFromError use errors.As to find this interface in
// an error chain and emit each returned key/value pair as log fields.
type InfoCarrier interface {
	LogInfo() map[string]any
}

// ErrorTypeCarrier marks an error as carrying a stable error.type value.
//
// ErrorFields uses errors.As to find this interface in an error chain and emit
// the returned value as error.type.
type ErrorTypeCarrier interface {
	LogErrorType() ErrorType
}

// InfoError is an error wrapper that carries structured logging metadata.
//
// It preserves Base for errors.Is/errors.As through Unwrap, optionally carries
// a canonical ErrorType, and exposes Info as string-keyed log metadata through
// LogInfo. Use it only when an error must carry observability metadata across an
// API boundary; do not use it as a general domain error wrapping policy.
type InfoError[K ~string, V any] struct {
	Base error
	Type ErrorType
	Info map[K]V
}

// Error returns the base error message, falling back to Type when Base is nil.
func (e *InfoError[K, V]) Error() string {
	if e == nil {
		return ""
	}

	if e.Base != nil {
		return e.Base.Error()
	}

	if e.Type != "" {
		return string(e.Type)
	}

	return ""
}

// Unwrap returns the wrapped base error for errors.Is and errors.As.
func (e *InfoError[K, V]) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Base
}

// LogInfo returns Info with string keys for structured logging.
//
// A new map is allocated so callers cannot mutate the original Info map through
// the returned value.
func (e *InfoError[K, V]) LogInfo() map[string]any {
	if e == nil || len(e.Info) == 0 {
		return nil
	}

	out := make(map[string]any, len(e.Info))
	for k, v := range e.Info {
		out[string(k)] = v
	}

	return out
}

// LogErrorType returns the canonical error type associated with the error.
func (e *InfoError[K, V]) LogErrorType() ErrorType {
	if e == nil {
		return ""
	}

	return e.Type
}

// NewInfoError returns an error carrying structured logging metadata.
//
// The returned error wraps base and exposes info through InfoCarrier. This
// constructor does not set error.type; use NewTypedInfoError when a stable error
// category is available.
func NewInfoError[K ~string, V any](base error, info map[K]V) error {
	return &InfoError[K, V]{
		Base: base,
		Info: info,
	}
}

// WrapInfoError returns an error carrying structured logging metadata.
//
// It is a compatibility alias for NewInfoError. Prefer NewInfoError for new code
// when the intent is to create an error that carries log metadata.
func WrapInfoError[K ~string, V any](base error, info map[K]V) error {
	return &InfoError[K, V]{
		Base: base,
		Info: info,
	}
}

// NewTypedInfoError returns an error carrying error.type and structured metadata.
//
// The returned error wraps base, exposes errorType through ErrorTypeCarrier, and
// exposes info through InfoCarrier.
func NewTypedInfoError[K ~string, V any](errorType ErrorType, base error, info map[K]V) error {
	return &InfoError[K, V]{
		Base: base,
		Type: errorType,
		Info: info,
	}
}

// WrapTypedInfoError returns an error carrying error.type and structured metadata.
//
// It is a compatibility alias for NewTypedInfoError. Prefer NewTypedInfoError
// for new code when the intent is to create an error that carries log metadata.
func WrapTypedInfoError[K ~string, V any](errorType ErrorType, base error, info map[K]V) error {
	return &InfoError[K, V]{
		Base: base,
		Type: errorType,
		Info: info,
	}
}

// InfoFromError extracts structured logging metadata from err.
//
// It returns nil when err is nil or no InfoCarrier is found in the error chain.
func InfoFromError(err error) map[string]any {
	if err == nil {
		return nil
	}

	var carrier InfoCarrier
	if errors.As(err, &carrier) {
		return carrier.LogInfo()
	}

	return nil
}

// ErrorTypeFromError extracts a stable error type from err.
//
// It returns an empty ErrorType when err is nil or no ErrorTypeCarrier is found
// in the error chain.
func ErrorTypeFromError(err error) ErrorType {
	if err == nil {
		return ""
	}

	var carrier ErrorTypeCarrier
	if errors.As(err, &carrier) {
		return carrier.LogErrorType()
	}

	return ""
}

// ErrorFields converts err into standard zap fields for structured error logs.
//
// The returned fields include zap.Error, error.message, exception.message,
// exception.type, optional error.type, and optional error.info.* fields.
// Stacktrace is intentionally excluded; use ErrorFieldsWithStacktrace when the
// call site needs a stacktrace.
func ErrorFields(err error) []zap.Field {
	if err == nil {
		return nil
	}

	fields := []zap.Field{
		zap.Error(err),
		zap.String("error.message", err.Error()),
	}

	if errorType := ErrorTypeFromError(err); errorType != "" {
		fields = append(fields, zap.String("error.type", string(errorType)))
	}

	fields = append(fields, InfoFieldsFromError("error.info", err)...)

	return fields
}

// ErrorFieldsWithStacktrace converts err into zap fields including a stacktrace.
//
// It includes all fields from ErrorFields and adds exception.stacktrace captured
// at the point this function is called.
func ErrorFieldsWithStacktrace(err error) []zap.Field {
	if err == nil {
		return nil
	}

	fields := ErrorFields(err)
	fields = append(fields, zap.String("exception.stacktrace", string(debug.Stack())))

	return fields
}

// InfoFieldsFromError converts InfoCarrier metadata into namespaced zap fields.
//
// If prefix is empty, fields are emitted under error.info. For example, an info
// key "operation" with the default prefix becomes error.info.operation.
func InfoFieldsFromError(prefix string, err error) []zap.Field {
	info := InfoFromError(err)
	if len(info) == 0 {
		return nil
	}

	// TODO: should we keep the prefix here?
	if prefix == "" {
		prefix = "error.info"
	}

	fields := make([]zap.Field, 0, len(info))
	for key, value := range info {
		if key == "" {
			continue
		}

		fields = append(fields, zap.Any(prefix+"."+key, value))
	}

	return fields
}
