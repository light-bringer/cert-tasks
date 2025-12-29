# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A simple RESTful API for task management built with Go. The project uses an in-memory repository and follows clean architecture principles with clear separation between handlers, models, and repository layers.

**IMPORTANT**: Despite the repository name "cert-tasks", this is a **task management API**, not a certificate management system. The name is a misnomer. This is an intentionally simple implementation (2-3 hour exercise) focusing on clean code and testability.

**Module**: `github.com/light-bringer/cert-tasks`
**Go Version**: 1.25.5
**Router**: chi v5.2.3
**Default Port**: 8080 (configurable via PORT env var)

## Common Development Commands

### Building and Running

```bash
# Build the application
make build                    # Outputs to bin/api
go build -o bin/api ./cmd/api

# Run the application
make run                      # Build and run
make run-dev                  # Run without building (go run)
./bin/api                     # Run built binary
PORT=3000 ./bin/api          # Run on custom port
```

### Testing

```bash
# Run all tests
make test
go test ./...
go test -v ./...             # Verbose output

# Run specific package tests
go test ./internal/repository/...
go test ./internal/handlers/...

# Run a specific test
go test -v -run TestCreateTask ./internal/handlers/

# Test with coverage
make test-coverage           # Generates coverage.html
go test -cover ./...

# Test with race detector (IMPORTANT for concurrent code)
make test-race
go test -race ./...

# Run Go integration tests (recommended)
make test-integration            # Requires server running on :8080
make test-integration-standalone # Starts server, runs tests, stops server
make test-all                    # Run unit + integration tests

# Run Python API integration tests (DEPRECATED - use Go version above)
make test-api                    # Runs Python test script
python3 test_api.py              # Summary table output
python3 test_api.py -v           # Verbose mode
```

### Code Quality

```bash
# Format code
make fmt                     # Runs gofmt and go mod tidy
go fmt ./...

# Run go vet
make vet
go vet ./...

# Run linter (requires golangci-lint)
make lint
golangci-lint run

# Run all checks (fmt + vet + lint + test)
make check
```

### Cleanup and Dependencies

```bash
# Clean build artifacts
make clean                   # Removes bin/, coverage files, etc.

# Manage dependencies
make install-deps            # Download and verify dependencies
go mod tidy                  # Clean up go.mod/go.sum
go mod download              # Download dependencies
go mod verify                # Verify dependencies
```

### Docker

```bash
# Build Docker image
docker build -t cert-tasks:latest .
make docker-build            # Same as above

# Run Docker container
docker run -p 8080:8080 cert-tasks:latest
docker run -p 3000:8080 -e PORT=8080 cert-tasks:latest  # Custom host port
make docker-run              # Runs on port 8080

# Using Docker Compose (recommended for local development)
docker-compose up            # Start services (API + healthcheck sidecar)
docker-compose up -d         # Start in detached mode
docker-compose down          # Stop and remove containers
docker-compose logs -f api   # Follow API logs
docker-compose logs -f healthcheck  # Follow health check logs

# Test the API in Docker
curl http://localhost:8080/tasks
```

## Architecture

### Project Structure

```
cert-tasks/
├── cmd/
│   └── api/
│       └── main.go              # Entry point: wires dependencies, starts server
├── internal/
│   ├── handlers/                # HTTP request handlers
│   │   ├── task_handler.go     # Task CRUD handlers
│   │   └── task_handler_test.go
│   ├── models/                  # Domain models and DTOs
│   │   └── task.go             # Task model, validation
│   ├── repository/              # Data access layer
│   │   ├── task_repository.go  # Repository interface
│   │   ├── memory_repository.go # In-memory implementation
│   │   └── memory_repository_test.go
│   └── server/                  # Server setup and routing
│       └── server.go           # Chi router setup, middleware
├── test/                        # Integration tests
│   └── integration_test.go     # Go integration tests (recommended)
├── Dockerfile                   # Multi-stage Docker build
├── .dockerignore                # Docker build exclusions
├── docker-compose.yml           # Local development setup
├── test_api.py                  # Python integration tests (deprecated)
├── Makefile
├── go.mod
└── README.md
```

### Layer Responsibilities

**cmd/api/main.go**: Application entry point. Minimal logic - creates repository, handler, server, and starts listening.

**internal/handlers**: HTTP handlers that:
- Parse and validate requests
- Call repository methods
- Return appropriate HTTP responses and status codes
- Handle errors with proper error responses

**internal/models**: Domain models with validation:
- `Task`: Core entity with ID, Title, Description, Status, timestamps
- `CreateTaskRequest` / `UpdateTaskRequest`: DTOs with validation methods
- `TaskStatus`: Enum type ("todo" | "done")

**internal/repository**: Data access abstraction:
- `TaskRepository`: Interface defining CRUD operations
- `MemoryRepository`: Thread-safe in-memory implementation using sync.RWMutex

**internal/server**: Server configuration:
- Chi router setup
- Middleware: logging, recovery, content-type headers
- Route registration
- Graceful shutdown handling

