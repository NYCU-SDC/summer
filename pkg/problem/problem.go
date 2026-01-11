package problem

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/NYCU-SDC/summer/pkg/database"
	"github.com/NYCU-SDC/summer/pkg/handler"
	"github.com/NYCU-SDC/summer/pkg/pagination"
	"github.com/go-playground/validator/v10"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"net/http"
)

// Problem represents a problem detail as defined in RFC 7807
type Problem struct {
	Title  string `json:"title"`
	Status int    `json:"status"`

	// Type indicates the URI that identifies the problem type.
	// In production, this would point to the project's documentation.
	// For demonstration purposes, we use an MDN URI here.
	Type   string `json:"type"`
	Detail string `json:"detail"`
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

func (h *HttpWriter) WriteError(ctx context.Context, w http.ResponseWriter, err error, logger *zap.Logger) {
	_, span := otel.Tracer("problem/problem").Start(ctx, "WriteError")
	defer span.End()

	if err == nil {
		return
	}

	var problem Problem

	// Check if the error matches the custom error type
	problem = h.ProblemMapping(err)

	// If the problem is still empty, check for standard error types
	if problem == (Problem{}) {
		var notFoundError handlerutil.NotFoundError
		var validationErrors validator.ValidationErrors
		var internalDbError databaseutil.InternalServerError
		switch {
		case errors.As(err, &notFoundError):
			problem = NewNotFoundProblem(err.Error())
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

	logger = logger.WithOptions(zap.AddCallerSkip(1))

	logger.Warn("Handling "+problem.Title, zap.String("problem", problem.Title), zap.Error(err), zap.Int("status", problem.Status), zap.String("type", problem.Type), zap.String("detail", problem.Detail))

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(problem.Status)
	jsonBytes, err := json.Marshal(problem)
	if err != nil {
		logger.Error("Failed to marshal problem response", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = w.Write(jsonBytes)
	if err != nil {
		logger.Error("Failed to write problem response", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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
