package logutil

import (
	"errors"
	"runtime/debug"

	"go.uber.org/zap"
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
