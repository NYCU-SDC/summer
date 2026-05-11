# summer

A Go toolkit for building RESTful services at NYCU SDC. Provides a CLI scaffolder to bootstrap new projects and a set of opinionated packages for structured logging, request handling, database access, tracing, and validation.

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
    - [CLI: Scaffold a new project](#cli-scaffold-a-new-project)
    - [Run the example](#run-the-example)
- [Packages](#packages)
    - [pkg/handler](#pkghandler)
    - [pkg/logger](#pkglogger)
    - [pkg/middleware](#pkgmiddleware)
    - [pkg/database](#pkgdatabase)
    - [pkg/response](#pkgresponse)
- [Project Layout](#project-layout)
- [sqlc Integration](#sqlc-integration)
- [Contributing](#contributing)
- [License](#license)

---

## Features

- **CLI scaffolder** — `summer init` creates a ready-to-run project with a health endpoint
- **Structured logging** — Zap-based logger wired for JSON output in production
- **Tracing** — OpenTelemetry integration out of the box
- **Database** — pgx + golang-migrate helpers for PostgreSQL; MSSQL supported
- **Validation** — `go-playground/validator` wrappers with consistent error responses
- **Request ID & middleware** — trace-ID propagation, recovery, and CORS helpers

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

### pkg/handler

Provides base handler utilities for consistent request parsing and error propagation.

```go
import "github.com/NYCU-SDC/summer/pkg/handler"

// TODO: add snippet once example/main.go is reviewed
```

### pkg/logger

Wraps `go.uber.org/zap` with sensible defaults. In development mode it emits human-readable output; in production it emits JSON.

```go
import "github.com/NYCU-SDC/summer/pkg/logger"

log := logger.New()
log.Info("server starting", zap.Int("port", 8080))
```

### pkg/middleware

Standard HTTP middleware compatible with `net/http`:

- **RequestID** — attaches a UUID to every request and propagates it via context and response headers
- **Logger** — structured per-request logging (method, path, status, latency)
- **Recovery** — catches panics and returns 500 without crashing the server
- **CORS** — configurable cross-origin headers

```go
import "github.com/NYCU-SDC/summer/pkg/middleware"

mux := http.NewServeMux()
handler := middleware.Chain(mux,
    middleware.RequestID,
    middleware.Logger(log),
    middleware.Recovery(log),
)
```

### pkg/database

Helpers for connecting to PostgreSQL via `pgx/v5` and running migrations with `golang-migrate`.

```go
import "github.com/NYCU-SDC/summer/pkg/database"

db, err := database.Connect(ctx, os.Getenv("DATABASE_URL"))
if err != nil {
    log.Fatal("db connect failed", zap.Error(err))
}
defer db.Close()

if err := database.Migrate(db, "migrations/"); err != nil {
    log.Fatal("migration failed", zap.Error(err))
}
```

### pkg/response

Standardised JSON response helpers to keep API responses consistent across handlers.

```go
import "github.com/NYCU-SDC/summer/pkg/response"

func GetUser(w http.ResponseWriter, r *http.Request) {
    user, err := store.Find(r.Context(), id)
    if err != nil {
        response.Error(w, http.StatusNotFound, "user not found")
        return
    }
    response.JSON(w, http.StatusOK, user)
}
```

---

## Project Layout

When you use `summer init`, your service should follow this layout:
```
.
├── cmd/
│   └── main.go          # entry point — wire everything together here
├── internal/
│   ├── <domain>/        # one directory per resource (user, post, …)
│   │   ├── handler.go
│   │   ├── store.go
│   │   └── schema.sql   # collected by create_full_schema.sh
│   └── database/
│       └── full_schema.sql
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