# Architecture Guide for cert-tasks

## Executive Summary

This document provides comprehensive architectural guidance for the cert-tasks project, a certificate management and task processing system built in Go. It establishes foundational architectural principles, patterns, and best practices to ensure the system is scalable, maintainable, secure, and evolvable.

**Last Updated**: 2025-12-24
**Status**: Initial Architecture Definition

---

## Table of Contents

1. [Architectural Vision](#architectural-vision)
2. [Core Architectural Principles](#core-architectural-principles)
3. [Recommended Architecture Patterns](#recommended-architecture-patterns)
4. [Detailed Project Structure](#detailed-project-structure)
5. [Domain Model Design](#domain-model-design)
6. [Security Architecture](#security-architecture)
7. [Data Architecture](#data-architecture)
8. [API Design](#api-design)
9. [Task Processing Architecture](#task-processing-architecture)
10. [Observability Architecture](#observability-architecture)
11. [Scalability Considerations](#scalability-considerations)
12. [Testing Strategy](#testing-strategy)
13. [Migration and Evolution](#migration-and-evolution)
14. [Architecture Decision Records](#architecture-decision-records)

---

## Architectural Vision

### System Goals

The cert-tasks system aims to provide:

1. **Reliable Certificate Management**: Secure creation, validation, renewal, and lifecycle management of X.509 certificates
2. **Robust Task Processing**: Distributed task queue for asynchronous certificate operations
3. **Audit Trail**: Complete traceability of all certificate operations
4. **High Availability**: Resilient architecture with graceful degradation
5. **Developer Experience**: Clear APIs and maintainable codebase

### Key Quality Attributes

**Priority Order**:

1. **Security** (Highest): Cryptographic operations must be uncompromising
2. **Reliability**: System must handle failures gracefully
3. **Maintainability**: Code must be understandable and modifiable
4. **Performance**: Efficient operation without sacrificing security
5. **Scalability**: Ability to grow with demand

---

## Core Architectural Principles

### 1. Clean Architecture (Hexagonal/Ports & Adapters)

Organize code in concentric layers with dependencies pointing inward:

```
┌─────────────────────────────────────────┐
│  Infrastructure Layer                   │
│  (HTTP, gRPC, PostgreSQL, Redis)       │
│                                         │
│  ┌───────────────────────────────────┐ │
│  │  Application Layer                │ │
│  │  (Use Cases, Services)            │ │
│  │                                   │ │
│  │  ┌─────────────────────────────┐ │ │
│  │  │  Domain Layer               │ │ │
│  │  │  (Business Logic, Entities) │ │ │
│  │  │                             │ │ │
│  │  │  ┌───────────────────────┐ │ │ │
│  │  │  │  Core Entities        │ │ │ │
│  │  │  │  (Certificate, Task)  │ │ │ │
│  │  │  └───────────────────────┘ │ │ │
│  │  └─────────────────────────────┘ │ │
│  └───────────────────────────────────┘ │
└─────────────────────────────────────────┘
```

**Dependency Rule**: Source code dependencies must point only inward. Inner layers know nothing about outer layers.

**Benefits**:
- Business logic independent of frameworks and infrastructure
- Testable without external dependencies
- Flexibility to swap implementations (database, API framework, etc.)
- Clear separation of concerns

**Implementation**:

```go
// Domain layer (internal/cert/domain.go)
package cert

// Domain entity - no external dependencies
type Certificate struct {
    ID        string
    Subject   string
    NotBefore time.Time
    NotAfter  time.Time
    PublicKey []byte
}

// Repository interface defined in domain layer
type Repository interface {
    Save(ctx context.Context, cert *Certificate) error
    FindByID(ctx context.Context, id string) (*Certificate, error)
    List(ctx context.Context, filter Filter) ([]*Certificate, error)
}

// Application layer (internal/cert/service.go)
type Service struct {
    repo   Repository  // Depends on interface from domain
    logger Logger
}

func (s *Service) RenewCertificate(ctx context.Context, id string) error {
    // Business logic here
}

// Infrastructure layer (internal/storage/postgres/repository.go)
package postgres

// Implementation of domain interface
type CertRepository struct {
    db *sql.DB
}

func (r *CertRepository) Save(ctx context.Context, cert *cert.Certificate) error {
    // PostgreSQL-specific implementation
}
```

### 2. Dependency Inversion Principle (DIP)

High-level modules should not depend on low-level modules. Both should depend on abstractions.

**Example**:

```go
// Good: Depend on abstraction
type CertificateService struct {
    storage   CertificateRepository  // Interface
    validator Validator              // Interface
    logger    Logger                 // Interface
}

// Bad: Depend on concrete implementation
type CertificateService struct {
    storage   *PostgreSQLRepo  // Concrete type
    validator *X509Validator   // Concrete type
}
```

### 3. Interface Segregation Principle (ISP)

Define small, focused interfaces. Clients should not be forced to depend on methods they don't use.

```go
// Good: Small, focused interfaces
type Reader interface {
    Read(ctx context.Context, id string) (*Certificate, error)
}

type Writer interface {
    Write(ctx context.Context, cert *Certificate) error
}

type Deleter interface {
    Delete(ctx context.Context, id string) error
}

// Bad: Large interface with many responsibilities
type CertificateStore interface {
    Create(...) error
    Read(...) error
    Update(...) error
    Delete(...) error
    List(...) error
    Search(...) error
    Archive(...) error
    Restore(...) error
}
```

### 4. Single Responsibility Principle (SRP)

Each package, type, and function should have one reason to change.

```go
// Good: Single responsibility
type CertificateValidator struct {
    now func() time.Time
}

func (v *CertificateValidator) Validate(cert *Certificate) error {
    // Only validation logic
}

type CertificateSigner struct {
    privateKey crypto.PrivateKey
}

func (s *CertificateSigner) Sign(cert *Certificate) ([]byte, error) {
    // Only signing logic
}

// Bad: Multiple responsibilities
type CertificateManager struct {
    db *sql.DB
}

func (m *CertificateManager) ValidateAndSignAndStoreCertificate(...) error {
    // Validation + Signing + Storage - too many responsibilities
}
```

### 5. Explicit Over Implicit

Make dependencies, errors, and behavior explicit. Avoid hidden state and side effects.

```go
// Good: Explicit dependencies
func NewService(repo Repository, logger Logger, config Config) *Service {
    return &Service{
        repo:   repo,
        logger: logger,
        config: config,
    }
}

// Bad: Implicit dependencies (global state)
var globalDB *sql.DB

func NewService() *Service {
    return &Service{} // Where do dependencies come from?
}
```

### 6. Fail Fast

Validate inputs early. Return errors immediately. Don't continue with invalid state.

```go
// Good: Fail fast
func (s *Service) ProcessCertificate(ctx context.Context, certID string) error {
    if certID == "" {
        return errors.New("certificate ID required")
    }

    cert, err := s.repo.Get(ctx, certID)
    if err != nil {
        return fmt.Errorf("failed to get certificate: %w", err)
    }

    if err := cert.Validate(); err != nil {
        return fmt.Errorf("invalid certificate: %w", err)
    }

    // Continue with processing
    return nil
}
```

### 7. Security by Design

Security must be built-in from the start, not added later.

**Core Security Principles**:
- Defense in depth (multiple security layers)
- Principle of least privilege (minimum necessary permissions)
- Zero trust (verify everything)
- Secure defaults (safe by default configuration)
- Fail securely (errors should not leak sensitive information)

---

## Recommended Architecture Patterns

### 1. Repository Pattern

Abstract data access behind interfaces. Enables testing and swappable storage backends.

```go
// Domain layer defines interface
package cert

type Repository interface {
    Create(ctx context.Context, cert *Certificate) error
    Get(ctx context.Context, id string) (*Certificate, error)
    Update(ctx context.Context, cert *Certificate) error
    Delete(ctx context.Context, id string) error
    List(ctx context.Context, filter ListFilter) ([]*Certificate, error)
}

// Infrastructure provides implementations
// - postgres.CertRepository
// - redis.CertRepository
// - memory.CertRepository (for testing)
```

**Benefits**:
- Testable with mock implementations
- Swappable storage backends
- Clear data access contract

### 2. Strategy Pattern

Encapsulate interchangeable algorithms. Useful for signing algorithms, validation strategies, etc.

```go
type SigningStrategy interface {
    Sign(data []byte) ([]byte, error)
    Algorithm() string
}

type RSASigningStrategy struct {
    privateKey *rsa.PrivateKey
    hashFunc   crypto.Hash
}

func (s *RSASigningStrategy) Sign(data []byte) ([]byte, error) {
    hash := s.hashFunc.New()
    hash.Write(data)
    hashed := hash.Sum(nil)
    return rsa.SignPKCS1v15(rand.Reader, s.privateKey, s.hashFunc, hashed)
}

type ECDSASigningStrategy struct {
    privateKey *ecdsa.PrivateKey
}

func (s *ECDSASigningStrategy) Sign(data []byte) ([]byte, error) {
    hash := sha256.Sum256(data)
    return ecdsa.SignASN1(rand.Reader, s.privateKey, hash[:])
}

// Usage
type CertificateSigner struct {
    strategy SigningStrategy
}

func (s *CertificateSigner) SignCertificate(cert *Certificate) error {
    signature, err := s.strategy.Sign(cert.TBSCertificate)
    if err != nil {
        return err
    }
    cert.Signature = signature
    return nil
}
```

### 3. Factory Pattern

Create complex objects with multiple implementation variants.

```go
type TaskExecutorFactory interface {
    Create(taskType string) (TaskExecutor, error)
}

type DefaultTaskExecutorFactory struct {
    certService *CertificateService
    logger      Logger
    meter       Meter
}

func (f *DefaultTaskExecutorFactory) Create(taskType string) (TaskExecutor, error) {
    switch taskType {
    case "cert_renewal":
        return NewCertRenewalExecutor(f.certService, f.logger, f.meter), nil
    case "cert_validation":
        return NewCertValidationExecutor(f.certService, f.logger, f.meter), nil
    case "cert_revocation":
        return NewCertRevocationExecutor(f.certService, f.logger, f.meter), nil
    default:
        return nil, fmt.Errorf("unknown task type: %s", taskType)
    }
}
```

### 4. Command Pattern

Encapsulate operations as objects. Perfect for task queue implementation.

```go
type Command interface {
    Execute(ctx context.Context) error
    Name() string
    Validate() error
}

type RenewCertificateCommand struct {
    CertificateID string
    ValidityDays  int
    Executor      *CertificateService
}

func (c *RenewCertificateCommand) Name() string {
    return "renew_certificate"
}

func (c *RenewCertificateCommand) Validate() error {
    if c.CertificateID == "" {
        return errors.New("certificate ID required")
    }
    if c.ValidityDays <= 0 {
        return errors.New("validity days must be positive")
    }
    return nil
}

func (c *RenewCertificateCommand) Execute(ctx context.Context) error {
    if err := c.Validate(); err != nil {
        return err
    }
    return c.Executor.RenewCertificate(ctx, c.CertificateID, c.ValidityDays)
}

// Task queue processes commands
type TaskQueue struct {
    commands chan Command
}

func (tq *TaskQueue) Enqueue(cmd Command) error {
    if err := cmd.Validate(); err != nil {
        return err
    }
    tq.commands <- cmd
    return nil
}

func (tq *TaskQueue) Worker(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case cmd := <-tq.commands:
            if err := cmd.Execute(ctx); err != nil {
                // Handle error (retry, dead letter queue, etc.)
            }
        }
    }
}
```

### 5. Observer Pattern (Event-Driven Architecture)

Decouple components through events. Useful for notifications, audit logging, metrics.

```go
type Event interface {
    Type() string
    Timestamp() time.Time
    Metadata() map[string]interface{}
}

type CertificateRenewedEvent struct {
    CertID    string
    OldExpiry time.Time
    NewExpiry time.Time
    timestamp time.Time
}

func (e *CertificateRenewedEvent) Type() string {
    return "certificate.renewed"
}

func (e *CertificateRenewedEvent) Timestamp() time.Time {
    return e.timestamp
}

type EventListener interface {
    OnEvent(ctx context.Context, event Event) error
    SupportedEvents() []string
}

type AuditLogger struct {
    repo AuditRepository
}

func (a *AuditLogger) OnEvent(ctx context.Context, event Event) error {
    return a.repo.LogEvent(ctx, event)
}

func (a *AuditLogger) SupportedEvents() []string {
    return []string{"certificate.*", "task.*"}
}

type EventBus struct {
    listeners map[string][]EventListener
    mu        sync.RWMutex
}

func (eb *EventBus) Subscribe(eventPattern string, listener EventListener) {
    eb.mu.Lock()
    defer eb.mu.Unlock()
    eb.listeners[eventPattern] = append(eb.listeners[eventPattern], listener)
}

func (eb *EventBus) Publish(ctx context.Context, event Event) error {
    eb.mu.RLock()
    defer eb.mu.RUnlock()

    var errs []error
    for pattern, listeners := range eb.listeners {
        if matchesPattern(event.Type(), pattern) {
            for _, listener := range listeners {
                if err := listener.OnEvent(ctx, event); err != nil {
                    errs = append(errs, err)
                }
            }
        }
    }
    return errors.Join(errs...)
}
```

### 6. Builder Pattern

Construct complex objects step-by-step with validation.

```go
type CertificateBuilder struct {
    cert   *Certificate
    errors []error
}

func NewCertificateBuilder() *CertificateBuilder {
    return &CertificateBuilder{
        cert:   &Certificate{},
        errors: make([]error, 0),
    }
}

func (b *CertificateBuilder) WithSubject(subject string) *CertificateBuilder {
    if subject == "" {
        b.errors = append(b.errors, errors.New("subject cannot be empty"))
    }
    b.cert.Subject = subject
    return b
}

func (b *CertificateBuilder) WithValidityPeriod(days int) *CertificateBuilder {
    if days <= 0 {
        b.errors = append(b.errors, errors.New("validity period must be positive"))
        return b
    }
    b.cert.NotBefore = time.Now().UTC()
    b.cert.NotAfter = time.Now().UTC().AddDate(0, 0, days)
    return b
}

func (b *CertificateBuilder) WithKeyPair(pub, priv []byte) *CertificateBuilder {
    if len(pub) == 0 || len(priv) == 0 {
        b.errors = append(b.errors, errors.New("key pair required"))
    }
    b.cert.PublicKey = pub
    // Store private key securely (not in Certificate object)
    return b
}

func (b *CertificateBuilder) Build() (*Certificate, error) {
    if len(b.errors) > 0 {
        return nil, errors.Join(b.errors...)
    }

    if err := b.cert.Validate(); err != nil {
        return nil, err
    }

    // Generate ID if not set
    if b.cert.ID == "" {
        b.cert.ID = generateID()
    }

    return b.cert, nil
}

// Usage
cert, err := NewCertificateBuilder().
    WithSubject("example.com").
    WithValidityPeriod(365).
    WithKeyPair(publicKey, privateKey).
    Build()
```

### 7. Decorator Pattern

Add behavior to objects dynamically. Useful for middleware, caching, logging.

```go
// Base interface
type CertificateRepository interface {
    Get(ctx context.Context, id string) (*Certificate, error)
}

// Logging decorator
type LoggingRepository struct {
    base   CertificateRepository
    logger Logger
}

func (r *LoggingRepository) Get(ctx context.Context, id string) (*Certificate, error) {
    start := time.Now()
    r.logger.Info("getting certificate", "id", id)

    cert, err := r.base.Get(ctx, id)

    duration := time.Since(start)
    if err != nil {
        r.logger.Error("failed to get certificate", "id", id, "error", err, "duration", duration)
        return nil, err
    }

    r.logger.Info("got certificate", "id", id, "duration", duration)
    return cert, nil
}

// Caching decorator
type CachingRepository struct {
    base  CertificateRepository
    cache Cache
    ttl   time.Duration
}

func (r *CachingRepository) Get(ctx context.Context, id string) (*Certificate, error) {
    // Check cache
    if cached, found := r.cache.Get(id); found {
        return cached.(*Certificate), nil
    }

    // Cache miss - fetch from base repository
    cert, err := r.base.Get(ctx, id)
    if err != nil {
        return nil, err
    }

    // Store in cache
    r.cache.Set(id, cert, r.ttl)
    return cert, nil
}

// Metrics decorator
type MetricsRepository struct {
    base  CertificateRepository
    meter Meter
}

func (r *MetricsRepository) Get(ctx context.Context, id string) (*Certificate, error) {
    start := time.Now()
    cert, err := r.base.Get(ctx, id)
    duration := time.Since(start)

    status := "success"
    if err != nil {
        status = "error"
    }

    r.meter.RecordHistogram("repo.get.duration", duration.Milliseconds(), "status", status)
    return cert, err
}

// Compose decorators
repo := &PostgresRepository{db: db}
repo = &LoggingRepository{base: repo, logger: logger}
repo = &CachingRepository{base: repo, cache: cache, ttl: 5 * time.Minute}
repo = &MetricsRepository{base: repo, meter: meter}
```

---

## Detailed Project Structure

```
cert-tasks/
├── cmd/                                    # Application entry points
│   ├── cert-tasks/                        # Main CLI application
│   │   └── main.go
│   ├── cert-server/                       # HTTP/gRPC API server
│   │   └── main.go
│   └── cert-worker/                       # Background task worker
│       └── main.go
│
├── internal/                              # Private application code
│   ├── cert/                             # Certificate domain
│   │   ├── certificate.go                # Core entity
│   │   ├── service.go                    # Business logic
│   │   ├── repository.go                 # Repository interface
│   │   ├── validator.go                  # Validation logic
│   │   ├── parser.go                     # Certificate parsing
│   │   ├── generator.go                  # Certificate generation
│   │   ├── errors.go                     # Domain errors
│   │   └── types.go                      # Value objects
│   │
│   ├── task/                             # Task processing domain
│   │   ├── task.go                       # Task entity
│   │   ├── executor.go                   # Task execution
│   │   ├── scheduler.go                  # Task scheduling
│   │   ├── queue.go                      # Queue interface
│   │   ├── handler.go                    # Task handlers
│   │   └── types.go                      # Task types
│   │
│   ├── crypto/                           # Cryptographic operations
│   │   ├── signer.go                     # Digital signing
│   │   ├── verifier.go                   # Signature verification
│   │   ├── keystore.go                   # Key management
│   │   └── algorithms.go                 # Supported algorithms
│   │
│   ├── storage/                          # Storage implementations
│   │   ├── postgres/                     # PostgreSQL
│   │   │   ├── repository.go
│   │   │   ├── migrations/
│   │   │   └── queries.go
│   │   ├── redis/                        # Redis cache
│   │   │   └── cache.go
│   │   └── memory/                       # In-memory (testing)
│   │       └── repository.go
│   │
│   ├── api/                              # API layer
│   │   ├── http/                         # HTTP handlers
│   │   │   ├── handlers.go
│   │   │   ├── middleware.go
│   │   │   └── routes.go
│   │   ├── grpc/                         # gRPC services
│   │   │   ├── server.go
│   │   │   └── services.go
│   │   └── dto/                          # Data transfer objects
│   │       └── certificate.go
│   │
│   ├── config/                           # Configuration
│   │   ├── config.go                     # Config types
│   │   ├── loader.go                     # Loading logic
│   │   └── validator.go                  # Config validation
│   │
│   ├── observability/                    # Monitoring & logging
│   │   ├── logger.go                     # Structured logging
│   │   ├── metrics.go                    # Metrics collection
│   │   ├── tracing.go                    # Distributed tracing
│   │   └── interfaces.go                 # Observability interfaces
│   │
│   └── audit/                            # Audit logging
│       ├── audit.go                      # Audit types
│       ├── logger.go                     # Audit logger
│       └── repository.go                 # Audit storage
│
├── pkg/                                  # Public library code
│   ├── certutil/                         # Certificate utilities
│   │   ├── parse.go
│   │   ├── format.go
│   │   └── validate.go
│   ├── errors/                           # Error utilities
│   │   └── errors.go
│   └── timeutil/                         # Time utilities
│       └── time.go
│
├── test/                                 # Integration & E2E tests
│   ├── integration/                      # Integration tests
│   │   ├── cert_test.go
│   │   └── task_test.go
│   ├── e2e/                              # End-to-end tests
│   │   └── scenarios/
│   └── testdata/                         # Test fixtures
│       ├── certs/
│       └── keys/
│
├── scripts/                              # Build & utility scripts
│   ├── build.sh
│   ├── test.sh
│   ├── migrate.sh
│   └── deploy.sh
│
├── deployments/                          # Deployment configs
│   ├── docker/
│   │   ├── Dockerfile
│   │   └── docker-compose.yml
│   ├── kubernetes/
│   │   ├── deployment.yaml
│   │   └── service.yaml
│   └── terraform/
│       └── main.tf
│
├── docs/                                 # Documentation
│   ├── architecture/                     # Architecture docs
│   │   ├── ARCHITECTURE.md               # This file
│   │   ├── ADR-001-database-choice.md
│   │   └── ADR-002-task-queue.md
│   ├── api/                              # API documentation
│   │   └── openapi.yaml
│   └── guides/                           # User guides
│       └── getting-started.md
│
├── api/                                  # API definitions
│   ├── proto/                            # Protocol buffers
│   │   └── cert.proto
│   └── openapi/                          # OpenAPI specs
│       └── api.yaml
│
├── .github/                              # GitHub config
│   └── workflows/
│       ├── ci.yml
│       └── release.yml
│
├── .golangci.yml                         # Linter config
├── Makefile                              # Build automation
├── go.mod
├── go.sum
├── README.md
├── CLAUDE.md
└── LICENSE
```

### Directory Responsibilities

**cmd/**: Application entry points only. Minimal logic - wire dependencies and start application.

**internal/**: Private code organized by domain and layer. Cannot be imported by external projects.

**pkg/**: Reusable utilities that could be extracted into separate libraries.

**test/**: Integration and E2E tests that span multiple packages.

**docs/**: All documentation including architecture decisions, API docs, and guides.

**deployments/**: Infrastructure as code and deployment configurations.

**scripts/**: Automation scripts for common tasks.

---

## Domain Model Design

### Certificate Domain

```go
package cert

import (
    "crypto"
    "time"
)

// Certificate represents a managed X.509 certificate
type Certificate struct {
    ID              string
    Subject         string
    Issuer          string
    SerialNumber    string
    NotBefore       time.Time
    NotAfter        time.Time
    KeyAlgorithm    string
    SignatureAlgorithm string
    PublicKey       []byte
    Fingerprint     string
    Extensions      map[string]interface{}
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

// Value objects
type CertificateID string
type SerialNumber string
type Fingerprint string

// Business rules
func (c *Certificate) IsExpired() bool {
    return time.Now().UTC().After(c.NotAfter)
}

func (c *Certificate) ExpiresWithin(duration time.Duration) bool {
    return time.Now().UTC().Add(duration).After(c.NotAfter)
}

func (c *Certificate) IsValid() bool {
    now := time.Now().UTC()
    return now.After(c.NotBefore) && now.Before(c.NotAfter)
}

func (c *Certificate) Validate() error {
    if c.Subject == "" {
        return ErrInvalidSubject
    }
    if c.NotBefore.IsZero() || c.NotAfter.IsZero() {
        return ErrInvalidValidity
    }
    if c.NotAfter.Before(c.NotBefore) {
        return ErrInvalidValidityPeriod
    }
    return nil
}

// Domain errors
var (
    ErrCertificateNotFound     = errors.New("certificate not found")
    ErrCertificateExpired      = errors.New("certificate expired")
    ErrInvalidCertificate      = errors.New("invalid certificate")
    ErrInvalidSubject          = errors.New("invalid subject")
    ErrInvalidValidity         = errors.New("invalid validity period")
    ErrInvalidValidityPeriod   = errors.New("validity period end before start")
)

// Repository interface (defined in domain)
type Repository interface {
    Create(ctx context.Context, cert *Certificate) error
    Get(ctx context.Context, id string) (*Certificate, error)
    Update(ctx context.Context, cert *Certificate) error
    Delete(ctx context.Context, id string) error
    List(ctx context.Context, filter ListFilter) ([]*Certificate, error)
    FindExpiring(ctx context.Context, within time.Duration) ([]*Certificate, error)
}

// Service (application layer)
type Service struct {
    repo      Repository
    validator Validator
    signer    Signer
    logger    Logger
    events    EventBus
}

func (s *Service) CreateCertificate(ctx context.Context, req CreateRequest) (*Certificate, error) {
    // Validate request
    if err := req.Validate(); err != nil {
        return nil, fmt.Errorf("invalid request: %w", err)
    }

    // Generate certificate
    cert, err := s.generator.Generate(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("failed to generate certificate: %w", err)
    }

    // Sign certificate
    if err := s.signer.Sign(ctx, cert); err != nil {
        return nil, fmt.Errorf("failed to sign certificate: %w", err)
    }

    // Store certificate
    if err := s.repo.Create(ctx, cert); err != nil {
        return nil, fmt.Errorf("failed to store certificate: %w", err)
    }

    // Publish event
    s.events.Publish(ctx, &CertificateCreatedEvent{
        CertID: cert.ID,
        Subject: cert.Subject,
    })

    s.logger.Info("certificate created", "id", cert.ID, "subject", cert.Subject)
    return cert, nil
}
```

### Task Domain

```go
package task

import (
    "context"
    "time"
)

// Task represents a unit of work
type Task struct {
    ID          string
    Type        string
    Payload     map[string]interface{}
    Status      Status
    Priority    Priority
    MaxRetries  int
    RetryCount  int
    ScheduledAt time.Time
    StartedAt   *time.Time
    CompletedAt *time.Time
    Error       string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type Status string

const (
    StatusPending   Status = "pending"
    StatusRunning   Status = "running"
    StatusCompleted Status = "completed"
    StatusFailed    Status = "failed"
    StatusRetrying  Status = "retrying"
)

type Priority int

const (
    PriorityLow Priority = iota
    PriorityNormal
    PriorityHigh
    PriorityCritical
)

// Business logic
func (t *Task) CanRetry() bool {
    return t.RetryCount < t.MaxRetries && t.Status == StatusFailed
}

func (t *Task) MarkStarted() {
    now := time.Now().UTC()
    t.StartedAt = &now
    t.Status = StatusRunning
    t.UpdatedAt = now
}

func (t *Task) MarkCompleted() {
    now := time.Now().UTC()
    t.CompletedAt = &now
    t.Status = StatusCompleted
    t.UpdatedAt = now
}

func (t *Task) MarkFailed(err error) {
    t.Error = err.Error()
    t.Status = StatusFailed
    t.UpdatedAt = time.Now().UTC()
}

// Executor interface
type Executor interface {
    Execute(ctx context.Context, task *Task) error
    CanHandle(taskType string) bool
}

// Queue interface
type Queue interface {
    Enqueue(ctx context.Context, task *Task) error
    Dequeue(ctx context.Context) (*Task, error)
    Ack(ctx context.Context, taskID string) error
    Nack(ctx context.Context, taskID string) error
    Retry(ctx context.Context, taskID string) error
}

// Repository interface
type Repository interface {
    Create(ctx context.Context, task *Task) error
    Get(ctx context.Context, id string) (*Task, error)
    Update(ctx context.Context, task *Task) error
    List(ctx context.Context, filter ListFilter) ([]*Task, error)
}
```

---

## Security Architecture

### Cryptographic Standards

**Supported Algorithms**:

1. **RSA**:
   - Minimum key size: 2048 bits
   - Recommended: 4096 bits
   - Hash: SHA-256 or stronger

2. **ECDSA**:
   - Curves: P-256 (secp256r1), P-384 (secp384r1)
   - Hash: SHA-256 for P-256, SHA-384 for P-384

3. **Hashing**:
   - SHA-256 minimum
   - SHA-384 for sensitive operations
   - Never MD5, SHA-1 (deprecated)

**Deprecated/Forbidden**:
- MD5
- SHA-1
- RSA < 2048 bits
- DES, 3DES
- RC4

### Key Management

```go
package crypto

type KeyStore interface {
    // Store encrypted private key
    StorePrivateKey(ctx context.Context, id string, key crypto.PrivateKey, passphrase []byte) error

    // Retrieve and decrypt private key
    GetPrivateKey(ctx context.Context, id string, passphrase []byte) (crypto.PrivateKey, error)

    // Store public key
    StorePublicKey(ctx context.Context, id string, key crypto.PublicKey) error

    // Retrieve public key
    GetPublicKey(ctx context.Context, id string) (crypto.PublicKey, error)

    // Rotate encryption key
    RotateEncryptionKey(ctx context.Context, oldKey, newKey []byte) error

    // Delete key (secure wipe)
    DeleteKey(ctx context.Context, id string) error
}

// Implementation using encrypted storage
type EncryptedKeyStore struct {
    storage Storage
    cipher  Cipher
}

func (ks *EncryptedKeyStore) StorePrivateKey(ctx context.Context, id string, key crypto.PrivateKey, passphrase []byte) error {
    // Marshal key to bytes
    keyBytes, err := x509.MarshalPKCS8PrivateKey(key)
    if err != nil {
        return err
    }

    // Derive encryption key from passphrase using PBKDF2
    salt := generateSalt()
    encKey := pbkdf2.Key(passphrase, salt, 100000, 32, sha256.New)

    // Encrypt key bytes using AES-256-GCM
    encrypted, err := ks.cipher.Encrypt(keyBytes, encKey)
    if err != nil {
        return err
    }

    // Store encrypted key with salt
    return ks.storage.Store(ctx, id, &KeyRecord{
        EncryptedKey: encrypted,
        Salt:         salt,
    })
}
```

### Certificate Validation

```go
package cert

type Validator interface {
    Validate(ctx context.Context, cert *Certificate) error
}

type X509Validator struct {
    trustStore TrustStore
    crlChecker CRLChecker
    ocspClient OCSPClient
    clockSkew  time.Duration
}

func (v *X509Validator) Validate(ctx context.Context, cert *Certificate) error {
    // 1. Check time validity with clock skew tolerance
    now := time.Now().UTC()
    if now.Add(v.clockSkew).Before(cert.NotBefore) {
        return ErrCertificateNotYetValid
    }
    if now.Add(-v.clockSkew).After(cert.NotAfter) {
        return ErrCertificateExpired
    }

    // 2. Verify certificate chain
    if err := v.verifyChain(ctx, cert); err != nil {
        return fmt.Errorf("chain verification failed: %w", err)
    }

    // 3. Check revocation status (CRL)
    if err := v.crlChecker.CheckRevocation(ctx, cert); err != nil {
        return fmt.Errorf("revocation check failed: %w", err)
    }

    // 4. Check OCSP (if available)
    if err := v.ocspClient.CheckStatus(ctx, cert); err != nil {
        return fmt.Errorf("OCSP check failed: %w", err)
    }

    // 5. Validate key usage and extended key usage
    if err := v.validateKeyUsage(cert); err != nil {
        return err
    }

    // 6. Verify signature
    if err := v.verifySignature(cert); err != nil {
        return fmt.Errorf("signature verification failed: %w", err)
    }

    return nil
}
```

### Audit Logging

All security-sensitive operations must be logged:

```go
package audit

type AuditEvent struct {
    ID        string
    Type      EventType
    Actor     string // User or service performing action
    Resource  string // Resource being acted upon
    Action    string // Action performed
    Result    Result // Success or failure
    Timestamp time.Time
    Metadata  map[string]interface{}
    IPAddress string
    UserAgent string
}

type EventType string

const (
    EventCertificateCreated  EventType = "certificate.created"
    EventCertificateRenewed  EventType = "certificate.renewed"
    EventCertificateRevoked  EventType = "certificate.revoked"
    EventCertificateDeleted  EventType = "certificate.deleted"
    EventKeyAccessed         EventType = "key.accessed"
    EventKeyRotated          EventType = "key.rotated"
    EventSigningPerformed    EventType = "signing.performed"
    EventValidationPerformed EventType = "validation.performed"
)

type Result string

const (
    ResultSuccess Result = "success"
    ResultFailure Result = "failure"
)

type Logger interface {
    Log(ctx context.Context, event *AuditEvent) error
}

// Usage
auditLogger.Log(ctx, &AuditEvent{
    Type:      EventCertificateCreated,
    Actor:     "user@example.com",
    Resource:  cert.ID,
    Action:    "create_certificate",
    Result:    ResultSuccess,
    Timestamp: time.Now().UTC(),
    Metadata: map[string]interface{}{
        "subject":    cert.Subject,
        "expires_at": cert.NotAfter,
    },
})
```

### Secrets Management

**Never store secrets in code or configuration files.**

```go
package config

type SecretsProvider interface {
    GetSecret(ctx context.Context, name string) (string, error)
}

// Environment variable provider (development)
type EnvSecretsProvider struct{}

func (p *EnvSecretsProvider) GetSecret(ctx context.Context, name string) (string, error) {
    value := os.Getenv(name)
    if value == "" {
        return "", fmt.Errorf("secret %s not found", name)
    }
    return value, nil
}

// AWS Secrets Manager provider (production)
type AWSSecretsProvider struct {
    client *secretsmanager.Client
}

func (p *AWSSecretsProvider) GetSecret(ctx context.Context, name string) (string, error) {
    result, err := p.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
        SecretId: aws.String(name),
    })
    if err != nil {
        return "", err
    }
    return *result.SecretString, nil
}

// HashiCorp Vault provider (production)
type VaultSecretsProvider struct {
    client *vault.Client
}

func (p *VaultSecretsProvider) GetSecret(ctx context.Context, name string) (string, error) {
    secret, err := p.client.Logical().Read(name)
    if err != nil {
        return "", err
    }
    return secret.Data["value"].(string), nil
}
```

---

## Data Architecture

### Database Schema (PostgreSQL)

```sql
-- Certificates table
CREATE TABLE certificates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject VARCHAR(255) NOT NULL,
    issuer VARCHAR(255) NOT NULL,
    serial_number VARCHAR(255) NOT NULL UNIQUE,
    not_before TIMESTAMP NOT NULL,
    not_after TIMESTAMP NOT NULL,
    key_algorithm VARCHAR(50) NOT NULL,
    signature_algorithm VARCHAR(50) NOT NULL,
    public_key BYTEA NOT NULL,
    fingerprint VARCHAR(64) NOT NULL,
    extensions JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    INDEX idx_certificates_subject (subject),
    INDEX idx_certificates_serial (serial_number),
    INDEX idx_certificates_expiry (not_after),
    INDEX idx_certificates_fingerprint (fingerprint)
);

-- Tasks table
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,
    retry_count INTEGER NOT NULL DEFAULT 0,
    scheduled_at TIMESTAMP NOT NULL,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    error TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    INDEX idx_tasks_status (status),
    INDEX idx_tasks_type (type),
    INDEX idx_tasks_scheduled (scheduled_at),
    INDEX idx_tasks_priority (priority)
);

-- Audit log table
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    actor VARCHAR(255) NOT NULL,
    resource VARCHAR(255) NOT NULL,
    action VARCHAR(100) NOT NULL,
    result VARCHAR(20) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    metadata JSONB,
    ip_address INET,
    user_agent TEXT,
    INDEX idx_audit_timestamp (timestamp),
    INDEX idx_audit_actor (actor),
    INDEX idx_audit_resource (resource),
    INDEX idx_audit_type (event_type)
);

-- Key metadata table (actual keys stored encrypted)
CREATE TABLE key_metadata (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_type VARCHAR(50) NOT NULL,
    algorithm VARCHAR(50) NOT NULL,
    key_size INTEGER NOT NULL,
    certificate_id UUID REFERENCES certificates(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW(),
    rotated_at TIMESTAMP,
    expires_at TIMESTAMP,
    INDEX idx_key_cert (certificate_id),
    INDEX idx_key_expiry (expires_at)
);
```

### Caching Strategy

```go
package storage

// Multi-layer caching
type CachedRepository struct {
    primary   Repository      // PostgreSQL
    l1Cache   Cache          // In-memory (Redis)
    l2Cache   Cache          // Local in-process cache
    ttl       time.Duration
}

func (r *CachedRepository) Get(ctx context.Context, id string) (*Certificate, error) {
    // L2 cache (local)
    if cert, found := r.l2Cache.Get(id); found {
        return cert.(*Certificate), nil
    }

    // L1 cache (Redis)
    if cert, found := r.l1Cache.Get(id); found {
        // Populate L2 cache
        r.l2Cache.Set(id, cert, r.ttl)
        return cert.(*Certificate), nil
    }

    // Cache miss - fetch from database
    cert, err := r.primary.Get(ctx, id)
    if err != nil {
        return nil, err
    }

    // Populate both cache layers
    r.l1Cache.Set(id, cert, r.ttl)
    r.l2Cache.Set(id, cert, r.ttl)

    return cert, nil
}

func (r *CachedRepository) Update(ctx context.Context, cert *Certificate) error {
    // Update database
    if err := r.primary.Update(ctx, cert); err != nil {
        return err
    }

    // Invalidate caches
    r.l1Cache.Delete(cert.ID)
    r.l2Cache.Delete(cert.ID)

    return nil
}
```

### Data Migration Strategy

Use golang-migrate for database migrations:

```go
// migrations/000001_create_certificates_table.up.sql
CREATE TABLE certificates (
    -- schema definition
);

// migrations/000001_create_certificates_table.down.sql
DROP TABLE certificates;

// Migration runner
package storage

import "github.com/golang-migrate/migrate/v4"

func RunMigrations(dbURL string) error {
    m, err := migrate.New(
        "file://migrations",
        dbURL,
    )
    if err != nil {
        return err
    }

    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return err
    }

    return nil
}
```

---

## API Design

### RESTful HTTP API

```go
// HTTP routes
GET    /api/v1/certificates         - List certificates
POST   /api/v1/certificates         - Create certificate
GET    /api/v1/certificates/:id     - Get certificate
PUT    /api/v1/certificates/:id     - Update certificate
DELETE /api/v1/certificates/:id     - Delete certificate
POST   /api/v1/certificates/:id/renew - Renew certificate
POST   /api/v1/certificates/:id/revoke - Revoke certificate

GET    /api/v1/tasks               - List tasks
POST   /api/v1/tasks               - Create task
GET    /api/v1/tasks/:id           - Get task
DELETE /api/v1/tasks/:id           - Cancel task

GET    /health                     - Health check
GET    /ready                      - Readiness check
GET    /metrics                    - Prometheus metrics

// Request/Response DTOs
package dto

type CreateCertificateRequest struct {
    Subject      string            `json:"subject" validate:"required"`
    ValidityDays int               `json:"validity_days" validate:"required,min=1,max=825"`
    KeyAlgorithm string            `json:"key_algorithm" validate:"required,oneof=RSA2048 RSA4096 ECDSA256 ECDSA384"`
    Extensions   map[string]string `json:"extensions,omitempty"`
}

type CertificateResponse struct {
    ID              string                 `json:"id"`
    Subject         string                 `json:"subject"`
    Issuer          string                 `json:"issuer"`
    NotBefore       time.Time              `json:"not_before"`
    NotAfter        time.Time              `json:"not_after"`
    Fingerprint     string                 `json:"fingerprint"`
    Status          string                 `json:"status"`
    CreatedAt       time.Time              `json:"created_at"`
}

// Error response
type ErrorResponse struct {
    Error   string                 `json:"error"`
    Code    string                 `json:"code"`
    Details map[string]interface{} `json:"details,omitempty"`
}
```

### API Middleware

```go
package middleware

// Logging middleware
func Logging(logger Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()

            // Wrap response writer to capture status code
            ww := &responseWriter{ResponseWriter: w}

            next.ServeHTTP(ww, r)

            logger.Info("http request",
                "method", r.Method,
                "path", r.URL.Path,
                "status", ww.status,
                "duration", time.Since(start),
                "ip", r.RemoteAddr,
            )
        })
    }
}

// Authentication middleware
func Authentication(authService AuthService) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := r.Header.Get("Authorization")
            if token == "" {
                http.Error(w, "unauthorized", http.StatusUnauthorized)
                return
            }

            user, err := authService.Validate(r.Context(), token)
            if err != nil {
                http.Error(w, "unauthorized", http.StatusUnauthorized)
                return
            }

            // Add user to context
            ctx := context.WithValue(r.Context(), "user", user)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// Rate limiting middleware
func RateLimit(limiter *rate.Limiter) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if !limiter.Allow() {
                http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

// Recovery middleware
func Recovery(logger Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            defer func() {
                if err := recover(); err != nil {
                    logger.Error("panic recovered",
                        "error", err,
                        "stack", string(debug.Stack()),
                    )
                    http.Error(w, "internal server error", http.StatusInternalServerError)
                }
            }()
            next.ServeHTTP(w, r)
        })
    }
}
```

---

## Task Processing Architecture

### Task Queue Design

```go
package task

// Distributed task queue using Redis
type RedisQueue struct {
    client   *redis.Client
    queueKey string
}

func (q *RedisQueue) Enqueue(ctx context.Context, task *Task) error {
    // Serialize task
    data, err := json.Marshal(task)
    if err != nil {
        return err
    }

    // Add to Redis sorted set (score = priority)
    return q.client.ZAdd(ctx, q.queueKey, &redis.Z{
        Score:  float64(task.Priority),
        Member: data,
    }).Err()
}

func (q *RedisQueue) Dequeue(ctx context.Context) (*Task, error) {
    // Pop highest priority task (atomic operation)
    result, err := q.client.ZPopMax(ctx, q.queueKey).Result()
    if err != nil {
        return nil, err
    }

    if len(result) == 0 {
        return nil, ErrQueueEmpty
    }

    // Deserialize task
    var task Task
    if err := json.Unmarshal([]byte(result[0].Member.(string)), &task); err != nil {
        return nil, err
    }

    return &task, nil
}

// Worker pool for task processing
type WorkerPool struct {
    workers   int
    queue     Queue
    executor  Executor
    logger    Logger
    meter     Meter
}

func (wp *WorkerPool) Start(ctx context.Context) error {
    wp.logger.Info("starting worker pool", "workers", wp.workers)

    errGroup, ctx := errgroup.WithContext(ctx)

    for i := 0; i < wp.workers; i++ {
        workerID := i
        errGroup.Go(func() error {
            return wp.worker(ctx, workerID)
        })
    }

    return errGroup.Wait()
}

func (wp *WorkerPool) worker(ctx context.Context, id int) error {
    wp.logger.Info("worker started", "worker_id", id)

    for {
        select {
        case <-ctx.Done():
            wp.logger.Info("worker stopping", "worker_id", id)
            return nil

        default:
            // Dequeue task with timeout
            taskCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
            task, err := wp.queue.Dequeue(taskCtx)
            cancel()

            if err != nil {
                if err == ErrQueueEmpty {
                    time.Sleep(1 * time.Second)
                    continue
                }
                wp.logger.Error("failed to dequeue task", "error", err)
                continue
            }

            // Process task
            wp.processTask(ctx, task)
        }
    }
}

func (wp *WorkerPool) processTask(ctx context.Context, task *Task) {
    start := time.Now()
    task.MarkStarted()

    wp.logger.Info("processing task",
        "task_id", task.ID,
        "type", task.Type,
    )

    // Execute task with timeout
    execCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
    defer cancel()

    err := wp.executor.Execute(execCtx, task)
    duration := time.Since(start)

    if err != nil {
        task.MarkFailed(err)
        wp.logger.Error("task failed",
            "task_id", task.ID,
            "error", err,
            "duration", duration,
        )

        // Retry if possible
        if task.CanRetry() {
            task.RetryCount++
            task.Status = StatusRetrying
            wp.queue.Enqueue(ctx, task)
        }

        wp.meter.RecordCounter("tasks.failed", 1, "type", task.Type)
    } else {
        task.MarkCompleted()
        wp.logger.Info("task completed",
            "task_id", task.ID,
            "duration", duration,
        )
        wp.meter.RecordCounter("tasks.completed", 1, "type", task.Type)
    }

    wp.meter.RecordHistogram("task.duration", duration.Milliseconds(), "type", task.Type)
}
```

---

## Observability Architecture

### Structured Logging

```go
package observability

import "log/slog"

// Logger interface
type Logger interface {
    Debug(msg string, args ...interface{})
    Info(msg string, args ...interface{})
    Warn(msg string, args ...interface{})
    Error(msg string, args ...interface{})
    With(args ...interface{}) Logger
}

// Implementation using slog
type SlogLogger struct {
    logger *slog.Logger
}

func NewLogger() Logger {
    return &SlogLogger{
        logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
            Level: slog.LevelInfo,
        })),
    }
}

func (l *SlogLogger) Info(msg string, args ...interface{}) {
    l.logger.Info(msg, args...)
}

// Usage
logger.Info("certificate created",
    "cert_id", cert.ID,
    "subject", cert.Subject,
    "expires_at", cert.NotAfter,
    "user_id", userID,
)
```

### Metrics Collection

```go
package observability

// Metrics interface
type Meter interface {
    RecordCounter(name string, value int64, labels ...string)
    RecordGauge(name string, value float64, labels ...string)
    RecordHistogram(name string, value float64, labels ...string)
}

// Prometheus implementation
type PrometheusMeter struct {
    counters   map[string]*prometheus.CounterVec
    gauges     map[string]*prometheus.GaugeVec
    histograms map[string]*prometheus.HistogramVec
}

// Usage
meter.RecordCounter("cert.created", 1, "subject", cert.Subject)
meter.RecordGauge("cert.expiring.count", float64(count))
meter.RecordHistogram("cert.validation.duration", duration.Milliseconds())
```

### Distributed Tracing

```go
package observability

import "go.opentelemetry.io/otel"

func (s *Service) CreateCertificate(ctx context.Context, req Request) error {
    ctx, span := otel.Tracer("cert-service").Start(ctx, "CreateCertificate")
    defer span.End()

    span.SetAttributes(
        attribute.String("subject", req.Subject),
        attribute.Int("validity_days", req.ValidityDays),
    )

    // Continue with logic
    return nil
}
```

---

## Scalability Considerations

### Horizontal Scaling

**Application Layer**:
- Stateless services (no in-memory state)
- Multiple instances behind load balancer
- Session affinity not required

**Database Layer**:
- Read replicas for read-heavy workloads
- Connection pooling
- Prepared statements

**Task Queue**:
- Multiple workers consuming from shared queue
- Distributed queue (Redis, RabbitMQ)

**Caching**:
- Distributed cache (Redis cluster)
- Cache invalidation strategy

### Performance Optimization

**Database**:
- Index frequently queried columns
- Use EXPLAIN ANALYZE for slow queries
- Implement pagination for large result sets
- Use database-level constraints

**Caching**:
- Cache immutable data (certificates rarely change)
- Multi-level caching (L1: in-memory, L2: Redis)
- Cache warming on startup
- TTL-based expiration

**Concurrency**:
- Worker pools for concurrent processing
- Context-based cancellation
- Rate limiting to prevent overload

---

## Testing Strategy

### Test Pyramid

```
        E2E Tests (5%)
       /            \
      /              \
     Integration Tests (15%)
    /                  \
   /                    \
  Unit Tests (80%)
```

### Unit Tests

```go
package cert_test

func TestCertificateValidation(t *testing.T) {
    tests := []struct {
        name    string
        cert    *cert.Certificate
        wantErr error
    }{
        {
            name: "valid certificate",
            cert: &cert.Certificate{
                Subject:   "example.com",
                NotBefore: time.Now().UTC(),
                NotAfter:  time.Now().UTC().AddDate(1, 0, 0),
            },
            wantErr: nil,
        },
        {
            name: "missing subject",
            cert: &cert.Certificate{
                Subject:   "",
                NotBefore: time.Now().UTC(),
                NotAfter:  time.Now().UTC().AddDate(1, 0, 0),
            },
            wantErr: cert.ErrInvalidSubject,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.cert.Validate()
            if !errors.Is(err, tt.wantErr) {
                t.Errorf("got %v, want %v", err, tt.wantErr)
            }
        })
    }
}
```

### Integration Tests

```go
package integration_test

func TestCertificateRepository(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer db.Close()

    repo := postgres.NewCertRepository(db)

    // Test create
    cert := &cert.Certificate{
        Subject:   "test.example.com",
        NotBefore: time.Now().UTC(),
        NotAfter:  time.Now().UTC().AddDate(1, 0, 0),
    }

    err := repo.Create(context.Background(), cert)
    require.NoError(t, err)
    require.NotEmpty(t, cert.ID)

    // Test get
    retrieved, err := repo.Get(context.Background(), cert.ID)
    require.NoError(t, err)
    require.Equal(t, cert.Subject, retrieved.Subject)
}
```

---

## Migration and Evolution

### Architecture Decision Records (ADRs)

Document all significant architectural decisions in `docs/architecture/ADR-XXX-title.md`:

```markdown
# ADR-001: Use PostgreSQL for Certificate Storage

## Status
Accepted

## Context
Need to choose a database for storing certificate metadata, audit logs, and task queue data.

Requirements:
- ACID compliance for audit integrity
- JSON support for flexible metadata
- Strong query capabilities
- Mature ecosystem

## Decision
Use PostgreSQL as the primary database.

## Consequences

Positive:
- ACID compliance ensures audit integrity
- JSONB support for flexible certificate metadata
- Strong indexing and query optimization
- Mature tooling and operational knowledge
- Support for advanced features (full-text search, GIS if needed)

Negative:
- Additional operational complexity vs SQLite
- Need to manage connection pooling
- Higher resource usage than embedded databases

## Alternatives Considered
- SQLite: Simple but limited scalability
- MongoDB: Good for documents but weaker consistency guarantees
- MySQL: Similar to PostgreSQL but weaker JSON support
```

---

## Summary

This architecture guide establishes a solid foundation for the cert-tasks project:

1. **Clean Architecture**: Separation of concerns, testability, flexibility
2. **Domain-Driven Design**: Rich domain models, business logic encapsulation
3. **Security First**: Built-in security, cryptographic best practices
4. **Scalability**: Horizontal scaling, caching, distributed processing
5. **Observability**: Logging, metrics, tracing from day one
6. **Testability**: Comprehensive testing strategy, mocking, isolation
7. **Maintainability**: Clear structure, documented decisions, evolution strategy

Follow these principles and patterns to build a robust, maintainable, and scalable certificate management system.

---

**Next Steps**:

1. Review and approve this architecture
2. Create ADRs for specific technology choices
3. Set up project structure
4. Implement domain models
5. Build infrastructure layer
6. Add comprehensive tests
7. Document API contracts
8. Set up CI/CD pipeline

For questions or clarifications, refer to the CLAUDE.md file or create an architecture discussion issue.