### Key Design Patterns

1. **Repository Pattern**: Data access abstracted behind `TaskRepository` interface
   - Enables testing with mock implementations
   - Allows swapping storage backends (could add PostgreSQL, Redis, etc.)

2. **Thread Safety**: `MemoryRepository` uses `sync.RWMutex` for concurrent access
   - Multiple readers OR single writer
   - Critical for production correctness

3. **Clean Separation**: Handlers don't know about storage implementation
   - Handlers depend on `TaskRepository` interface, not concrete types
   - Follows Dependency Inversion Principle

4. **Request Validation**: Validation in model layer via `Validate()` methods
   - Title cannot be empty/whitespace
   - Status must be "todo" or "done"

## API Endpoints

All endpoints use JSON request/response bodies.

```
POST   /tasks        Create a task (requires title)
GET    /tasks        List all tasks
GET    /tasks/{id}   Get a specific task
PUT    /tasks/{id}   Update a task (full replacement)
DELETE /tasks/{id}   Delete a task
```

### Task Model

```go
type Task struct {
    ID          int64      `json:"id"`           // Auto-generated
    Title       string     `json:"title"`        // Required, non-empty
    Description string     `json:"description"`  // Optional
    Status      TaskStatus `json:"status"`       // "todo" or "done"
    CreatedAt   time.Time  `json:"created_at"`   // Auto-set
    UpdatedAt   time.Time  `json:"updated_at"`   // Auto-updated
}
```

### HTTP Status Codes

- `200 OK` - Successful GET or PUT
- `201 Created` - Successful POST
- `204 No Content` - Successful DELETE
- `400 Bad Request` - Validation error, invalid ID format, malformed JSON
- `404 Not Found` - Task not found
- `500 Internal Server Error` - Unexpected server error

## Testing Strategy

### Unit Tests

Tests are colocated with implementation files (`*_test.go`).

**Handlers** (`internal/handlers/task_handler_test.go`):
- Uses httptest for HTTP testing
- Tests all endpoints with various scenarios
- Validates status codes, response bodies, error handling

**Repository** (`internal/repository/memory_repository_test.go`):
- Tests CRUD operations
- Tests concurrent access (race conditions)
- Tests edge cases (not found, updates, etc.)

**Models** (`internal/models/task.go`):
- Validation logic tested inline

### Integration Tests

**Go integration tests** (`test/integration_test.go`) - **RECOMMENDED**:
- Native Go end-to-end API testing with tabular output
- Located in `test/` directory (separate from unit tests)
- Tests all endpoints: CREATE, LIST, GET, UPDATE, DELETE
- Comprehensive validation testing (missing fields, invalid data, etc.)
- **21 total test cases** covering happy paths and error scenarios
- Displays results in formatted table with statistics
- Exit code 0 on success, 1 on failure (CI/CD friendly)
- Uses standard Go testing framework

**Running integration tests**:
```bash
make test-integration            # Requires server already running on :8080
make test-integration-standalone # Starts server, runs tests, stops server automatically
make test-all                    # Runs unit tests + integration tests
```

**Example output**:
```
============================================================
🚀 TASK MANAGEMENT API - INTEGRATION TEST SUITE
============================================================
Base URL: http://localhost:8080
✅ Server is reachable

#    | Category   | Test Name                      | Method | Endpoint        | Status     | Code       | Time(ms)
-------------------------------------------------------------------------------------------------------------------------
1    | CREATE     | Valid task with description    | POST   | /tasks          | ✅ PASS    | 201/201    | 12.34
2    | CREATE     | Missing title (validation)     | POST   | /tasks          | ✅ PASS    | 400/400    | 5.67
...

📊 STATISTICS
  Total Tests          21
  Passed               ✅ 21
  Failed               ❌ 0
  Success Rate         100.0%
  Total Duration       125.45ms
  Average Duration     5.97ms

✅ ALL TESTS PASSED! 🎉
```

**Python script** (`test_api.py`) - **DEPRECATED**:
- Original Python-based integration tests
- Still available but Go version is recommended
- Requires Python 3 and `requests` library
- Optional `tabulate` library for enhanced formatting

Run with: `make test-api` (deprecated, use `make test-integration` instead)

## Implementation Details

### In-Memory Storage

Data is stored in memory using a map with mutex protection:
- Data persists only while server is running
- All operations are thread-safe
- IDs auto-increment from 1

### Graceful Shutdown

Server handles `SIGINT` and `SIGTERM` signals:
- Waits up to 10 seconds for in-flight requests
- Logs shutdown progress
- Prevents data corruption from abrupt termination

### Middleware Stack

Chi middleware in order:
1. `middleware.Logger` - Logs all requests
2. `middleware.Recoverer` - Recovers from panics, returns 500
3. `SetHeader("Content-Type", "application/json")` - Sets JSON content type

### Docker Setup

**Multi-stage build**:
- Build stage: Uses `golang:1.25.5-alpine` to compile the binary
- Runtime stage: Uses `gcr.io/distroless/static-debian12:nonroot` for maximum security
- Binary is statically linked (CGO_ENABLED=0) for portability
- Final image size: ~5-8MB (even smaller than Alpine!)

