package problem

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/NYCU-SDC/summer/pkg/database"
	errutil "github.com/NYCU-SDC/summer/pkg/error"
	"github.com/NYCU-SDC/summer/pkg/handler"
	logutil "github.com/NYCU-SDC/summer/pkg/log"
	"github.com/NYCU-SDC/summer/pkg/pagination"
	"github.com/go-playground/validator/v10"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

// Problem represents a problem detail as defined in RFC 9457
type Problem struct {
	Title  string `json:"title"`
	Status int    `json:"status"`

	// Type indicates the URI that identifies the problem type.
	// In production, this would point to the project's documentation.
	// For demonstration purposes, we use an MDN URI here.
	Type   string `json:"type"`
	Detail string `json:"detail"`

	Instance string `json:"instance,omitempty"`

	Errors []string `json:"errors,omitempty"`
}

func (p Problem) IsEmpty() bool {
	return p.Title == "" && p.Status == 0 && p.Type == "" && p.Detail == "" && p.Instance == "" && len(p.Errors) == 0
}

type HttpWriter struct {
	ProblemMapping func(error) Problem
}

func New() *HttpWriter {
	return &HttpWriter{
		ProblemMapping: func(err error) Problem {
			return Problem{}
		},
	}
}

func NewWithMapping(ProblemMapping func(error) Problem) *HttpWriter {
	return &HttpWriter{
		ProblemMapping: ProblemMapping,
	}
}

// buildProblem converts an error into a Problem struct
func (h *HttpWriter) buildProblem(err error) Problem {
	// Check if the error matches the custom error type
	problem := h.ProblemMapping(err)

	// If the problem is still empty, check for standard error types
	if problem.IsEmpty() {
		var notFoundError handlerutil.NotFoundError
		var validationError handlerutil.ValidationError
		var validationErrors validator.ValidationErrors
		var internalDbError databaseutil.InternalServerError
		switch {
		case errors.As(err, &notFoundError):
			problem = NewNotFoundProblem(err.Error())
		case errors.As(err, &validationError):
			if len(validationError.Errors) > 0 {
				problem = NewValidateProblemWithErrors(validationError.Error(), validationError.Errors)
			} else {
				problem = NewValidateProblem(validationError.Error())
			}
		case errors.As(err, &validationErrors):
			problem = NewValidateProblem(validationErrors.Error())
		case errors.Is(err, handlerutil.ErrUserAlreadyExists):
			problem = NewValidateProblem("User already exists")
		case errors.Is(err, handlerutil.ErrCredentialInvalid):
			problem = NewUnauthorizedProblem("Invalid username or password")
		case errors.Is(err, handlerutil.ErrForbidden):
			problem = NewForbiddenProblem("Make sure you have the right permissions")
		case errors.Is(err, handlerutil.ErrUnauthorized):
			problem = NewUnauthorizedProblem("You must be logged in to access this resource")
		case errors.Is(err, handlerutil.ErrInvalidUUID):
			problem = NewValidateProblem("Invalid UUID format")
		case errors.Is(err, handlerutil.ErrValidation):
			problem = NewValidateProblem("Validation error")
		case errors.Is(err, handlerutil.ErrNotFound):
			problem = NewNotFoundProblem("Resource not found")
		case errors.As(err, &internalDbError):
			problem = NewInternalServerProblem("Internal server error")
		case errors.Is(err, pagination.ErrInvalidPageOrSize):
			problem = NewValidateProblem("Invalid page or size")
		case errors.Is(err, pagination.ErrInvalidSortingField):
			problem = NewValidateProblem("Invalid sorting field")
		default:
			problem = NewInternalServerProblem("Internal server error")
		}
	}

	return problem
}

