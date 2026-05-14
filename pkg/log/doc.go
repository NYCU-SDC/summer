// Package logutil provides the project's structured logging helpers on top of zap.
//
// The package intentionally supports two usage paths.
//
// Context-first logging stores request-scoped fields in context.Context with
// helpers such as WithFields, WithUserID, and WithRequestID. The level helpers
// such as Info and Error then call Constructs to inject those fields, active
// OpenTelemetry trace fields, user fields, request fields, and source code
// location fields into the log entry.
//
// Logger-first logging decorates an existing *zap.Logger directly with helpers
// such as WithEventName, WithEventOutcome, WithEventAction, and WithErrorType.
// Use this path when a logger already represents a specific event, domain
// operation, or error handling branch.
//
// Error helpers in this package are for extracting structured log fields from
// errors. They should not be used as a general domain error wrapping policy.
package logutil
