package problem

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	handlerutil "github.com/NYCU-SDC/summer/pkg/handler"
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
			name: "Should handle ValidationError correctly",
			setupError: func() error {
				return handlerutil.NewValidationError("email", "test", "invalid email")
			},
			wantStatus: http.StatusBadRequest,
			wantTitle:  "Validation Problem",
		},
		{
			name: "Should handle NotFoundError correctly",
			setupError: func() error {
				return handlerutil.NewNotFoundError("users", "id", "123", "")
			},
			wantStatus: http.StatusNotFound,
			wantTitle:  "Not Found",
		},
		{
			name: "Should handle generic validation error",
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
