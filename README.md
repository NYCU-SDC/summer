# summer

A Go toolkit for building RESTful services at NYCU SDC. Provides a CLI scaffolder to bootstrap new projects and a set of opinionated packages for structured logging, request handling, database access, tracing, and validation.

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
    - [CLI: Scaffold a new project](#cli-scaffold-a-new-project)
    - [Run the example](#run-the-example)
- [Packages](#packages)
    - [pkg/log](#pkglog)
    - [pkg/handler](#pkghandler)
    - [pkg/problem](#pkgproblem)
    - [pkg/middleware](#pkgmiddleware)
    - [pkg/trace](#pkgtrace)
    - [pkg/cors](#pkgcors)
    - [pkg/database](#pkgdatabase)
    - [pkg/pagination](#pkgpagination)
    - [pkg/config](#pkgconfig)
- [Wiring Everything Together](#wiring-everything-together)
- [Project Layout](#project-layout)
- [sqlc Integration](#sqlc-integration)
- [Contributing](#contributing)
- [License](#license)

---

## Features

- **CLI scaffolder** — `summer init` creates a ready-to-run project with a health endpoint
- **Structured logging** — Zap-based logger wired for JSON output in production and pretty console output in development, with automatic trace/span ID injection
- **Tracing** — OpenTelemetry tracing middleware with upstream context propagation
- **Database** — pgx + golang-migrate helpers for PostgreSQL; MSSQL also supported
- **Validation** — `go-playground/validator` wrappers with consistent RFC 9457 problem-detail error responses
- **Middleware set** — composable middleware chain builder compatible with `net/http`
- **Pagination** — generic, type-safe paginated list helper

---

## Installation

### CLI tool

```bash
go install github.com/NYCU-SDC/summer/cmd/summer@latest
summer -v   # verify
```

### Library packages

```bash
go get github.com/NYCU-SDC/summer
```

Requires Go 1.24+.

---

## Quick Start

### CLI: Scaffold a new project

```bash
summer -b main init
```

summer will ask for a project name (used as the Go module name in `go.mod`). It creates:
```
.
├── cmd/
│   └── main.go          # minimal server with /healthz endpoint
├── internal/
└── scripts/
    └── create_full_schema.sh
```

### Run the example

```bash
go mod tidy
go run ./cmd/main.go
```

Hit the health endpoint:

```bash
curl localhost:8080/healthz
```

---

## Packages

### pkg/log

**Import path:** `github.com/NYCU-SDC/summer/pkg/log`  
**Package name:** `logutil`

Wraps `go.uber.org/zap` with two pre-configured profiles and a context-aware logger decorator.

#### Logger configs

`ZapProductionConfig()` returns a JSON logger at Info level (no sampling).  
`ZapDevelopmentConfig()` returns a color console logger at Debug level with GoLand-clickable caller links.

```go
import logutil "github.com/NYCU-SDC/summer/pkg/log"

// In production
logger, err := logutil.ZapProductionConfig().Build()

// In development / debug mode
logger, err := logutil.ZapDevelopmentConfig().Build()
```

#### WithContext

`WithContext` enriches a logger with fields extracted from the request context: OpenTelemetry `trace_id` / `span_id`, and user fields (`user_id`, `username`, `name`) if present.

```go
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (User, error) {
    logger := logutil.WithContext(ctx, s.logger)
    // logger now carries trace_id, span_id, user_id, etc.
    user, err := s.queries.GetByID(ctx, id)
    if err != nil {
        logger.Error("query failed", zap.Error(err))
        return User{}, err
    }
    return user, nil
}
```

---

### pkg/handler

**Import path:** `github.com/NYCU-SDC/summer/pkg/handler`  
**Package name:** `handlerutil`

Provides shared error sentinel values, structured error types, and HTTP utility functions for handlers.

#### Sentinel errors

```go
var (
    ErrNotFound          = errors.New("record not found")
    ErrForbidden         = errors.New("forbidden")
    ErrCredentialInvalid = errors.New("invalid username or password")
    ErrUserAlreadyExists = errors.New("user already exists")
    ErrUnauthorized      = errors.New("unauthorized")
    ErrInternalServer    = errors.New("internal server error")
    ErrInvalidUUID       = errors.New("failed to parse UUID")
    ErrValidation        = errors.New("validation error")
)
```

#### Structured error types

`NotFoundError` and `ValidationError` carry extra context and implement `errors.Is` against the corresponding sentinel:

```go
// Return a not-found error with table/key/value context
return handlerutil.NewNotFoundError("users", "id", id.String(), "")

// Return a validation error with a message
return handlerutil.NewValidationError("email", req.Email, "must be a valid email")

// Return a validation error with multiple field-level messages
return handlerutil.NewValidationErrorWithErrors("invalid request", []string{"field A", "field B"})
```

#### ParseAndValidateRequestBody

Reads the request body, unmarshals JSON into `s`, and runs `go-playground/validator` struct validation. Returns a `ValidationError` on JSON parse failure or validation failure.

```go
var req CreateUserRequest
if err := handlerutil.ParseAndValidateRequestBody(ctx, h.validator, r, &req); err != nil {
    h.problemWriter.WriteError(ctx, w, err, logger)
    return
}
```

#### WriteJSONResponse

Sets `Content-Type: application/json`, writes the status code, and marshals `data` as JSON.

```go
handlerutil.WriteJSONResponse(w, http.StatusOK, Response{ID: user.ID, Email: user.Email})
```

#### ParseUUID

Parses a URL path parameter (or any string) as a UUID. Wraps parse errors as `ErrInvalidUUID`.

```go
idStr := r.PathValue("user_id")
id, err := handlerutil.ParseUUID(idStr)
if err != nil {
    h.problemWriter.WriteError(ctx, w, handlerutil.ErrInvalidUUID, logger)
    return
}
```

---

### pkg/problem

**Import path:** `github.com/NYCU-SDC/summer/pkg/problem`  
**Package name:** `problem`

Implements [RFC 9457](https://www.rfc-editor.org/rfc/rfc9457) Problem Details for HTTP APIs. Converts Go errors into structured JSON error responses with `Content-Type: application/problem+json`.

#### HttpWriter

`HttpWriter` maps errors to `Problem` structs and writes them to the response. It handles all standard errors from `pkg/handler`, `pkg/database`, and `pkg/pagination` automatically.

```go
// Create a writer with no custom mapping
writer := problem.New()

// Create a writer with a custom mapping for application-specific errors
writer := problem.NewWithMapping(func(err error) problem.Problem {
    if errors.Is(err, myapp.ErrQuotaExceeded) {
        return problem.Problem{
            Title:  "Quota Exceeded",
            Status: http.StatusTooManyRequests,
            Type:   "https://example.com/errors/quota-exceeded",
            Detail: err.Error(),
        }
    }
    return problem.Problem{} // return empty to fall through to default handling
})
```

#### WriteError / WriteErrorWithRequest

```go
// Write error without request context
writer.WriteError(ctx, w, err, logger)

// Write error and populate the `instance` field with the request path
writer.WriteErrorWithRequest(ctx, r, w, err, logger)
```

#### Error-to-HTTP mapping (automatic)

| Error | HTTP Status |
|---|---|
| `handlerutil.NotFoundError` / `ErrNotFound` | 404 Not Found |
| `handlerutil.ValidationError` / `validator.ValidationErrors` / `ErrValidation` | 400 Bad Request |
| `handlerutil.ErrUnauthorized` / `ErrCredentialInvalid` | 401 Unauthorized |
| `handlerutil.ErrForbidden` | 403 Forbidden |
| `handlerutil.ErrUserAlreadyExists` / `ErrInvalidUUID` | 400 Bad Request |
| `databaseutil.InternalServerError` | 500 Internal Server Error |
| `pagination.ErrInvalidPageOrSize` / `ErrInvalidSortingField` | 400 Bad Request |
| anything else | 500 Internal Server Error |

#### Problem constructors

Pre-built constructors for common responses:

```go
problem.NewInternalServerProblem("something went wrong")
problem.NewNotFoundProblem("user not found")
problem.NewValidateProblem("invalid request body")
problem.NewValidateProblemWithErrors("validation failed", []string{"name: required", "email: invalid"})
problem.NewUnauthorizedProblem("you must be logged in")
problem.NewForbiddenProblem("insufficient permissions")
problem.NewBadRequestProblem("malformed request")
```

---

### pkg/middleware

**Import path:** `github.com/NYCU-SDC/summer/pkg/middleware`  
**Package name:** `middleware`

Provides a composable middleware `Set` for `net/http` `HandlerFunc` chains.

#### Building a middleware set

```go
import "github.com/NYCU-SDC/summer/pkg/middleware"

// Create a set from one or more middlewares
basicMiddleware := middleware.NewSet(traceMiddleware.RecoverMiddleware)
basicMiddleware = basicMiddleware.Append(traceMiddleware.TraceMiddleWare)

// Append does NOT modify the original — it returns a new Set
authMiddleware := basicMiddleware.Append(jwtMiddleware.HandlerFunc)
authMiddleware = authMiddleware.Append(roleMiddleware.HandlerFunc)
```

#### Applying to a handler

`HandlerFunc` wraps a `http.HandlerFunc` with all middlewares in the set, applied in the order they were appended:

```go
mux.HandleFunc("GET /api/users/me", authMiddleware.HandlerFunc(userHandler.GetMeHandler))
```

---

### pkg/trace

**Import path:** `github.com/NYCU-SDC/summer/pkg/trace`  
**Package name:** `traceutil`

OpenTelemetry tracing middleware and panic recovery.

#### TraceMiddleware

Creates an OTel span for each request, propagates upstream trace context from headers, enriches the logger with trace/span IDs, and logs request completion at the appropriate level (`Info` for 2xx/3xx, `Error` for 4xx/5xx).

When `debug` is `true`, it buffers the request and response bodies and includes them in error logs for 5xx responses. **Avoid enabling debug mode for endpoints that handle large payloads (e.g. file uploads).**

```go
func TraceMiddleware(next http.HandlerFunc, logger *zap.Logger, debug bool) http.HandlerFunc
```

#### RecoverMiddleware

Catches panics in downstream handlers, logs the stack trace, and responds with `500 Internal Server Error`.

```go
func RecoverMiddleware(next http.HandlerFunc, logger *zap.Logger, debug bool) http.HandlerFunc
```

Always place `RecoverMiddleware` before `TraceMiddleware` in the chain so panics are captured inside the trace span:

```go
mw := middleware.NewSet(recoverMw).Append(traceMw)
```

#### PanicRecoveryError

A helper that unpacks `recover()` output into a `(needsRecovery bool, errString string, callers []string)` tuple. Used internally by `RecoverMiddleware`.

---

### pkg/cors

**Import path:** `github.com/NYCU-SDC/summer/pkg/cors`  
**Package name:** `cors`

A single `CORSMiddleware` function that handles `Origin` validation, preflight `OPTIONS` requests, and response headers.

```go
func CORSMiddleware(next http.HandlerFunc, logger *zap.Logger, allowOrigin []string) http.HandlerFunc
```

Pass `"*"` in `allowOrigin` to allow all origins, or list specific origins. Requests from unlisted origins receive `403 Forbidden`.

```go
// Applied at the outermost layer, wrapping the entire mux
entrypoint := cors.CORSMiddleware(mux.ServeHTTP, logger, cfg.AllowOrigins)
srv := &http.Server{Handler: entrypoint}
```

---

### pkg/database

**Import path:** `github.com/NYCU-SDC/summer/pkg/database`  
**Package name:** `databaseutil`

Helpers for PostgreSQL (pgx) and MSSQL: schema migrations and error wrapping.

#### Migrations

```go
// Apply all pending migrations
err := databaseutil.MigrationUp(sourceURL, databaseURL, logger)
// sourceURL example: "file://migrations"
// databaseURL example: "postgres://user:pass@localhost:5432/mydb?sslmode=disable"

// Roll back all applied migrations
err := databaseutil.MigrationDown(sourceURL, databaseURL, logger)
```

`MigrationUp` is idempotent — it logs a message and returns `nil` if the schema is already up to date.

#### PostgreSQL error wrapping

Both functions log the original error, classify it into a well-known type, and return a wrapped error for consistent handling in `pkg/problem`.

```go
// Generic wrap — maps pgx.ErrNoRows to ErrNotFound
err = databaseutil.WrapDBError(err, logger, "get user by id")

// Richer wrap — maps pgx.ErrNoRows to NotFoundError{Table, Key, Value}
err = databaseutil.WrapDBErrorWithKeyValue(err, "users", "id", id.String(), logger, "get user by id")
```

Mapped error types:

| Database error | Wrapped as |
|---|---|
| `pgx.ErrNoRows` | `handlerutil.ErrNotFound` or `NotFoundError` |
| `context.DeadlineExceeded` | `ErrQueryTimeout` |
| PG code `23505` | `ErrUniqueViolation` |
| PG code `23503` | `ErrForeignKeyViolation` |
| PG code `40P01` | `ErrDeadlockDetected` |
| anything else | `InternalServerError{Source: err}` |

#### MSSQL error wrapping

Same API, same mapped error types, for Microsoft SQL Server:

```go
err = databaseutil.WrapMSSQLError(err, logger, "create record")
err = databaseutil.WrapMSSQLErrorWithKeyValue(err, "users", "id", id.String(), logger, "get user")
```

---

### pkg/pagination

**Import path:** `github.com/NYCU-SDC/summer/pkg/pagination`  
**Package name:** `pagination`

Generic, type-safe helpers for offset-based paginated list endpoints.

#### Factory

`Factory[T]` is created once per handler or resource with a maximum page size and the list of allowed sort columns:

```go
factory := pagination.NewFactory[UserResponse](200, []string{"studentId", "fullName", "email"})
```

#### GetRequest

Parses `page`, `size`, `sort`, and `sortBy` query parameters. Returns `ErrInvalidPageOrSize` if `size` exceeds the maximum, and `ErrInvalidSortingField` if `sortBy` is not in the allowed list and a sort direction is specified.

```go
pageRequest, err := factory.GetRequest(r)
if err != nil {
    h.problemWriter.WriteError(ctx, w, err, logger)
    return
}
// pageRequest.Page, pageRequest.Size, pageRequest.Sort, pageRequest.SortBy
```

#### NewResponse

Builds a `Response[T]` with pagination metadata:

```go
items, totalCount, err := store.ListUsers(ctx, pageRequest.Page, pageRequest.Size)
// ...
pageResponse := factory.NewResponse(items, totalCount, pageRequest.Page, pageRequest.Size)
handlerutil.WriteJSONResponse(w, http.StatusOK, pageResponse)
```

JSON response shape:

```json
{
  "items": [...],
  "totalPages": 5,
  "totalItems": 42,
  "currentPage": 1,
  "pageSize": 10,
  "hasNextPage": true
}
```

---

### pkg/config

**Import path:** `github.com/NYCU-SDC/summer/pkg/config`  
**Package name:** `configutil`

A single generic function for merging configuration structs. Any field in `override` that is non-zero overwrites the corresponding field in `base`. Zero-value and empty-slice fields in `override` are ignored, preserving `base` defaults.

```go
func Merge[T any](base *T, override *T) (*T, error)
```

```go
type Config struct {
    Host  string
    Port  int
    Debug bool
}

base := &Config{Host: "0.0.0.0", Port: 8080, Debug: false}
override := &Config{Port: 9090} // only override Port

merged, err := configutil.Merge(base, override)
// merged: {Host: "0.0.0.0", Port: 9090, Debug: false}
```

---

## Wiring Everything Together

The following sketch shows how all packages connect in a typical service:

```go
package main

import (
    databaseutil "github.com/NYCU-SDC/summer/pkg/database"
    logutil      "github.com/NYCU-SDC/summer/pkg/log"
    "github.com/NYCU-SDC/summer/pkg/middleware"
    traceutil    "github.com/NYCU-SDC/summer/pkg/trace"
    "github.com/NYCU-SDC/summer/pkg/cors"
    "github.com/NYCU-SDC/summer/pkg/problem"
    handlerutil  "github.com/NYCU-SDC/summer/pkg/handler"
    "github.com/NYCU-SDC/summer/pkg/pagination"
)

func main() {
    // 1. Logger
    logger, _ := logutil.ZapProductionConfig().Build()

    // 2. Database migration
    databaseutil.MigrationUp("file://migrations", os.Getenv("DATABASE_URL"), logger)

    // 3. Problem writer (shared across all handlers)
    problemWriter := problem.NewWithMapping(myAppErrorMapping)

    // 4. Middleware chain
    traceMw := func(next http.HandlerFunc) http.HandlerFunc {
        return traceutil.TraceMiddleware(next, logger, false)
    }
    recoverMw := func(next http.HandlerFunc) http.HandlerFunc {
        return traceutil.RecoverMiddleware(next, logger, false)
    }
    corsMw := func(next http.HandlerFunc) http.HandlerFunc {
        return cors.CORSMiddleware(next, logger, []string{"https://example.com"})
    }

    base := middleware.NewSet(recoverMw).Append(traceMw)

    // 5. Routes
    mux := http.NewServeMux()
    mux.HandleFunc("GET /api/users", base.HandlerFunc(listUsersHandler))

    // 6. CORS wraps the whole mux
    http.ListenAndServe(":8080", corsMw(mux.ServeHTTP))
}

// 7. In a handler — parse, validate, paginate, respond
func listUsersHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    logger := logutil.WithContext(ctx, baseLogger)

    factory := pagination.NewFactory[UserResponse](100, []string{"email", "name"})
    pageReq, err := factory.GetRequest(r)
    if err != nil {
        problemWriter.WriteError(ctx, w, err, logger)
        return
    }

    items, total, err := userStore.List(ctx, pageReq.Page, pageReq.Size)
    if err != nil {
        // WrapDBError maps pgx errors → sentinel errors → problem responses
        err = databaseutil.WrapDBError(err, logger, "list users")
        problemWriter.WriteError(ctx, w, err, logger)
        return
    }

    handlerutil.WriteJSONResponse(w, http.StatusOK, factory.NewResponse(items, total, pageReq.Page, pageReq.Size))
}
```

---

## Project Layout

When you use `summer init`, your service should follow this layout:
```
.
├── cmd/
│   └── main.go           # entry point — wire everything together here
├── internal/
│   ├── <domain>/         # one directory per resource (user, group, …)
│   │   ├── handler.go    # HTTP handlers using pkg/handler and pkg/problem
│   │   ├── service.go    # business logic using pkg/log and pkg/database
│   │   ├── query.sql.go  # generated by sqlc
│   │   └── schema.sql    # collected by create_full_schema.sh
│   └── database/
│       └── full_schema.sql
├── migrations/           # golang-migrate SQL files
├── scripts/
│   └── create_full_schema.sh
├── go.mod
└── go.sum
```

---

## sqlc Integration

`create_full_schema.sh` collects every `schema.sql` file under `internal/` and merges them into `internal/database/full_schema.sql`, which you can point sqlc at.

Run from the project root:

```bash
./scripts/create_full_schema.sh
```

If you get `permission denied`, make the script executable first:

```bash
chmod +x ./scripts/create_full_schema.sh
```

---

## Contributing

Open a PR against `main`. Please run `go test ./...` and `go vet ./...` before submitting.

---

## License

MIT — see [LICENSE](LICENSE).
