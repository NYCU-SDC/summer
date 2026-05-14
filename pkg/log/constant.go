package logutil

type ErrorType string

const (
	OK                  ErrorType = "OK"
	CANCELLED           ErrorType = "CANCELLED"
	UNKNOWN             ErrorType = "UNKNOWN"
	INVALID_ARGUMENT    ErrorType = "INVALID_ARGUMENT"
	DEADLINE_EXCEEDED   ErrorType = "DEADLINE_EXCEEDED"
	NOT_FOUND           ErrorType = "NOT_FOUND"
	ALREADY_EXISTS      ErrorType = "ALREADY_EXISTS"
	PERMISSION_DENIED   ErrorType = "PERMISSION_DENIED"
	UNAUTHENTICATED     ErrorType = "UNAUTHENTICATED"
	RESOURCE_EXHAUSTED  ErrorType = "RESOURCE_EXHAUSTED"
	FAILED_PRECONDITION ErrorType = "FAILED_PRECONDITION"
	ABORTED             ErrorType = "ABORTED"
	OUT_OF_RANGE        ErrorType = "OUT_OF_RANGE"
	UNIMPLEMENTED       ErrorType = "UNIMPLEMENTED"
	INTERNAL            ErrorType = "INTERNAL"
	UNAVAILABLE         ErrorType = "UNAVAILABLE"
	DATA_LOSS           ErrorType = "DATA_LOSS"
)
