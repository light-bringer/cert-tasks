# cert-tasks

A simple RESTful API for task management built with Go.

## Features

- Create, read, update, and delete tasks
- In-memory storage (data persists while server is running)
- Clean architecture with separation of concerns
- Comprehensive test coverage
- Graceful shutdown support

## Quick Start

### Prerequisites

- Go 1.25.5 or higher

### Build and Run

```bash
# Build the application
go build -o bin/api ./cmd/api

# Run the application
./bin/api

# Or run directly
go run ./cmd/api
```

The server will start on `http://localhost:8080` by default.

### Configure Port

You can configure the port using the `PORT` environment variable:

```bash
PORT=3000 ./bin/api
```

### Run with Docker

The easiest way to run the application is using Docker:

```bash
# Using Docker Compose (recommended)
docker-compose up

# Or build and run manually
docker build -t cert-tasks:latest .
docker run -p 8080:8080 cert-tasks:latest

# Run on a different port
docker run -p 3000:8080 cert-tasks:latest
```

The Docker image uses a multi-stage build with Google's **distroless** base image for maximum security:
- **Ultra-minimal size**: ~5-8MB final image
- **No shell or package manager**: Eliminates shell-based attacks
- **Non-root user**: Runs as user "nonroot" (uid 65532)
- **Production-ready**: Google-maintained, security-focused base image

## API Endpoints

### Task Model

```json
{
  "id": 1,
  "title": "Task title",
  "description": "Task description",
  "status": "todo",
  "created_at": "2025-12-24T10:00:00Z",
  "updated_at": "2025-12-24T10:00:00Z"
}
```

**Fields:**
- `id` (int64): Auto-generated unique identifier
- `title` (string): Task title (required, non-empty)
- `description` (string): Task description (optional)
- `status` (string): Task status - either `"todo"` or `"done"` (default: `"todo"`)
- `created_at` (timestamp): Creation timestamp (auto-generated)
- `updated_at` (timestamp): Last update timestamp (auto-updated)

### Create a Task

**POST /tasks**

Create a new task.

**Request:**
```json
{
  "title": "Complete project documentation",
  "description": "Write API docs and README"
}
```

**Response:** `201 Created`
```json
{
  "id": 1,
  "title": "Complete project documentation",
  "description": "Write API docs and README",
  "status": "todo",
  "created_at": "2025-12-24T10:00:00Z",
  "updated_at": "2025-12-24T10:00:00Z"
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"Complete documentation","description":"Write README"}'
```

### List All Tasks

**GET /tasks**

Retrieve all tasks.

**Response:** `200 OK`
```json
[
  {
    "id": 1,
    "title": "Complete project documentation",
    "description": "Write API docs and README",
    "status": "todo",
    "created_at": "2025-12-24T10:00:00Z",
    "updated_at": "2025-12-24T10:00:00Z"
  },
  {
    "id": 2,
    "title": "Review code",
    "description": "Review all pull requests",
    "status": "done",
    "created_at": "2025-12-24T10:05:00Z",
    "updated_at": "2025-12-24T10:10:00Z"
  }
]
```

**Example:**
```bash
curl http://localhost:8080/tasks
```

### Get a Specific Task

**GET /tasks/{id}**

Retrieve a single task by ID.

**Response:** `200 OK` or `404 Not Found`
```json
{
  "id": 1,
  "title": "Complete project documentation",
  "description": "Write API docs and README",
  "status": "todo",
  "created_at": "2025-12-24T10:00:00Z",
  "updated_at": "2025-12-24T10:00:00Z"
}
```

**Example:**
```bash
curl http://localhost:8080/tasks/1
```

### Update a Task

**PUT /tasks/{id}**

Update an existing task (full replacement).

**Request:**
```json
{
  "title": "Updated task title",
  "description": "Updated description",
  "status": "done"
}
```

**Response:** `200 OK` or `404 Not Found`
```json
{
  "id": 1,
  "title": "Updated task title",
  "description": "Updated description",
  "status": "done",
  "created_at": "2025-12-24T10:00:00Z",
  "updated_at": "2025-12-24T10:15:00Z"
}
```

**Example:**
```bash
curl -X PUT http://localhost:8080/tasks/1 \
  -H "Content-Type: application/json" \
  -d '{"title":"Updated title","description":"New desc","status":"done"}'
```

### Delete a Task

**DELETE /tasks/{id}**

Delete a task by ID.

**Response:** `204 No Content` or `404 Not Found`

**Example:**
```bash
curl -X DELETE http://localhost:8080/tasks/1
```

## Error Responses

All error responses follow this format:

```json
{
  "error": "error message description"
}
```

**HTTP Status Codes:**
- `200 OK` - Successful GET or PUT request
- `201 Created` - Successful POST request
- `204 No Content` - Successful DELETE request
- `400 Bad Request` - Invalid request (validation errors, malformed JSON, invalid ID)
- `404 Not Found` - Task not found
- `500 Internal Server Error` - Unexpected server error

## Validation Rules

- **Title**: Required, cannot be empty or whitespace-only
- **Description**: Optional
- **Status**: Must be either `"todo"` or `"done"`

## Development

### Run Tests

```bash
# Run unit tests
make test
go test ./cmd/... ./internal/...

# Run integration tests (requires server on :8080)
make test-integration

# Run integration tests with automatic server start/stop
make test-integration-standalone

# Run all tests (unit + integration)
make test-all

# Run tests with coverage
make test-coverage

# Run tests with race detector
make test-race

# Run specific package tests
go test ./internal/repository/...
go test ./internal/handlers/...
```

### Project Structure

```
cert-tasks/
├── cmd/
│   └── api/
│       └── main.go              # Application entry point
├── internal/
│   ├── handlers/                # HTTP request handlers
│   ├── models/                  # Domain models and DTOs
│   ├── repository/              # Data access layer
│   └── server/                  # Server setup and routing
├── test/
│   └── integration_test.go      # Go integration tests
├── test_api.py                  # Python test script (deprecated)
└── README.md                    # This file
```

### Integration Tests

**Go Integration Tests**

Go integration tests are available in the `test/` directory:

```bash
# Run integration tests (requires server on :8080)
make test-integration

# Run with automatic server start/stop
make test-integration-standalone

# Run all tests (unit + integration)
make test-all
```

**Features**:
- 18 test cases covering all endpoints and validation scenarios
- Tabular output with detailed statistics
- Uses standard Go testing framework
- CI/CD friendly (exit code 0/1)
- Automatically starts/stops server in standalone mode


## Implementation Notes

- **In-Memory Storage**: Data is stored in memory and will be lost when the server stops
- **Thread-Safe**: All repository operations are thread-safe using `sync.RWMutex`
- **Graceful Shutdown**: Server handles `SIGINT` and `SIGTERM` signals for graceful shutdown
- **Auto-Generated IDs**: Task IDs are auto-incremented starting from 1
- **Timestamps**: All timestamps are in RFC3339 format

## License

MIT License - see LICENSE file for details
