package logutil

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

// EventOutcome describes the result of a logged event.
//
// Use it with WithEventOutcome when the event result fits one of the standard
// outcome values. Use WithOutcome only when a custom outcome string is needed.
type EventOutcome string

const (
	// EventOutcomeSuccess indicates that an event completed successfully.
	EventOutcomeSuccess EventOutcome = "success"
	// EventOutcomeFailure indicates that an event failed.
	EventOutcomeFailure EventOutcome = "failure"
	// EventOutcomeCancelled indicates that an event was cancelled.
	EventOutcomeCancelled EventOutcome = "cancelled"
	// EventOutcomeTimeout indicates that an event timed out.
	EventOutcomeTimeout EventOutcome = "timeout"
	// EventOutcomeUnknown indicates that the event result is not known.
	EventOutcomeUnknown EventOutcome = "unknown"
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
