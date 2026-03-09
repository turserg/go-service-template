# Go Service Template

A reference Go project focused on best practices, scalable architecture, and maintainable development.

## Goal

Build a service template that includes:
- multiple gRPC services;
- PostgreSQL integration;
- clear business logic and repository layers;
- Swagger/OpenAPI;
- logging, tracing, and metrics;
- infrastructure managed with `docker-compose`;
- extension points for future components (Kafka, Redis, etc.).

## Selected Domain

Use domain: **Event Ticketing Platform**.

This domain gives us clear transactional workflows, concurrent load scenarios, and clean boundaries for multiple gRPC services.

### Core bounded contexts
- `catalog`: events, venues, seat maps, pricing plans.
- `booking`: reservations, checkout, cancellation, order lifecycle.
- `ticketing`: ticket issuance and delivery state.

### gRPC services (internal API)
- `CatalogService`: browse events, seat availability, pricing.
- `BookingService`: reserve seats, checkout order, cancel order.
- `TicketService`: issue tickets, get ticket status, re-send ticket.

### Transaction-heavy use cases
- Reserve seats: lock selected seats, validate availability, create reservation atomically.
- Checkout order: persist payment attempt and order status transition atomically.
- Cancel order: rollback reservation and release seats atomically.

### Concurrency-focused use cases
- High-contention seat reservation with row locking (`SELECT ... FOR UPDATE`) and idempotency keys.
- Parallel execution of independent checks during checkout (for example, fraud + loyalty + pricing validation) via `errgroup`.
- Background worker pool for asynchronous ticket delivery and retry with backoff.

### External gRPC dependencies (mocked locally)
- `PaymentGatewayService` (authorize/capture/refund).
- `FraudCheckService` (risk scoring).
- `NotificationService` (email/push delivery).

Mocks strategy:
- Run lightweight mock gRPC servers in `docker-compose` for local development.
- Support deterministic failure modes (timeouts, unavailable, business reject) via config flags.
- Use in-memory `bufconn` mocks for fast unit/integration tests where possible.

## Locked Base Stack

- Language/runtime: Go `1.26.1`.
- API contracts: Protobuf + `buf` (lint, breaking checks, generation).
- RPC transport: `google.golang.org/grpc`.
- HTTP gateway + docs: `grpc-gateway/v2` + OpenAPI generation from proto annotations.
- Database access: PostgreSQL + `pgx/v5` + `sqlc`.
- Migrations: `golang-migrate`.
- Logging: standard `log/slog` with JSON output.
- Tracing/metrics: OpenTelemetry SDK + OTLP exporter + Prometheus.
- Test stack: Go `testing`, `testify`, `testcontainers-go` (repository integration tests).

## Technology Cheat Sheet

- Go: core language/runtime for business services. Docs: https://go.dev/doc/
- Protobuf: API schema and contract definitions. Docs: https://protobuf.dev/
- Buf: lint, breaking checks, and proto generation workflow. Docs: https://buf.build/docs/
- gRPC (Go): internal RPC transport. Docs: https://grpc.io/docs/languages/go/
- gRPC-Gateway: HTTP/JSON facade over gRPC + OpenAPI generation. Docs: https://grpc-ecosystem.github.io/grpc-gateway/docs/
- PostgreSQL: primary transactional database. Docs: https://www.postgresql.org/docs/
- pgx: PostgreSQL driver/pool for Go. Docs: https://github.com/jackc/pgx
- sqlc: type-safe SQL code generation. Docs: https://docs.sqlc.dev/
- golang-migrate: SQL migrations management. Docs: https://github.com/golang-migrate/migrate
- slog: structured logging in standard library. Docs: https://pkg.go.dev/log/slog
- OpenTelemetry (Go): traces and telemetry instrumentation. Docs: https://opentelemetry.io/docs/languages/go/
- Prometheus: metrics collection and querying. Docs: https://prometheus.io/docs/introduction/overview/
- Docker Compose: local infrastructure orchestration. Docs: https://docs.docker.com/compose/

## Project Skeleton

```text
.
|-- api/proto/
|   |-- buf.gen.yaml
|   |-- buf.yaml
|   |-- booking/v1/
|   |-- catalog/v1/
|   |-- dependency/
|   |   |-- fraud/v1/
|   |   |-- notification/v1/
|   |   `-- payment/v1/
|   |-- google/api/
|   `-- ticketing/v1/
|-- cmd/server/
|-- internal/
|   |-- domain/
|   |   |-- booking/
|   |   |-- catalog/
|   |   `-- ticketing/
|   |-- platform/
|   |   |-- config/
|   |   |-- logger/
|   |   |-- observability/
|   |   `-- postgres/
|   |-- repository/postgres/
|   |-- transport/
|   |   |-- grpc/
|   |   `-- http/
|   `-- usecase/
|       |-- booking/
|       |-- catalog/
|       `-- ticketing/
`-- migrations/
```

