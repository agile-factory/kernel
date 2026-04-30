# Agile Factory Boilerplate 🚀

![Go Version](https://img.shields.io/badge/Go-1.25.0-00ADD8?style=flat&logo=go)
![Architecture](https://img.shields.io/badge/Architecture-Clean-brightgreen)
![Status](https://img.shields.io/badge/Status-Active-success)

A highly scalable, production-grade Go backend boilerplate designed for Multi-Tenant SaaS applications. Built strictly following **Clean Architecture** principles and relying on the Go standard library to avoid framework lock-in.

---

## 🏗 Architecture

This project is structured in three distinct layers, ensuring that business logic is completely decoupled from technical implementation details. 

- **Kernel (`internal/kernel`)**: The pure technical foundation. It contains shared, domain-agnostic utilities such as configuration, the HTTP server, robust error handling, and structured logging.
- **Platform (`internal/platform`)**: Reusable SaaS building blocks (e.g., Auth, Tenant management, Billing). These components are identical across different SaaS products.
- **Modules (`internal/modules`)**: The specific business logic of the application. 

*For an in-depth explanation of the architecture, see [RFC-0002](docs/rfcs/0002-Create-Boilerplate-for-Agile-Factory.md).*

---

## ✨ Features

### 🛡 Production-Grade Error Handling
A sophisticated error system (`internal/kernel/errors`) designed for observability and developer experience:
- **Traceable Stacks**: Captures up to 64 stack frames at the exact moment of failure, ensuring you never have to guess where an error originated.
- **Consistent API Responses**: Automatically maps domain errors to HTTP status codes, leaking zero internal stack traces to the public API while preserving them in the logs.
- **Structured Error Chain**: Supports standard Go `errors.Is`/`errors.As` wrapping, preserving the original cause of an error.
- **Sentinel Errors**: Extensive catalog of predefined error codes spanning domains like Auth, Tenant, Billing, and Data.

### 📝 Centralized Fan-Out Logging
A unified logging architecture (`internal/kernel/logger`) utilizing `log/slog` to write structured data concurrently:
- **`stdout`**: Human-readable text in development, JSON in production.
- **`logs/app.log`**: A complete JSON audit trail of all events.
- **`logs/error.log`**: Exclusively captures `ERROR` level and above for immediate incident triage.
- **`logs/server.log`**: Dedicated HTTP access log for tracking API requests, durations, and status codes.

### 🔌 Zero-Magic Standard Library Stack
- **Routing**: Native Go 1.22+ pattern-based HTTP multiplexer (`net/http`).
- **Middleware**: Hand-rolled, lightweight middleware chain for recovery, logging, and tenant extraction.
- **Minimal Dependencies**: Only relying on established standards like `bcrypt`.

---

## 🚀 Getting Started

### Prerequisites
- Go 1.25.0+
- Docker & Docker Compose (Optional, for database and infrastructure)

### Configuration
The application is configured entirely via environment variables. By default, it runs smoothly out of the box. Key variables:
- `APP_ENV`: `development` (default), `staging`, or `production`
- `LOG_LEVEL`: `debug`, `info` (default), `warn`, `error`
- `LOG_DIR`: `logs` (default)
- `PORT`: `8080` (default)

### Running the Application

```bash
# Clone the repository
git clone https://github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate.git
cd Agile-Factory-Boilerplate

# Run the backend
cd backend
go run ./cmd/api/main.go
```

You should see the application start up and log its fan-out destinations to the console and to the `logs/` directory.

### Testing

```bash
cd backend
go test ./... -v
```

---

## 📂 Project Layout

```text
.
├── backend/
│   ├── cmd/api/main.go             # Application entry point
│   ├── internal/
│   │   ├── bootstrap/              # Dependency Injection & Routing
│   │   ├── delivery/http/          # Global HTTP Middlewares & Routers
│   │   ├── kernel/                 # Core technical infra (config, logger, errors)
│   │   ├── platform/               # Reusable SaaS domains (auth, tenant, user)
│   │   └── modules/                # SaaS-specific business logic
├── docs/
│   └── rfcs/                       # Architectural Decision Records & Proposals
├── Makefile                        # Build scripts
└── docker-compose.yml              # Infrastructure orchestration
```

---

## 📄 License
This project is licensed under the terms specified in the `LICENSE` file.