// writeProblemResponse writes the Problem struct as JSON to the response writer
func (h *HttpWriter) writeProblemResponse(ctx context.Context, w http.ResponseWriter, problem Problem, err error, logger *zap.Logger) {
	logger = logutil.Constructs(ctx, logger).WithOptions(zap.AddCallerSkip(2))

	fields := []zap.Field{
		zap.Int("http.status_code", problem.Status),
		zap.String("problem.type", problem.Type),
		zap.String("problem.title", problem.Title),
		zap.String("problem.detail", problem.Detail),
		zap.String("error.kind", problemErrorKind(problem.Status)),
	}
	if problem.Instance != "" {
		fields = append(fields, zap.String("problem.instance", problem.Instance))
	}

	if problem.Status >= http.StatusInternalServerError {
		fields = append(fields, errutil.ErrorFieldsWithStacktrace(err)...)
		logger.Error("request failed", fields...)
	} else {
		fields = append(fields, errutil.ErrorFields(err)...)
		logger.Warn("request failed", fields...)
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(problem.Status)
	jsonBytes, marshalErr := json.Marshal(problem)
	if marshalErr != nil {
		marshalFields := []zap.Field{
			zap.Int("http.status_code", http.StatusInternalServerError),
			zap.String("problem.title", "Internal Server Error"),
			zap.String("error.kind", problemErrorKind(http.StatusInternalServerError)),
		}
		marshalFields = append(marshalFields, errutil.ErrorFields(marshalErr)...)
		logger.Error("problem response marshal failed", marshalFields...)
		http.Error(w, marshalErr.Error(), http.StatusInternalServerError)
		return
	}

	_, writeErr := w.Write(jsonBytes)
	if writeErr != nil {
		writeFields := []zap.Field{
			zap.Int("http.status_code", http.StatusInternalServerError),
			zap.String("problem.title", "Internal Server Error"),
			zap.String("error.kind", problemErrorKind(http.StatusInternalServerError)),
		}
		writeFields = append(writeFields, errutil.ErrorFields(writeErr)...)
		logger.Error("problem response write failed", writeFields...)
		http.Error(w, writeErr.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *HttpWriter) WriteError(ctx context.Context, w http.ResponseWriter, err error, logger *zap.Logger) {
	_, span := otel.Tracer("problem/problem").Start(ctx, "WriteError")
	defer span.End()

	if err == nil {
		return
	}

	problem := h.buildProblem(err)
	h.writeProblemResponse(ctx, w, problem, err, logger)
}

func (h *HttpWriter) WriteErrorWithRequest(ctx context.Context, r *http.Request, w http.ResponseWriter, err error, logger *zap.Logger) {
	_, span := otel.Tracer("problem/problem").Start(ctx, "WriteErrorWithRequest")
	defer span.End()

	if err == nil {
		return
	}

	problem := h.buildProblem(err)
	if r != nil && r.URL != nil {
		problem.Instance = r.URL.Path
	}
	h.writeProblemResponse(ctx, w, problem, err, logger)
}

func problemErrorKind(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "INVALID_ARGUMENT"
	case http.StatusUnauthorized:
		return "UNAUTHENTICATED"
	case http.StatusForbidden:
		return "PERMISSION_DENIED"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusConflict:
		return "ALREADY_EXISTS"
	case http.StatusTooManyRequests:
		return "RESOURCE_EXHAUSTED"
	case http.StatusRequestTimeout, http.StatusGatewayTimeout:
		return "DEADLINE_EXCEEDED"
	}

	if status >= http.StatusInternalServerError {
		return "INTERNAL"
	}

	return "UNKNOWN"
}

func NewInternalServerProblem(detail string) Problem {
	return Problem{
		Title:  "Internal Server Error",
		Status: http.StatusInternalServerError,
		Type:   "https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/500",
		Detail: detail,
	}
}

func NewNotFoundProblem(detail string) Problem {
	return Problem{
		Title:  "Not Found",
		Status: http.StatusNotFound,
		Type:   "https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/404",
		Detail: detail,
	}
}

func NewValidateProblem(detail string) Problem {
	return Problem{
		Title:  "Validation Problem",
		Status: http.StatusBadRequest,
		Type:   "https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/400",
		Detail: detail,
	}
}

func NewValidateProblemWithErrors(detail string, errors []string) Problem {
	return Problem{
		Title:  "Validation Problem",
		Status: http.StatusBadRequest,
		Type:   "https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/400",
		Detail: detail,
		Errors: errors,
	}
}

func NewUnauthorizedProblem(detail string) Problem {
	return Problem{
		Title:  "Unauthorized",
		Status: http.StatusUnauthorized,
		Type:   "https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/401",
		Detail: detail,
	}
}

func NewForbiddenProblem(detail string) Problem {
	return Problem{
		Title:  "Forbidden",
		Status: http.StatusForbidden,
		Type:   "https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/403",
		Detail: detail,
	}
}

func NewBadRequestProblem(detail string) Problem {
	return Problem{
		Title:  "Bad Request",
		Status: http.StatusBadRequest,
		Type:   "https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/400",
		Detail: detail,
	}
}
