# Build stage
FROM golang:1.25.5-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
# - CGO_ENABLED=0: Build a statically linked binary
# - -ldflags="-w -s": Strip debug info and symbol table for smaller binary
# - -trimpath: Remove file system paths from binary for reproducibility
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags="-w -s -extldflags '-static'" \
    -trimpath \
    -o /app/bin/api ./cmd/api

# Runtime stage - using distroless
FROM gcr.io/distroless/static-debian12:nonroot

# Copy CA certificates from builder (for HTTPS requests)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary from builder
COPY --from=builder /app/bin/api /app/api

# Distroless nonroot image already uses uid/gid 65532
# No need to create user - nonroot variant runs as user "nonroot"

# Expose port
EXPOSE 8080

# Set working directory
WORKDIR /app

# Distroless doesn't support HEALTHCHECK (no shell)
# Health checks should be configured at orchestration layer
# (Docker Compose, Kubernetes, etc.)

# Run the application
ENTRYPOINT ["/app/api"]
