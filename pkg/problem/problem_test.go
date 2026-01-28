package problem

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	handlerutil "github.com/NYCU-SDC/summer/pkg/handler"
	"github.com/NYCU-SDC/summer/pkg/pagination"
	"go.uber.org/zap"
)

func TestWriteError_ValidationError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		wantStatus     int
		wantTitle      string
		wantDetail     string
		wantErrorsLen  int
		checkErrorList bool
	}{
		{
			name:       "Should handle simple validation error with message",
			err:        handlerutil.NewValidationError("email", "invalid-email", "Email format is invalid"),
			wantStatus: http.StatusBadRequest,
			wantTitle:  "Validation Problem",
			wantDetail: "Email format is invalid",
		},
		{
			name:       "Should handle validation error with field only",
			err:        handlerutil.NewValidationError("username", "test", ""),
			wantStatus: http.StatusBadRequest,
			wantTitle:  "Validation Problem",
			wantDetail: "validation failed for field 'username'",
		},
		{
			name: "Should handle validation error with multiple errors",
			err: handlerutil.NewValidationErrorWithErrors(
				"Multiple validation errors occurred",
				[]string{
					"email: Email format is invalid",
					"password: Password must be at least 8 characters",
					"username: Username is already taken",
				},
			),
			wantStatus:     http.StatusBadRequest,
			wantTitle:      "Validation Problem",
			wantDetail:     "Multiple validation errors occurred",
			wantErrorsLen:  3,
			checkErrorList: true,
		},
		{
			name: "Should handle validation error with empty errors list",
			err: handlerutil.NewValidationErrorWithErrors(
				"Validation failed",
				[]string{},
			),
			wantStatus: http.StatusBadRequest,
			wantTitle:  "Validation Problem",
			wantDetail: "Validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a response recorder
			w := httptest.NewRecorder()

			// Create logger
			logger, _ := zap.NewDevelopment()

			// Create HttpWriter
			hw := New()

			// Call WriteError
			ctx := context.Background()
			hw.WriteError(ctx, w, tt.err, logger)

			// Check status code
			if w.Code != tt.wantStatus {
				t.Errorf("WriteError() status = %v, want %v", w.Code, tt.wantStatus)
			}

			// Check content type
			contentType := w.Header().Get("Content-Type")
			if contentType != "application/problem+json" {
				t.Errorf("WriteError() Content-Type = %v, want application/problem+json", contentType)
			}

			// Decode response
			var problem Problem
			if err := json.NewDecoder(w.Body).Decode(&problem); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// Check title
			if problem.Title != tt.wantTitle {
				t.Errorf("WriteError() title = %v, want %v", problem.Title, tt.wantTitle)
			}

			// Check detail
			if problem.Detail != tt.wantDetail {
				t.Errorf("WriteError() detail = %v, want %v", problem.Detail, tt.wantDetail)
			}

			// Check status
			if problem.Status != tt.wantStatus {
				t.Errorf("WriteError() problem.Status = %v, want %v", problem.Status, tt.wantStatus)
			}

			// Check type
			if problem.Type != "https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/400" {
				t.Errorf("WriteError() type = %v, want https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/400", problem.Type)
			}

			// Check errors list if needed
			if tt.checkErrorList {
				if len(problem.Errors) != tt.wantErrorsLen {
					t.Errorf("WriteError() errors length = %v, want %v", len(problem.Errors), tt.wantErrorsLen)
				}
			}
		})
	}
}

