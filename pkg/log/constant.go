package logutil

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
