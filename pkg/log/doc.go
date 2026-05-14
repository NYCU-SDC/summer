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
// Error helpers in this package let errors carry structured detail to the place
// where they are logged. Domain errors may implement InfoCarrier directly, while
// WrapInfoError and WrapTypedInfoError can attach detail to an existing error
// when passing it across an API boundary.
package logutil