## Why This Folder Layout

- `api/proto`: contract-first API source of truth; transport contracts evolve independently from business code.
- `cmd/server`: composition root; app wiring starts here and keeps bootstrap code away from domain logic.
- `internal/domain`: pure domain model and business primitives without transport/storage dependencies.
- `internal/usecase`: application services that orchestrate business flows and transactions.
- `internal/repository`: persistence adapters and DB-specific implementations behind interfaces.
- `internal/transport`: delivery adapters (gRPC, HTTP gateway); this layer translates transport <-> use case.
- `internal/platform`: infrastructural building blocks (config, logging, observability, DB wiring).
- `migrations`: database schema lifecycle in versioned SQL.

Architecture references:
- Go project layout reference: https://github.com/golang-standards/project-layout
- Ports and Adapters (Hexagonal): https://alistair.cockburn.us/hexagonal-architecture
- Clean Architecture (overview): https://blog.cleancoder.com/uncle-bob/2011/11/22/Clean-Architecture.html
- Twelve-Factor App principles: https://12factor.net/

## Tooling

Build and run:

```bash
make build
make run
```

Proto workflow (installs all required generators into `./bin`):

```bash
make tools-install
make proto-lint
make proto-generate
```

Optional (only if remote deps are added to `buf.yaml` in future):

```bash
make proto-update-lock
```

Optional: override tool versions:

```bash
make tools-install BUF_VERSION=v1.21.0 PROTOC_GEN_GO_VERSION=v1.28.1 PROTOC_GEN_GO_GRPC_VERSION=v1.2.0
```

## Roadmap And Progress

> Status format:
> - `[ ]` not started
> - `[x]` done

### Stage 0. Planning
- [x] Align on and document the project roadmap in `README.md`.

### Stage 1. Foundation And Architecture
- [x] Define architectural principles and system boundaries.
- [x] Select and lock the base stack (`grpc`, `grpc-gateway`, `OpenAPI`, `pgx/sqlc`, `migrate`, `slog/zap`, `OpenTelemetry`, `Prometheus`).
- [x] Set up the project skeleton by layers: `transport`, `usecase`, `repository`, `domain`, `internal/platform`.

### Stage 2. API And Transport
- [x] Define `proto` files for multiple gRPC services.
- [x] Configure gRPC code generation.
- [ ] Integrate HTTP gateway.
- [x] Configure Swagger/OpenAPI generation from `proto`.

### Stage 3. Business Logic And Data
- [ ] Implement 2-3 domain use case sets (for example: `catalog`, `booking`, `ticketing`) via `usecase` + `repository`.
- [ ] Integrate PostgreSQL through the repository layer.
- [ ] Add and apply database migrations.

### Stage 4. Observability And Infrastructure
- [ ] Bring up infrastructure via `docker-compose`: `app`, `postgres`, `migrator`, `otel-collector`, `prometheus`, `grafana`, `jaeger/tempo`.
- [ ] Configure structured logging.
- [ ] Configure tracing (`trace`/`span`).
- [ ] Configure baseline metrics (`RPS`, `latency`, `errors`, `DB pool`).

### Stage 5. Testing And CI
- [ ] Cover business logic (`usecase`) with unit tests.
- [ ] Cover repository layer with tests (integration tests with PostgreSQL).
- [ ] Add minimal gRPC end-to-end smoke tests.
- [ ] Configure CI: `lint`, `test`, `race`, migration checks, and codegen checks.

### Stage 6. Documentation And Extensibility
- [ ] Prepare run and setup documentation.
- [ ] Document architecture and layer diagram.
- [ ] Document how to add a new service or use case.
- [ ] Prepare extension points for Kafka/Redis (interfaces, config, `docker-compose` profiles).

## Definition Of Done

- [ ] At least 2-3 gRPC services and OpenAPI/Swagger are in place.
- [ ] PostgreSQL works through migrations and repository layer.
- [ ] Use case and repository layers are covered by tests.
- [ ] Logs, metrics, and tracing are available through `docker-compose`.
- [ ] The project starts with one command and is clear as a reusable template.

## How We Track The Plan

- After finishing a task, change its checkbox from `[ ]` to `[x]`.
- If a new task appears, add it to the relevant stage.
- If it does not fit current stages, add it to `Backlog / Ideas`.
- Use roadmap checkboxes as the single source of progress.

## Backlog / Ideas

Add new items here as they come up during implementation.

- [ ] Add an event module scaffold (for future Kafka integration).
- [ ] Add a caching layer scaffold (for future Redis integration).