func TestNewValidateProblemWithErrors(t *testing.T) {
	tests := []struct {
		name       string
		detail     string
		errors     []string
		wantStatus int
		wantTitle  string
	}{
		{
			name:   "Should create problem with multiple errors",
			detail: "Multiple validation errors occurred",
			errors: []string{
				"email: invalid format",
				"password: too short",
			},
			wantStatus: http.StatusBadRequest,
			wantTitle:  "Validation Problem",
		},
		{
			name:       "Should create problem with empty errors",
			detail:     "Validation failed",
			errors:     []string{},
			wantStatus: http.StatusBadRequest,
			wantTitle:  "Validation Problem",
		},
		{
			name:       "Should create problem with single error",
			detail:     "Request validation failed",
			errors:     []string{"body: required"},
			wantStatus: http.StatusBadRequest,
			wantTitle:  "Validation Problem",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			problem := NewValidateProblemWithErrors(tt.detail, tt.errors)

			if problem.Status != tt.wantStatus {
				t.Errorf("NewValidateProblemWithErrors().Status = %v, want %v", problem.Status, tt.wantStatus)
			}

			if problem.Title != tt.wantTitle {
				t.Errorf("NewValidateProblemWithErrors().Title = %v, want %v", problem.Title, tt.wantTitle)
			}

			if problem.Detail != tt.detail {
				t.Errorf("NewValidateProblemWithErrors().Detail = %v, want %v", problem.Detail, tt.detail)
			}

			if len(problem.Errors) != len(tt.errors) {
				t.Errorf("NewValidateProblemWithErrors().Errors length = %v, want %v", len(problem.Errors), len(tt.errors))
			}

			for i := range problem.Errors {
				if problem.Errors[i] != tt.errors[i] {
					t.Errorf("NewValidateProblemWithErrors().Errors[%d] = %v, want %v", i, problem.Errors[i], tt.errors[i])
				}
			}
		})
	}
}

