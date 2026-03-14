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
- Migrations: `goose` SQL migrations (auto-apply on startup or via `cmd/migrate`).
- Logging: standard `log/slog` with JSON output.
- Tracing/metrics: OpenTelemetry SDK + OTLP exporter + Prometheus (+ Grafana/Jaeger in local stack).
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
- goose: SQL migrations tool for Go. Docs: https://github.com/pressly/goose
- slog: structured logging in standard library. Docs: https://pkg.go.dev/log/slog
- OpenTelemetry (Go): traces and telemetry instrumentation. Docs: https://opentelemetry.io/docs/languages/go/
- Prometheus: metrics collection and querying. Docs: https://prometheus.io/docs/introduction/overview/
- Grafana: metrics visualization. Docs: https://grafana.com/docs/grafana/latest/
- Jaeger: distributed tracing backend and UI. Docs: https://www.jaegertracing.io/docs/
- Docker Compose: local infrastructure orchestration. Docs: https://docs.docker.com/compose/

## Project Skeleton

```text
.
|-- api/proto/
|   |-- buf.gen.yaml
|   |-- buf.openapi.gen.yaml
|   |-- buf.yaml
|   |-- booking/v1/
|   |-- catalog/v1/
|   |-- dependency/
|   |   |-- fraud/v1/
|   |   |-- notification/v1/
|   |   `-- payment/v1/
|   |-- google/api/
|   `-- ticketing/v1/
|-- cmd/migrate/
|-- cmd/server/
|-- deploy/observability/
|   |-- grafana/
|   |-- otel/
|   `-- prometheus/
|-- docs/diagrams/
|-- docker-compose.yml
|-- Dockerfile
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
|   |-- repository/memory/
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
- `migrations`: goose migration files for database schema lifecycle.

Architecture references:
- Go project layout reference: https://github.com/golang-standards/project-layout
- Ports and Adapters (Hexagonal): https://alistair.cockburn.us/hexagonal-architecture
- Clean Architecture (overview): https://blog.cleancoder.com/uncle-bob/2011/11/22/Clean-Architecture.html
- Twelve-Factor App principles: https://12factor.net/

## Architecture Diagram (PlantUML)

- Main runtime flow diagram: `docs/diagrams/booking-flow.puml`
- Covers: HTTP gateway -> gRPC transport -> usecase -> repository -> PostgreSQL transactions -> observability.
- Suggested render command (if PlantUML is installed):

```bash
plantuml docs/diagrams/booking-flow.puml
```

## Tooling

Build and run:

```bash
make build
POSTGRES_DSN='postgres://postgres:postgres@localhost:5432/ticketing?sslmode=disable' make run
make migrate # optional: run only migrations against POSTGRES_DSN
```

Goose CLI helpers:

```bash
POSTGRES_DSN='postgres://postgres:postgres@localhost:5432/ticketing?sslmode=disable' make goose-status
POSTGRES_DSN='postgres://postgres:postgres@localhost:5432/ticketing?sslmode=disable' make goose-up
POSTGRES_DSN='postgres://postgres:postgres@localhost:5432/ticketing?sslmode=disable' make goose-down
make goose-create NAME=add_index_to_orders
```

Default runtime addresses:
- gRPC: `:9090` (`GRPC_ADDR`)
- HTTP gateway: `:8080` (`HTTP_ADDR`)

Run on alternative ports (for parallel local runs):

```bash
POSTGRES_DSN='postgres://postgres:postgres@localhost:5432/ticketing?sslmode=disable' make run HTTP_ADDR=:18080 GRPC_ADDR=:19090
```

`POSTGRES_DSN` is required. The service does not start without PostgreSQL.

Run with PostgreSQL backend (auto-applies SQL migrations on startup):

```bash
POSTGRES_DSN='postgres://postgres:postgres@localhost:5432/ticketing?sslmode=disable' make run
```

Optional tracing in local run:

```bash
POSTGRES_DSN='postgres://postgres:postgres@localhost:5432/ticketing?sslmode=disable' OTEL_EXPORTER_OTLP_ENDPOINT=127.0.0.1:4317 OTEL_EXPORTER_OTLP_INSECURE=true make run
```

