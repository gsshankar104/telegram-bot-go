.PHONY: build run test clean

# Build the application
build:
	go build -o bin/bot cmd/bot/main.go

# Run the application
run: build
	./bin/bot

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Install dependencies
deps:
	go mod download
	go mod tidy

# Create config from example if it doesn't exist
config:
	@if [ ! -f config/config.yaml ]; then \
		cp config/config.yaml.example config/config.yaml; \
		echo "Created config/config.yaml from example"; \
	else \
		echo "config/config.yaml already exists"; \
	fi

# Initial setup
setup: deps config
	@echo "Setup complete. Don't forget to update config/config.yaml with your bot token"