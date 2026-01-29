.PHONY: all build proto clean test test-unit test-e2e lint deps init run examples help

GO := go
PROTOC := protoc

# Output directories
BIN_DIR := bin

# Binary name
SERVER_BIN := sprut

all: deps proto build

deps:
	$(GO) mod tidy

proto:
	$(PROTOC) --go_out=. --go_opt=paths=source_relative pkg/message/message.proto

build:
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$(SERVER_BIN) ./cmd/sprut

clean:
	rm -rf $(BIN_DIR)
	rm -f pkg/message/message.pb.go

# Run all tests
test:
	$(GO) test -v -race ./...

# Run only unit tests (fast, no containers)
test-unit:
	$(GO) test -v -race -short ./...

# Run only e2e tests (requires Docker)
test-e2e:
	$(GO) test -v -race ./tests/e2e/... ./pkg/testsprut/...

lint:
	golangci-lint run

# Initialize app directory (XDG config dir)
init: build
	$(BIN_DIR)/$(SERVER_BIN) --init

# Run server (uses XDG config dir)
run: build
	$(BIN_DIR)/$(SERVER_BIN)

# Build examples
examples:
	$(GO) build -o $(BIN_DIR)/simple-client ./examples/simple-client
	$(GO) build -o $(BIN_DIR)/echo-bot ./examples/echo-bot

help:
	@echo "Available targets:"
	@echo "  all        - Download deps, generate proto, build server"
	@echo "  deps       - Download dependencies"
	@echo "  proto      - Generate protobuf code"
	@echo "  build      - Build server binary"
	@echo "  clean      - Remove build artifacts"
	@echo "  test       - Run all tests (requires Docker)"
	@echo "  test-unit  - Run unit tests only (fast, no Docker)"
	@echo "  test-e2e   - Run e2e tests only (requires Docker)"
	@echo "  lint       - Run linter"
	@echo "  init       - Initialize app directory (~/.config/sprut)"
	@echo "  run        - Build and run server"
	@echo "  examples   - Build example clients"