Full local stack via Docker Compose:

```bash
make docker-up
```

`make docker-up` starts infrastructure only (`postgres`, `otel-collector`, `jaeger`, `prometheus`, `grafana`) to keep app debugging convenient from GoLand.
Run the service locally with:

```bash
POSTGRES_DSN='postgres://postgres:postgres@localhost:5432/ticketing?sslmode=disable' OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 OTEL_EXPORTER_OTLP_INSECURE=true make run
```

Optional: run app + migrator in Docker when needed:

```bash
make docker-up-app
```

Main local URLs in compose mode:
- app portal: http://127.0.0.1:8080/
- Swagger UI: http://127.0.0.1:8080/swagger/
- Prometheus: http://127.0.0.1:9091/
- Grafana (`admin` / `admin`): http://127.0.0.1:3000/
- Jaeger: http://127.0.0.1:16686/

Stop and cleanup compose resources:

```bash
make docker-down
```

Smoke-check endpoints:

```bash
curl -i http://127.0.0.1:8080/
curl -i http://127.0.0.1:8080/swagger/
curl -i http://127.0.0.1:8080/swagger/specs/public.swagger.json
curl -i http://127.0.0.1:8080/healthz
curl -i http://127.0.0.1:8080/metrics
curl -i http://127.0.0.1:8080/debug/pprof/
curl -i http://127.0.0.1:8080/v1/catalog/events
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
- [x] Integrate HTTP gateway.
- [x] Configure Swagger/OpenAPI generation from `proto`.
- [x] Add a single merged Swagger UI for `catalog`, `booking`, and `ticketing` APIs.
- [x] Add developer portal page with links to Swagger, metrics, pprof, health, and sample API routes.

### Stage 3. Business Logic And Data
- [x] Implement 2-3 domain use case sets (for example: `catalog`, `booking`, `ticketing`) via `usecase` + `repository`.
- [x] Integrate PostgreSQL through the repository layer (booking, startup requires PostgreSQL).
- [x] Add and apply database migrations (goose migrations auto-applied when `POSTGRES_DSN` is set).

### Stage 4. Observability And Infrastructure
- [x] Bring up infrastructure via `docker-compose`: `app`, `postgres`, `migrator`, `otel-collector`, `prometheus`, `grafana`, `jaeger/tempo`.
- [x] Configure structured logging.
- [x] Configure tracing (`trace`/`span`).
- [x] Configure baseline metrics (`RPS`, `latency`, `errors`, `DB pool`).
- [ ] Add Grafana dashboard(s) for API, database pool, and runtime metrics.
- [ ] Add Pyroscope for continuous profiling and integrate profiling data into observability workflow.
- [ ] Add `pyroscope` service to `docker-compose` with persistent storage and UI access.
- [ ] Integrate Pyroscope Go SDK in the app (`cpu`, `alloc`, `inuse`, `goroutines`, `mutex`, `block` profiles).
- [ ] Add Pyroscope runtime config (`PYROSCOPE_SERVER_ADDRESS`, app name, env labels).
- [ ] Add Grafana datasource/panels for profiling (or link to Pyroscope UI from the developer portal).

### Stage 5. Testing And CI
- [ ] Cover business logic (`usecase`) with unit tests.
- [ ] Cover repository layer with tests (integration tests with PostgreSQL).
- [ ] Add minimal gRPC end-to-end smoke tests.
- [ ] Configure CI: `lint`, `test`, `race`, migration checks, and codegen checks.

### Stage 6. Documentation And Extensibility
- [ ] Prepare run and setup documentation.
- [x] Document architecture and layer diagram.
- [ ] Document how to add a new service or use case.
- [ ] Prepare extension points for Kafka/Redis (interfaces, config, `docker-compose` profiles).

## Definition Of Done

- [x] At least 2-3 gRPC services and OpenAPI/Swagger are in place.
- [x] PostgreSQL works through migrations and repository layer.
- [ ] Use case and repository layers are covered by tests.
- [x] Logs, metrics, and tracing are available through `docker-compose`.
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
