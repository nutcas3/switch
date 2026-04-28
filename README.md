# GopherSwitch

A high-performance Electronic Funds Transfer (EFT) switch built in Go, implementing the ISO8583 messaging standard for payment card authorization, routing, and settlement processing.

## Features

- **ISO8583 Message Processing**: Full support for ISO8583 specification for financial transaction messages
- **Authorization Engine**: Real-time transaction authorization with PIN verification, balance checking, and limit enforcement
- **Transaction Routing**: Intelligent routing of transactions to appropriate issuers/acquirers based on BIN patterns
- **Settlement Processing**: Automated settlement and reconciliation of transactions with Mastercard IPM format
- **Privacy Protection**: Built-in PAN masking using HMAC for audit trails
- **Metrics & Monitoring**: Prometheus metrics integration for TPS, latency, and error tracking
- **Simulators**: Terminal and issuer simulators for testing and development
- **CLI Tools**: Command-line utilities for PAN validation, message generation, and data checking

## Architecture

The project is organized into modular components:

- **auth**: Authorization processing including card validation, PIN verification, and account operations
- **routing**: Transaction routing logic for directing messages to appropriate endpoints
- **settlement**: Batch processing and settlement of transactions
- **privacy**: Data masking and sensitive information protection

## Getting Started

### Prerequisites

- Go 1.26.1 or higher
- PostgreSQL (for database operations)
- Docker (optional, for containerized deployment)

### Installation

1. Clone the repository
2. Install dependencies:
```bash
make deps
```

3. Set up development environment:
```bash
make setup-dev
```

### Running the Application

Build all components:
```bash
make build
```

Start the main switch server:
```bash
make run-switch
```

Start the terminal simulator:
```bash
make run-terminal
```

Start the issuer simulator:
```bash
make run-issuer
```

### Development

Run with hot reload:
```bash
make dev
```

Run tests:
```bash
make test
```

Run tests with coverage:
```bash
make test-coverage
```

### Docker

Build and run with Docker Compose:
```bash
make docker-run
```

Stop Docker containers:
```bash
make docker-down
```

## Makefile Commands

### Build Targets
- `make build` - Build all components (switch, simulators, CLI)
- `make build-switch` - Build main switch application
- `make build-simulators` - Build terminal and issuer simulators
- `make build-cli` - Build CLI utilities
- `make build-prod` - Build for production (Linux binaries)

### Run Targets
- `make run-switch` - Run main switch server
- `make run-terminal` - Run terminal simulator
- `make run-issuer` - Run issuer simulator
- `make run-cli` - Run CLI utilities
- `make dev` - Development server with hot reload

### Test Targets
- `make test` - Run unit tests
- `make test-coverage` - Run tests with coverage report
- `make test-integration` - Run integration tests
- `make load-test` - Run load test with terminal simulator
- `make bench` - Run benchmark tests

### Database Targets
- `make db-migrate` - Run database migrations
- `make db-rollback` - Rollback database
- `make db-reset` - Reset database

### Docker Targets
- `make docker-build` - Build Docker image
- `make docker-run` - Run with Docker Compose
- `make docker-down` - Stop Docker containers
- `make docker-logs` - Show Docker logs

### Development Targets
- `make fmt` - Format code
- `make lint` - Lint code with golangci-lint
- `make vet` - Run go vet
- `make security` - Run security scan with gosec
- `make vuln-check` - Check for vulnerabilities
- `make clean` - Clean build artifacts
- `make help` - Show all available commands

## Project Structure

```
gopherswitch/
├── cmd/                    # Application entry points
│   ├── switch/            # Main switch server
│   ├── cli/               # CLI utilities
│   └── simulators/        # Terminal and issuer simulators
├── internal/
│   ├── modules/           # Business logic modules
│   │   ├── auth/         # Authorization processing
│   │   ├── routing/      # Transaction routing
│   │   ├── settlement/   # Settlement processing
│   │   └── privacy/      # Data protection
│   ├── platform/          # Platform services
│   └── server/            # Server implementations
├── pkg/                   # Shared packages
└── deployments/           # Deployment configurations
```
