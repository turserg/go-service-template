.PHONY: build run migrate goose-status goose-up goose-down goose-create docker-up docker-up-app docker-down tools-install .bin-deps proto-update-lock .buf-lint .buf-generate proto-lint proto-generate

APP_NAME := service-template
LOCAL_BIN := $(CURDIR)/bin
BUF := PATH="$(LOCAL_BIN):$$PATH" "$(LOCAL_BIN)/buf"
PROTO_MODULE_DIR := api/proto
BUF_TEMPLATE := $(PROTO_MODULE_DIR)/buf.gen.yaml
BUF_OPENAPI_TEMPLATE := $(PROTO_MODULE_DIR)/buf.openapi.gen.yaml
HTTP_ADDR ?= :8080
GRPC_ADDR ?= :9090
MIGRATIONS_DIR ?= migrations
GOOSE_VERSION ?= v3.27.0

PROTOC_GEN_GO_VERSION ?= v1.28.1
PROTOC_GEN_GO_GRPC_VERSION ?= v1.2.0
PROTOC_GEN_GRPC_GATEWAY_VERSION ?= v2.16.2
PROTOC_GEN_OPENAPIV2_VERSION ?= v2.16.2
BUF_VERSION ?= v1.21.0

build:
	@mkdir -p bin
	go build -o bin/$(APP_NAME) ./cmd/server

run:
	@CGO_ENABLED=0 HTTP_ADDR="$(HTTP_ADDR)" GRPC_ADDR="$(GRPC_ADDR)" go run ./cmd/server/main.go

migrate:
	@CGO_ENABLED=0 go run ./cmd/migrate/main.go

goose-status:
	@test -n "$(POSTGRES_DSN)" || (echo "POSTGRES_DSN is required"; exit 1)
	@go run github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION) -dir $(MIGRATIONS_DIR) postgres "$(POSTGRES_DSN)" status

goose-up:
	@test -n "$(POSTGRES_DSN)" || (echo "POSTGRES_DSN is required"; exit 1)
	@go run github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION) -dir $(MIGRATIONS_DIR) postgres "$(POSTGRES_DSN)" up

goose-down:
	@test -n "$(POSTGRES_DSN)" || (echo "POSTGRES_DSN is required"; exit 1)
	@go run github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION) -dir $(MIGRATIONS_DIR) postgres "$(POSTGRES_DSN)" down

goose-create:
	@test -n "$(NAME)" || (echo "NAME is required. Example: make goose-create NAME=add_index_to_orders"; exit 1)
	@go run github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION) -dir $(MIGRATIONS_DIR) create "$(NAME)" sql

docker-up:
	@docker compose up -d

docker-up-app:
	@docker compose --profile app up --build -d

docker-down:
	@docker compose down --volumes

tools-install: .bin-deps

.bin-deps:
	$(info Installing binary dependencies...)
	@mkdir -p "$(LOCAL_BIN)"
	@if [ -x "$(LOCAL_BIN)/protoc-gen-go" ] && \
		[ -x "$(LOCAL_BIN)/protoc-gen-go-grpc" ] && \
		[ -x "$(LOCAL_BIN)/protoc-gen-grpc-gateway" ] && \
		[ -x "$(LOCAL_BIN)/protoc-gen-openapiv2" ] && \
		[ -x "$(LOCAL_BIN)/buf" ]; then \
		echo "Binary dependencies already installed."; \
	else \
		GOBIN="$(LOCAL_BIN)" go install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION) && \
		GOBIN="$(LOCAL_BIN)" go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$(PROTOC_GEN_GO_GRPC_VERSION) && \
		GOBIN="$(LOCAL_BIN)" go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@$(PROTOC_GEN_GRPC_GATEWAY_VERSION) && \
		GOBIN="$(LOCAL_BIN)" go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@$(PROTOC_GEN_OPENAPIV2_VERSION) && \
		GOBIN="$(LOCAL_BIN)" go install github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION); \
	fi
	@echo "Binary dependencies are ready."

proto-update-lock: .bin-deps
	@$(BUF) mod update "$(PROTO_MODULE_DIR)"

.buf-lint:
	@$(BUF) lint "$(PROTO_MODULE_DIR)" --config "$(PROTO_MODULE_DIR)/buf.yaml"

.buf-generate:
	@$(BUF) generate "$(PROTO_MODULE_DIR)" --template "$(BUF_TEMPLATE)" --config "$(PROTO_MODULE_DIR)/buf.yaml"
	@$(BUF) generate "$(PROTO_MODULE_DIR)" --template "$(BUF_OPENAPI_TEMPLATE)" --config "$(PROTO_MODULE_DIR)/buf.yaml" \
		--path "$(PROTO_MODULE_DIR)/booking/v1" --path "$(PROTO_MODULE_DIR)/catalog/v1" --path "$(PROTO_MODULE_DIR)/ticketing/v1"

proto-lint: .bin-deps .buf-lint

proto-generate: .bin-deps .buf-generate