**Security features (Distroless)**:
- **No shell**: Eliminates entire class of shell-based attacks
- **No package manager**: Cannot install malicious packages
- **Minimal dependencies**: Only contains the application binary and CA certificates
- **Non-root user**: Runs as user "nonroot" (uid/gid 65532)
- **Reproducible builds**: Uses `-trimpath` flag to remove file paths
- **Smallest attack surface**: Drastically reduced compared to traditional base images

**Health checks**:
- Distroless images don't support HEALTHCHECK directive (no shell)
- **Docker Compose solution**: Includes a busybox sidecar container that monitors API health
  - Runs `wget` checks every 30 seconds against the API
  - Logs health status with timestamps
  - Lightweight (~2MB busybox image)
  - View logs: `docker-compose logs -f healthcheck`
- For Kubernetes: Use httpGet probes (recommended):
  ```yaml
  livenessProbe:
    httpGet:
      path: /tasks
      port: 8080
    initialDelaySeconds: 5
    periodSeconds: 10
  ```
- For Docker Swarm: Use external health monitoring

**docker-compose.yml**:
- Ready for local development
- Includes healthcheck sidecar container (workaround for distroless)
- Includes commented PostgreSQL service for future use
- Automatic restart unless stopped

**Why Distroless?**
- Google-maintained, security-focused base images
- Contains only application and runtime dependencies
- Reduces CVE exposure (fewer packages = fewer vulnerabilities)
- Meets compliance requirements for minimal container images
- Ideal for production deployments

**Debugging Distroless Containers**:
Since distroless has no shell, traditional debugging is different:

```bash
# View logs (standard approach)
docker logs <container-id>
docker-compose logs -f api

# Cannot exec into container (no shell available)
# docker exec -it <container-id> /bin/sh  # This won't work!

# For debugging, temporarily use debug variant with busybox shell
# Change FROM line to:
# FROM gcr.io/distroless/static-debian12:debug-nonroot
# Then you can: docker exec -it <container-id> /busybox/sh

# Check if container is running and healthy
curl http://localhost:8080/tasks

# Inspect container
docker inspect <container-id>
```

## Common Patterns to Follow

### Error Handling

```go
// Repository layer: return domain errors
var ErrTaskNotFound = errors.New("task not found")

// Handler layer: convert to HTTP responses
if errors.Is(err, repository.ErrTaskNotFound) {
    http.Error(w, `{"error":"task not found"}`, http.StatusNotFound)
    return
}
```

### Request Validation

```go
// Parse request
var req models.CreateTaskRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
    return
}

// Validate
if err := req.Validate(); err != nil {
    http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
    return
}
```

### Thread-Safe Repository Operations

```go
// Read operations use RLock
func (r *MemoryRepository) GetByID(id int64) (*models.Task, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    // ... read operation
}

// Write operations use Lock
func (r *MemoryRepository) Create(task *models.Task) (*models.Task, error) {
    r.mu.Lock()
    defer r.mu.Unlock()
    // ... write operation
}
```

## Extending the Project

### Adding a New Endpoint

1. Add handler method to `internal/handlers/task_handler.go`
2. Register route in `internal/server/server.go`
3. Add tests in `internal/handlers/task_handler_test.go`

### Switching to Persistent Storage

1. Implement `TaskRepository` interface (e.g., `PostgresRepository`)
2. Update `cmd/api/main.go` to instantiate new repository
3. No changes needed to handlers (they depend on interface)

### Adding Validation Rules

1. Update validation in `internal/models/task.go`
2. Add test cases for new validation rules
3. Update API documentation if needed

## Development Workflow

1. Make changes to code
2. Run `make fmt` to format
3. Run `make test` to verify tests pass
4. Run `make lint` if available (requires golangci-lint)
5. Build and test locally: `make run-dev`
6. Commit changes

## Important Notes

### About the ARCHITECTURE.md File

The `docs/ARCHITECTURE.md` file describes a **future vision** for an enterprise-grade certificate management system with extensive features (certificate lifecycle, task queues, audit logging, etc.). **This is NOT the current implementation**.

The current codebase is an **intentionally simple** task management API (2-3 hour coding exercise) with:
- Basic CRUD operations for tasks
- In-memory storage only
- No authentication
- No certificate functionality

**When working on this codebase**: Follow the simple patterns already established. Do not implement the complex architecture from ARCHITECTURE.md unless explicitly requested. The project philosophy is "simple, clean, and testable" over "enterprise-ready and feature-rich."

### Current Limitations

- **Name Confusion**: Repository is called "cert-tasks" but implements task management, not certificate management
- **In-Memory Only**: Data lost on restart - suitable for development/testing only
- **No Authentication**: API is open - add auth middleware for production use
- **No Pagination**: `/tasks` returns all tasks - add pagination for large datasets
- **Auto-Incrementing IDs**: Start from 1, increment on each create (not concurrent-safe across restarts)
