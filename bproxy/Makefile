.PHONY: all build clean proto test install

BINARY_DIR=bin
PROTO_DIR=proto
GO=go
PROTOC=protoc

all: proto build

proto:
	@echo "Generating protobuf code..."
	@$(PROTOC) --go_out=. --go_opt=paths=source_relative $(PROTO_DIR)/message.proto

build: proto
	@echo "Building BProxy..."
	@mkdir -p $(BINARY_DIR)
	@$(GO) build -o $(BINARY_DIR)/admin cmd/admin/main.go
	@$(GO) build -o $(BINARY_DIR)/admin-tui cmd/admin-tui/main.go
	@$(GO) build -o $(BINARY_DIR)/agent cmd/agent/main.go
	@echo "Build complete! Binaries in $(BINARY_DIR)/"

clean:
	@echo "Cleaning..."
	@rm -rf $(BINARY_DIR)
	@rm -f $(PROTO_DIR)/*.pb.go
	@echo "Clean complete!"

test:
	@echo "Running tests..."
	@$(GO) test -v ./...

install: build
	@echo "Installing binaries to /usr/local/bin..."
	@sudo cp $(BINARY_DIR)/admin /usr/local/bin/bproxy-admin
	@sudo cp $(BINARY_DIR)/admin-tui /usr/local/bin/bproxy-admin-tui
	@sudo cp $(BINARY_DIR)/agent /usr/local/bin/bproxy-agent
	@echo "Installation complete!"

deps:
	@echo "Installing dependencies..."
	@$(GO) mod download
	@$(GO) mod tidy

run-admin:
	@./$(BINARY_DIR)/admin -addr 0.0.0.0:8443

run-admin-tui:
	@./$(BINARY_DIR)/admin-tui -addr 0.0.0.0:8443

run-agent:
	@./$(BINARY_DIR)/agent -admin 127.0.0.1:8443

help:
	@echo "BProxy Makefile Commands:"
	@echo "  make all          - Generate proto and build all binaries"
	@echo "  make proto        - Generate protobuf code"
	@echo "  make build        - Build all binaries"
	@echo "  make clean        - Remove build artifacts"
	@echo "  make test         - Run tests"
	@echo "  make install      - Install binaries to /usr/local/bin"
	@echo "  make deps         - Download and tidy dependencies"
	@echo "  make run-admin    - Run admin server (CLI mode)"
	@echo "  make run-admin-tui - Run admin server (TUI mode)"
	@echo "  make run-agent    - Run agent"