func TestProblem_JSONSerialization(t *testing.T) {
	tests := []struct {
		name    string
		problem Problem
		want    string
	}{
		{
			name: "Should serialize problem with errors list",
			problem: Problem{
				Title:  "Validation Problem",
				Status: 400,
				Type:   "https://example.com/probs/validation",
				Detail: "Multiple errors",
				Errors: []string{"error1", "error2"},
			},
			want: `{"title":"Validation Problem","status":400,"type":"https://example.com/probs/validation","detail":"Multiple errors","errors":["error1","error2"]}`,
		},
		{
			name: "Should serialize problem without errors list",
			problem: Problem{
				Title:  "Not Found",
				Status: 404,
				Type:   "https://example.com/probs/not-found",
				Detail: "Resource not found",
			},
			want: `{"title":"Not Found","status":404,"type":"https://example.com/probs/not-found","detail":"Resource not found"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.problem)
			if err != nil {
				t.Fatalf("Failed to marshal problem: %v", err)
			}

			// Compare JSON strings
			var gotMap, wantMap map[string]interface{}
			if err := json.Unmarshal(got, &gotMap); err != nil {
				t.Fatalf("Failed to unmarshal got JSON: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.want), &wantMap); err != nil {
				t.Fatalf("Failed to unmarshal want JSON: %v", err)
			}

			gotJSON, _ := json.Marshal(gotMap)
			wantJSON, _ := json.Marshal(wantMap)

			if !bytes.Equal(gotJSON, wantJSON) {
				t.Errorf("JSON serialization mismatch:\ngot:  %s\nwant: %s", string(gotJSON), string(wantJSON))
			}
		})
	}
}

func TestWriteError_Integration(t *testing.T) {
	tests := []struct {
		name       string
		setupError func() error
		wantStatus int
		wantTitle  string
	}{
		{
			name: "should handle ValidationError correctly",
			setupError: func() error {
				return handlerutil.NewValidationError("email", "test", "invalid email")
			},
			wantStatus: http.StatusBadRequest,
			wantTitle:  "Validation Problem",
		},
		{
			name: "should handle NotFoundError correctly",
			setupError: func() error {
				return handlerutil.NewNotFoundError("users", "id", "123", "")
			},
			wantStatus: http.StatusNotFound,
			wantTitle:  "Not Found",
		},
		{
			name: "should handle generic validation error",
			setupError: func() error {
				return handlerutil.ErrValidation
			},
			wantStatus: http.StatusBadRequest,
			wantTitle:  "Validation Problem",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			logger, _ := zap.NewDevelopment()
			hw := New()

			err := tt.setupError()
			hw.WriteError(context.Background(), w, err, logger)

			if w.Code != tt.wantStatus {
				t.Errorf("WriteError() status = %v, want %v", w.Code, tt.wantStatus)
			}

			var problem Problem
			if err := json.NewDecoder(w.Body).Decode(&problem); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if problem.Title != tt.wantTitle {
				t.Errorf("WriteError() title = %v, want %v", problem.Title, tt.wantTitle)
			}
		})
	}
}

func TestWriteErrorWithRequest(t *testing.T) {
	tests := []struct {
		name         string
		requestURI   string
		err          error
		wantStatus   int
		wantTitle    string
		wantInstance string
	}{
		{
			name:         "should set instance from request URI for validation error",
			requestURI:   "/api/v1/users/123",
			err:          handlerutil.NewValidationError("email", "test", "invalid email"),
			wantStatus:   http.StatusBadRequest,
			wantTitle:    "Validation Problem",
			wantInstance: "/api/v1/users/123",
		},
		{
			name:         "should set instance from request URI for not found error",
			requestURI:   "/api/v1/posts/456",
			err:          handlerutil.NewNotFoundError("posts", "id", "456", ""),
			wantStatus:   http.StatusNotFound,
			wantTitle:    "Not Found",
			wantInstance: "/api/v1/posts/456",
		},
		{
			name:         "should set instance from request URI with query parameters",
			requestURI:   "/api/v1/search?q=test&limit=10",
			err:          handlerutil.ErrValidation,
			wantStatus:   http.StatusBadRequest,
			wantTitle:    "Validation Problem",
			wantInstance: "/api/v1/search",
		},
		{
			name:       "should handle request with multiple validation errors",
			requestURI: "/api/v1/users",
			err: handlerutil.NewValidationErrorWithErrors(
				"Multiple validation errors",
				[]string{"email: invalid", "password: too short"},
			),
			wantStatus:   http.StatusBadRequest,
			wantTitle:    "Validation Problem",
			wantInstance: "/api/v1/users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest(http.MethodPost, tt.requestURI, nil)
			w := httptest.NewRecorder()
			logger, _ := zap.NewDevelopment()
			hw := New()

			// Call WriteErrorWithRequest
			hw.WriteErrorWithRequest(context.Background(), req, w, tt.err, logger)

			// Check status code
			if w.Code != tt.wantStatus {
				t.Errorf("WriteErrorWithRequest() status = %v, want %v", w.Code, tt.wantStatus)
			}

			// Check content type
			contentType := w.Header().Get("Content-Type")
			if contentType != "application/problem+json" {
				t.Errorf("WriteErrorWithRequest() Content-Type = %v, want application/problem+json", contentType)
			}

			// Decode response
			var problem Problem
			if err := json.NewDecoder(w.Body).Decode(&problem); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// Check title
			if problem.Title != tt.wantTitle {
				t.Errorf("WriteErrorWithRequest() title = %v, want %v", problem.Title, tt.wantTitle)
			}

			// Check instance
			if problem.Instance != tt.wantInstance {
				t.Errorf("WriteErrorWithRequest() instance = %v, want %v", problem.Instance, tt.wantInstance)
			}

			// Check status
			if problem.Status != tt.wantStatus {
				t.Errorf("WriteErrorWithRequest() problem.Status = %v, want %v", problem.Status, tt.wantStatus)
			}
		})
	}
}

func TestWriteErrorWithRequest_NilRequest(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantTitle  string
	}{
		{
			name:       "should handle nil request gracefully",
			err:        handlerutil.NewValidationError("email", "test", "invalid email"),
			wantStatus: http.StatusBadRequest,
			wantTitle:  "Validation Problem",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			logger, _ := zap.NewDevelopment()
			hw := New()

			// Call with nil request
			hw.WriteErrorWithRequest(context.Background(), nil, w, tt.err, logger)

			if w.Code != tt.wantStatus {
				t.Errorf("WriteErrorWithRequest() status = %v, want %v", w.Code, tt.wantStatus)
			}

			var problem Problem
			if err := json.NewDecoder(w.Body).Decode(&problem); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if problem.Title != tt.wantTitle {
				t.Errorf("WriteErrorWithRequest() title = %v, want %v", problem.Title, tt.wantTitle)
			}

			// Instance should be empty when request is nil
			if problem.Instance != "" {
				t.Errorf("WriteErrorWithRequest() instance = %v, want empty string", problem.Instance)
			}
		})
	}
}

func TestHttpWriter_buildProblem(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantTitle  string
		wantType   string
		wantDetail string
	}{
		{
			name:       "Should handle NotFoundError",
			err:        handlerutil.NewNotFoundError("users", "id", "123", ""),
			wantStatus: http.StatusNotFound,
			wantTitle:  "Not Found",
			wantType:   "https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/404",
		},
		{
			name:       "Should handle ValidationError",
			err:        handlerutil.NewValidationError("email", "test", "invalid email format"),
			wantStatus: http.StatusBadRequest,
			wantTitle:  "Validation Problem",
			wantType:   "https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/400",
			wantDetail: "invalid email format",
		},
		{
			name:       "Should handle ValidationError with multiple errors",
			err:        handlerutil.NewValidationErrorWithErrors("Multiple errors", []string{"error1", "error2"}),
			wantStatus: http.StatusBadRequest,
			wantTitle:  "Validation Problem",
			wantType:   "https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/400",
			wantDetail: "Multiple errors",
		},
		{
			name:       "Should handle ErrUserAlreadyExists",
			err:        handlerutil.ErrUserAlreadyExists,
			wantStatus: http.StatusBadRequest,
			wantTitle:  "Validation Problem",
			wantType:   "https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/400",
			wantDetail: "User already exists",
		},
		{
			name:       "Should handle ErrCredentialInvalid",
			err:        handlerutil.ErrCredentialInvalid,
			wantStatus: http.StatusUnauthorized,
			wantTitle:  "Unauthorized",
			wantType:   "https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/401",
			wantDetail: "Invalid username or password",
		},
		{
			name:       "Should handle ErrForbidden",
			err:        handlerutil.ErrForbidden,
			wantStatus: http.StatusForbidden,
			wantTitle:  "Forbidden",
			wantType:   "https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/403",
			wantDetail: "Make sure you have the right permissions",
		},
		{
			name:       "Should handle ErrUnauthorized",
			err:        handlerutil.ErrUnauthorized,
			wantStatus: http.StatusUnauthorized,
			wantTitle:  "Unauthorized",
			wantType:   "https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/401",
			wantDetail: "You must be logged in to access this resource",
		},
		{
			name:       "Should handle ErrInvalidUUID",
			err:        handlerutil.ErrInvalidUUID,
			wantStatus: http.StatusBadRequest,
			wantTitle:  "Validation Problem",
			wantType:   "https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/400",
			wantDetail: "Invalid UUID format",
		},
		{
			name:       "Should handle ErrValidation",
			err:        handlerutil.ErrValidation,
			wantStatus: http.StatusBadRequest,
			wantTitle:  "Validation Problem",
			wantType:   "https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/400",
			wantDetail: "Validation error",
		},
		{
			name:       "Should handle ErrNotFound",
			err:        handlerutil.ErrNotFound,
			wantStatus: http.StatusNotFound,
			wantTitle:  "Not Found",
			wantType:   "https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/404",
			wantDetail: "Resource not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hw := New()
			problem := hw.buildProblem(tt.err)

			if problem.Status != tt.wantStatus {
				t.Errorf("buildProblem().Status = %v, want %v", problem.Status, tt.wantStatus)
			}

			if problem.Title != tt.wantTitle {
				t.Errorf("buildProblem().Title = %v, want %v", problem.Title, tt.wantTitle)
			}

			if problem.Type != tt.wantType {
				t.Errorf("buildProblem().Type = %v, want %v", problem.Type, tt.wantType)
			}

			if tt.wantDetail != "" && problem.Detail != tt.wantDetail {
				t.Errorf("buildProblem().Detail = %v, want %v", problem.Detail, tt.wantDetail)
			}
		})
	}
}

func TestHttpWriter_buildProblem_WithCustomMapping(t *testing.T) {
	tests := []struct {
		name           string
		problemMapping func(error) Problem
		err            error
		wantStatus     int
		wantTitle      string
	}{
		{
			name: "Should use custom mapping when provided",
			problemMapping: func(err error) Problem {
				return Problem{
					Title:  "Custom Error",
					Status: http.StatusTeapot,
					Type:   "https://example.com/custom",
					Detail: "This is a custom error",
				}
			},
			err:        handlerutil.ErrValidation,
			wantStatus: http.StatusTeapot,
			wantTitle:  "Custom Error",
		},
		{
			name: "Should fallback to standard mapping when custom returns empty",
			problemMapping: func(err error) Problem {
				return Problem{}
			},
			err:        handlerutil.ErrNotFound,
			wantStatus: http.StatusNotFound,
			wantTitle:  "Not Found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hw := NewWithMapping(tt.problemMapping)
			problem := hw.buildProblem(tt.err)

			if problem.Status != tt.wantStatus {
				t.Errorf("buildProblem().Status = %v, want %v", problem.Status, tt.wantStatus)
			}

			if problem.Title != tt.wantTitle {
				t.Errorf("buildProblem().Title = %v, want %v", problem.Title, tt.wantTitle)
			}
		})
	}
}

func TestHttpWriter_WriteError_NilError(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "Should not write response when error is nil",
			err:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			logger, _ := zap.NewDevelopment()
			hw := New()

			hw.WriteError(context.Background(), w, tt.err, logger)

			// Should not have written anything
			if w.Code != http.StatusOK {
				t.Errorf("WriteError() with nil error should not set status, got %v", w.Code)
			}

			if w.Body.Len() > 0 {
				t.Errorf("WriteError() with nil error should not write body, got %v bytes", w.Body.Len())
			}
		})
	}
}

func TestHttpWriter_WriteErrorWithRequest_NilError(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "Should not write response when error is nil",
			err:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
			w := httptest.NewRecorder()
			logger, _ := zap.NewDevelopment()
			hw := New()

			hw.WriteErrorWithRequest(context.Background(), req, w, tt.err, logger)

			// Should not have written anything
			if w.Code != http.StatusOK {
				t.Errorf("WriteErrorWithRequest() with nil error should not set status, got %v", w.Code)
			}

			if w.Body.Len() > 0 {
				t.Errorf("WriteErrorWithRequest() with nil error should not write body, got %v bytes", w.Body.Len())
			}
		})
	}
}

func TestHttpWriter_WriteErrorWithRequest_InstancePath(t *testing.T) {
	tests := []struct {
		name         string
		requestPath  string
		err          error
		wantInstance string
	}{
		{
			name:         "Should extract path from simple request",
			requestPath:  "/api/v1/users",
			err:          handlerutil.ErrValidation,
			wantInstance: "/api/v1/users",
		},
		{
			name:         "Should extract path with ID parameter",
			requestPath:  "/api/v1/users/123",
			err:          handlerutil.ErrNotFound,
			wantInstance: "/api/v1/users/123",
		},
		{
			name:         "Should extract path ignoring query string",
			requestPath:  "/api/v1/search?q=test&limit=10",
			err:          handlerutil.ErrValidation,
			wantInstance: "/api/v1/search",
		},
		{
			name:         "Should extract root path",
			requestPath:  "/",
			err:          handlerutil.ErrNotFound,
			wantInstance: "/",
		},
		{
			name:         "Should handle complex nested path",
			requestPath:  "/api/v1/organizations/123/teams/456/members",
			err:          handlerutil.ErrForbidden,
			wantInstance: "/api/v1/organizations/123/teams/456/members",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.requestPath, nil)
			w := httptest.NewRecorder()
			logger, _ := zap.NewDevelopment()
			hw := New()

			hw.WriteErrorWithRequest(context.Background(), req, w, tt.err, logger)

			var problem Problem
			if err := json.NewDecoder(w.Body).Decode(&problem); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if problem.Instance != tt.wantInstance {
				t.Errorf("WriteErrorWithRequest().Instance = %v, want %v", problem.Instance, tt.wantInstance)
			}
		})
	}
}

func TestHttpWriter_buildProblem_AllErrorTypes(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantTitle  string
	}{
		{
			name:       "Should handle pagination invalid page or size error",
			err:        pagination.ErrInvalidPageOrSize,
			wantStatus: http.StatusBadRequest,
			wantTitle:  "Validation Problem",
		},
		{
			name:       "Should handle pagination invalid sorting field error",
			err:        pagination.ErrInvalidSortingField,
			wantStatus: http.StatusBadRequest,
			wantTitle:  "Validation Problem",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hw := New()
			problem := hw.buildProblem(tt.err)

			if problem.Status != tt.wantStatus {
				t.Errorf("buildProblem().Status = %v, want %v", problem.Status, tt.wantStatus)
			}

			if problem.Title != tt.wantTitle {
				t.Errorf("buildProblem().Title = %v, want %v", problem.Title, tt.wantTitle)
			}
		})
	}
}

func TestNewInternalServerProblem(t *testing.T) {
	tests := []struct {
		name       string
		detail     string
		wantStatus int
		wantTitle  string
		wantType   string
	}{
		{
			name:       "Should create internal server error problem",
			detail:     "Database connection failed",
			wantStatus: http.StatusInternalServerError,
			wantTitle:  "Internal Server Error",
			wantType:   "https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/500",
		},
		{
			name:       "Should create internal server error with empty detail",
			detail:     "",
			wantStatus: http.StatusInternalServerError,
			wantTitle:  "Internal Server Error",
			wantType:   "https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			problem := NewInternalServerProblem(tt.detail)

			if problem.Status != tt.wantStatus {
				t.Errorf("NewInternalServerProblem().Status = %v, want %v", problem.Status, tt.wantStatus)
			}

			if problem.Title != tt.wantTitle {
				t.Errorf("NewInternalServerProblem().Title = %v, want %v", problem.Title, tt.wantTitle)
			}

			if problem.Type != tt.wantType {
				t.Errorf("NewInternalServerProblem().Type = %v, want %v", problem.Type, tt.wantType)
			}

			if problem.Detail != tt.detail {
				t.Errorf("NewInternalServerProblem().Detail = %v, want %v", problem.Detail, tt.detail)
			}
		})
	}
}

func TestNewBadRequestProblem(t *testing.T) {
	tests := []struct {
		name       string
		detail     string
		wantStatus int
		wantTitle  string
		wantType   string
	}{
		{
			name:       "Should create bad request problem",
			detail:     "Invalid request format",
			wantStatus: http.StatusBadRequest,
			wantTitle:  "Bad Request",
			wantType:   "https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/400",
		},
		{
			name:       "Should create bad request problem with empty detail",
			detail:     "",
			wantStatus: http.StatusBadRequest,
			wantTitle:  "Bad Request",
			wantType:   "https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			problem := NewBadRequestProblem(tt.detail)

			if problem.Status != tt.wantStatus {
				t.Errorf("NewBadRequestProblem().Status = %v, want %v", problem.Status, tt.wantStatus)
			}

			if problem.Title != tt.wantTitle {
				t.Errorf("NewBadRequestProblem().Title = %v, want %v", problem.Title, tt.wantTitle)
			}

			if problem.Type != tt.wantType {
				t.Errorf("NewBadRequestProblem().Type = %v, want %v", problem.Type, tt.wantType)
			}

			if problem.Detail != tt.detail {
				t.Errorf("NewBadRequestProblem().Detail = %v, want %v", problem.Detail, tt.detail)
			}
		})
	}
}
