.PHONY: all build proto clean test lint deps

GO := go
PROTOC := protoc

# Output directories
BIN_DIR := bin

# Binary names
SERVER_BIN := gorod

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

test:
	$(GO) test -v -race ./...

lint:
	golangci-lint run

# Generate TLS certificates for testing
certs:
	mkdir -p certs
	openssl req -x509 -newkey rsa:4096 -keyout certs/server.key -out certs/server.crt \
		-days 365 -nodes -subj "/CN=localhost"

# Run server with test config
run: build
	$(BIN_DIR)/$(SERVER_BIN) -config config.yaml

# Build examples
examples:
	$(GO) build -o $(BIN_DIR)/simple-client ./examples/simple-client
	$(GO) build -o $(BIN_DIR)/echo-bot ./examples/echo-bot

help:
	@echo "Available targets:"
	@echo "  all       - Download deps, generate proto, build server"
	@echo "  deps      - Download dependencies"
	@echo "  proto     - Generate protobuf code"
	@echo "  build     - Build server binary"
	@echo "  clean     - Remove build artifacts"
	@echo "  test      - Run tests"
	@echo "  lint      - Run linter"
	@echo "  certs     - Generate test TLS certificates"
	@echo "  run       - Build and run server"
	@echo "  examples  - Build example clients"
