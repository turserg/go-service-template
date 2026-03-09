.PHONY: build run tools-install .bin-deps proto-update-lock .buf-lint .buf-generate proto-lint proto-generate

APP_NAME := service-template
LOCAL_BIN := $(CURDIR)/bin
BUF := PATH="$(LOCAL_BIN):$$PATH" "$(LOCAL_BIN)/buf"
PROTO_MODULE_DIR := api/proto
BUF_TEMPLATE := $(PROTO_MODULE_DIR)/buf.gen.yaml

PROTOC_GEN_GO_VERSION ?= v1.28.1
PROTOC_GEN_GO_GRPC_VERSION ?= v1.2.0
PROTOC_GEN_GRPC_GATEWAY_VERSION ?= v2.16.2
PROTOC_GEN_OPENAPIV2_VERSION ?= v2.16.2
BUF_VERSION ?= v1.21.0

build:
	@mkdir -p bin
	go build -o bin/$(APP_NAME) ./cmd/server

run:
	@CGO_ENABLED=0 go run ./cmd/server/main.go

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

proto-lint: .bin-deps .buf-lint

proto-generate: .bin-deps .buf-generate
