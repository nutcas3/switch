all: build-switch build-simulators build-cli test

# Build main switch application
build-switch:
	@echo "Building GopherSwitch..."
	@go build -o bin/switch cmd/switch/main.go

# Build simulators
build-simulators:
	@echo "Building terminal simulator..."
	@go build -o bin/terminal cmd/simulators/terminal/main.go
	@echo "Building issuer simulator..."
	@go build -o bin/issuer cmd/simulators/issuer/main.go

# Build CLI utilities
build-cli:
	@echo "Building CLI utilities..."
	@go build -o bin/cli cmd/cli/main.go

# Build all components
build: build-switch build-simulators build-cli

# Run main switch
run-switch:
	@echo "Starting GopherSwitch..."
	@./bin/switch

# Run terminal simulator
run-terminal:
	@echo "Starting terminal simulator..."
	@./bin/terminal --host localhost --port 8583 --tps 10

# Run issuer simulator
run-issuer:
	@echo "Starting issuer simulator..."
	@./bin/issuer --port 9001 --approval-rate 0.8

# Run CLI
run-cli:
	@echo "Running CLI..."
	@./bin/cli --help

# Test the application
test:
	@echo "Running tests..."
	@go test ./... -v

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test ./... -v -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	@go test ./tests/... -v -tags=integration

# Load test
load-test:
	@echo "Running load test..."
	@./bin/terminal --host localhost --port 8583 --tps 100 --count 1000

# Create directories
init-dirs:
	@mkdir -p bin logs settlement_files data

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@rm -f *.log

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Update dependencies
update-deps:
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	@golangci-lint run ./...

# Vet code
vet:
	@echo "Vetting code..."
	@go vet ./...

# Security scan
security:
	@echo "Running security scan..."
	@gosec ./...

# Generate mocks
mocks:
	@echo "Generating mocks..."
	@go generate ./...

# Database operations
db-migrate:
	@echo "Running database migrations..."
	@go run cmd/migrate/main.go up

db-rollback:
	@echo "Rolling back database..."
	@go run cmd/migrate/main.go down

db-reset:
	@echo "Resetting database..."
	@go run cmd/migrate/main.go drop
	@go run cmd/migrate/main.go up

# Docker operations
docker-build:
	@echo "Building Docker image..."
	@docker build -t gopherswitch:latest .

docker-run:
	@echo "Running Docker container..."
	@docker-compose up --build

docker-down:
	@echo "Stopping Docker containers..."
	@docker-compose down

docker-logs:
	@echo "Showing Docker logs..."
	@docker-compose logs -f

# Development setup
setup-dev: init-dirs deps
	@echo "Setting up development environment..."
	@cp .env.example .env
	@echo "Development environment ready!"

# Production build
build-prod:
	@echo "Building for production..."
	@CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/switch cmd/switch/main.go
	@CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/terminal cmd/simulators/terminal/main.go
	@CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/issuer cmd/simulators/issuer/main.go
	@CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/cli cmd/cli/main.go

# Development server with hot reload
dev:
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Installing air for hot reload..."; \
		go install github.com/air-verse/air@latest; \
		air; \
	fi

# Generate documentation
docs:
	@echo "Generating documentation..."
	@go doc -all > docs/api.txt
	@swag init -g cmd/switch/main.go

# Performance profiling
profile:
	@echo "Starting with profiling..."
	@go run cmd/switch/main.go -cpuprofile=cpu.prof -memprofile=mem.prof

# Benchmark tests
bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

# Check for vulnerabilities
vuln-check:
	@echo "Checking for vulnerabilities..."
	@govulncheck ./...

# CI pipeline
ci: fmt vet lint security test test-coverage
	@echo "CI pipeline completed successfully!"

# Help target
help:
	@echo "GopherSwitch EFT Switch - Available targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  build-switch     Build main switch application"
	@echo "  build-simulators Build terminal and issuer simulators"
	@echo "  build-cli        Build CLI utilities"
	@echo "  build            Build all components"
	@echo "  build-prod       Build for production"
	@echo ""
	@echo "Run targets:"
	@echo "  run-switch       Run main switch server"
	@echo "  run-terminal     Run terminal simulator"
	@echo "  run-issuer       Run issuer simulator"
	@echo "  run-cli          Run CLI utilities"
	@echo "  dev              Development server with hot reload"
	@echo ""
	@echo "Test targets:"
	@echo "  test             Run unit tests"
	@echo "  test-coverage    Run tests with coverage"
	@echo "  test-integration Run integration tests"
	@echo "  load-test        Run load test"
	@echo "  bench            Run benchmarks"
	@echo ""
	@echo "Database targets:"
	@echo "  db-migrate       Run database migrations"
	@echo "  db-rollback      Rollback database"
	@echo "  db-reset         Reset database"
	@echo ""
	@echo "Docker targets:"
	@echo "  docker-build     Build Docker image"
	@echo "  docker-run       Run with Docker Compose"
	@echo "  docker-down      Stop Docker containers"
	@echo "  docker-logs       Show Docker logs"
	@echo ""
	@echo "Development targets:"
	@echo "  setup-dev        Setup development environment"
	@echo "  fmt              Format code"
	@echo "  lint             Lint code"
	@echo "  vet              Vet code"
	@echo "  security         Run security scan"
	@echo "  vuln-check       Check for vulnerabilities"
	@echo "  deps             Install dependencies"
	@echo "  update-deps      Update dependencies"
	@echo "  clean            Clean build artifacts"
	@echo "  help             Show this help"

# Create DB container (legacy)
docker-run-legacy:
	@if docker compose up --build 2>/dev/null; then \
		: ; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose up --build; \
	fi

# Shutdown DB container (legacy)
docker-down-legacy:
	@if docker compose down 2>/dev/null; then \
		: ; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose down; \
	fi

# Live Reload (legacy)
watch:
	@if command -v air > /dev/null; then \
		air; \
	else \
		read -p "Go's 'air' is not installed. Install it? [Y/n] " choice; \
		if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
			go install github.com/air-verse/air@latest; \
			air; \
		fi; \
	fi

.PHONY: all build-switch build-simulators build-cli build run-switch run-terminal run-issuer run-cli test test-coverage test-integration load-test init-dirs clean deps update-deps fmt lint vet security mocks db-migrate db-rollback db-reset docker-build docker-run docker-down docker-logs setup-dev build-prod dev docs profile bench vuln-check ci help docker-run-legacy docker-down-legacy watch
# Create DB container
docker-run:
	@if docker compose up --build 2>/dev/null; then \
		: ; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose up --build; \
	fi

# Shutdown DB container
docker-down:
	@if docker compose down 2>/dev/null; then \
		: ; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose down; \
	fi

# Test the application
test:
	@echo "Testing..."
	@go test ./... -v

# Clean the binary
clean:
	@echo "Cleaning..."
	@rm -f main

# Live Reload
watch:
	@if command -v air > /dev/null; then \
            air; \
            echo "Watching...";\
        else \
            read -p "Go's 'air' is not installed on your machine. Do you want to install it? [Y/n] " choice; \
            if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
                go install github.com/air-verse/air@latest; \
                air; \
                echo "Watching...";\
            else \
                echo "You chose not to install air. Exiting..."; \
                exit 1; \
            fi; \
        fi

.PHONY: all build run test clean watch
