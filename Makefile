.PHONY: build clean test run

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
BINARY_NAME=bin/bot
MAIN_PATH=cmd/bot/main.go

# Make all
all: test build

# Build the application
build:
	mkdir -p bin
	$(GOBUILD) -o $(BINARY_NAME) $(MAIN_PATH)

# Clean build files
clean:
	$(GOCLEAN)
	rm -rf bin

# Run tests
test:
	$(GOTEST) -v ./...

# Run the application
run: build
	./$(BINARY_NAME)

# Run with auto-reload during development (requires air)
dev:
	air

# Initialize project directories
init:
	mkdir -p cmd/bot
	mkdir -p internal/bot
	mkdir -p internal/config
	mkdir -p internal/models
	mkdir -p internal/storage/drive
	mkdir -p config

# Download dependencies
deps:
	$(GOCMD) mod tidy
	$(GOCMD) mod verify

# Format code
fmt:
	$(GOCMD) fmt ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Help
help:
	@echo "Available commands:"
	@echo "  make build    - Build the bot"
	@echo "  make clean    - Clean build artifacts"
	@echo "  make test     - Run tests"
	@echo "  make run      - Build and run the bot"
	@echo "  make dev      - Run with auto-reload (requires air)"
	@echo "  make init     - Initialize project directories"
	@echo "  make deps     - Download dependencies"
	@echo "  make fmt      - Format code"
	@echo "  make lint     - Run linter (requires golangci-lint)"
	@echo "  make help     - Show this help message"

.DEFAULT_GOAL := help