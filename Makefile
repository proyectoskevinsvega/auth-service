.PHONY: help build run test test-unit test-integration test-all bench clean keys migrate proto swagger load-test docker fmt lint pre-commit

# Load .env file if it exists
ifneq (,$(wildcard .env))
    include .env
    export
endif

help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the auth service binary
	@echo "Building auth-service..."
	@CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/auth-service ./cmd/auth-service
	@echo "Binary created at bin/auth-service"

build-linux: ## Build for Linux AMD64
	@echo "Building for Linux AMD64..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/auth-service-linux-amd64 ./cmd/auth-service

build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/auth-service-linux-amd64 ./cmd/auth-service
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o bin/auth-service-linux-arm64 ./cmd/auth-service
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o bin/auth-service-darwin-amd64 ./cmd/auth-service
	@CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o bin/auth-service-darwin-arm64 ./cmd/auth-service
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o bin/auth-service-windows-amd64.exe ./cmd/auth-service
	@echo "All binaries created in bin/"

run: ## Run the auth service
	go run ./cmd/auth-service/main.go

test: ## Run all tests (unit + integration)
	@echo "Running all tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out

test-unit: ## Run unit tests only
	@echo "Running unit tests..."
	@go test -v -race -coverprofile=coverage-unit.out ./internal/usecase/...
	@go tool cover -func=coverage-unit.out

test-integration: ## Run integration tests only
	@echo "Running integration tests..."
	@go test -v -race -coverprofile=coverage-integration.out ./tests/integration/...

test-all: test-unit test-integration ## Run unit and integration tests separately

coverage: ## Generate HTML coverage report
	@echo "Generating coverage report..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"

bench: ## Run benchmarks
	go test -bench=. -benchmem ./tests/benchmarks

keys: ## Generate RSA key pair for JWT signing
	bash scripts/generate_rsa_keys.sh

proto: ## Generate Go code from protobuf definitions
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		internal/adapters/grpc/proto/auth.proto

install-swag: ## Install swag CLI tool for Swagger documentation
	@echo "Installing swag..."
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "swag installed successfully"

swagger: ## Generate Swagger documentation
	@echo "Generating Swagger docs..."
	@swag init -g cmd/auth-service/main.go -o docs --parseDependency --parseInternal
	@echo "Swagger docs generated at docs/"

swagger-fmt: ## Format Swagger comments
	@echo "Formatting Swagger comments..."
	@swag fmt

install-migrate: ## Install golang-migrate CLI tool
	@echo "Installing golang-migrate..."
	@go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "golang-migrate installed successfully"

migrate-up: ## Run database migrations (up)
	@echo "Running migrations up..."
	@if [ -z "$$POSTGRES_HOST" ]; then \
		echo "Error: POSTGRES_HOST environment variable not set"; \
		exit 1; \
	fi
	@if [ -z "$$POSTGRES_USER" ]; then \
		echo "Error: POSTGRES_USER environment variable not set"; \
		exit 1; \
	fi
	@if [ -z "$$POSTGRES_NAME" ]; then \
		echo "Error: POSTGRES_NAME environment variable not set"; \
		exit 1; \
	fi
	@POSTGRES_URL="postgresql://$$POSTGRES_USER:$$POSTGRES_PASSWORD@$$POSTGRES_HOST:$${POSTGRES_PORT:-5432}/$$POSTGRES_NAME?sslmode=$${POSTGRES_SSLMODE:-disable}"; \
	migrate -database "$$POSTGRES_URL" -path internal/adapters/postgres/migrations up
	@echo "Migrations completed successfully"

migrate-down: ## Rollback database migrations (down)
	@echo "Rolling back migrations..."
	@if [ -z "$$POSTGRES_HOST" ]; then \
		echo "Error: POSTGRES_HOST environment variable not set"; \
		exit 1; \
	fi
	@if [ -z "$$POSTGRES_USER" ]; then \
		echo "Error: POSTGRES_USER environment variable not set"; \
		exit 1; \
	fi
	@if [ -z "$$POSTGRES_NAME" ]; then \
		echo "Error: POSTGRES_NAME environment variable not set"; \
		exit 1; \
	fi
	@POSTGRES_URL="postgresql://$$POSTGRES_USER:$$POSTGRES_PASSWORD@$$POSTGRES_HOST:$${POSTGRES_PORT:-5432}/$$POSTGRES_NAME?sslmode=$${POSTGRES_SSLMODE:-disable}"; \
	migrate -database "$$POSTGRES_URL" -path internal/adapters/postgres/migrations down
	@echo "Rollback completed successfully"

migrate-force: ## Force migration version (usage: make migrate-force VERSION=1)
	@if [ -z "$$POSTGRES_HOST" ]; then \
		echo "Error: POSTGRES_HOST environment variable not set"; \
		exit 1; \
	fi
	@if [ -z "$$POSTGRES_USER" ]; then \
		echo "Error: POSTGRES_USER environment variable not set"; \
		exit 1; \
	fi
	@if [ -z "$$POSTGRES_NAME" ]; then \
		echo "Error: POSTGRES_NAME environment variable not set"; \
		exit 1; \
	fi
	@if [ -z "$$VERSION" ]; then \
		echo "Error: VERSION not specified. Usage: make migrate-force VERSION=1"; \
		exit 1; \
	fi
	@POSTGRES_URL="postgresql://$$POSTGRES_USER:$$POSTGRES_PASSWORD@$$POSTGRES_HOST:$${POSTGRES_PORT:-5432}/$$POSTGRES_NAME?sslmode=$${POSTGRES_SSLMODE:-disable}"; \
	migrate -database "$$POSTGRES_URL" -path internal/adapters/postgres/migrations force $$VERSION

migrate-version: ## Show current migration version
	@if [ -z "$$POSTGRES_HOST" ]; then \
		echo "Error: POSTGRES_HOST environment variable not set"; \
		exit 1; \
	fi
	@if [ -z "$$POSTGRES_USER" ]; then \
		echo "Error: POSTGRES_USER environment variable not set"; \
		exit 1; \
	fi
	@if [ -z "$$POSTGRES_NAME" ]; then \
		echo "Error: POSTGRES_NAME environment variable not set"; \
		exit 1; \
	fi
	@POSTGRES_URL="postgresql://$$POSTGRES_USER:$$POSTGRES_PASSWORD@$$POSTGRES_HOST:$${POSTGRES_PORT:-5432}/$$POSTGRES_NAME?sslmode=$${POSTGRES_SSLMODE:-disable}"; \
	migrate -database "$$POSTGRES_URL" -path internal/adapters/postgres/migrations version

clean: ## Clean build artifacts
	rm -rf bin/
	go clean

deps: ## Download dependencies
	go mod download
	go mod tidy

fmt: ## Format Go code
	@echo "Formatting code..."
	@gofmt -s -w .
	@goimports -w . 2>/dev/null || go install golang.org/x/tools/cmd/goimports@latest && goimports -w .

lint: ## Run linters
	@echo "Running linters..."
	@golangci-lint run --timeout=5m

lint-fix: ## Run linters and auto-fix issues
	@echo "Running linters with auto-fix..."
	@golangci-lint run --fix --timeout=5m

pre-commit: fmt lint test-unit build ## Run all pre-commit checks
	@echo "✅ All pre-commit checks passed!"

# Docker commands
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t auth-service:latest .

docker-build-prod: ## Build production Docker image with version
	@echo "Building production Docker image..."
	@docker build \
		--build-arg VERSION=$$(git describe --tags --always) \
		--build-arg BUILD_TIME=$$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
		-t auth-service:$$(git describe --tags --always) \
		-t auth-service:latest \
		.

docker-run: ## Run in Docker with docker-compose
	@echo "Starting services with docker-compose..."
	@docker-compose up -d

docker-logs: ## Show Docker logs
	@docker-compose logs -f auth-service

docker-down: ## Stop Docker containers
	@echo "Stopping Docker containers..."
	@docker-compose down

docker-clean: ## Remove Docker containers and volumes
	@echo "Cleaning Docker containers and volumes..."
	@docker-compose down -v

docker-restart: ## Restart Docker services
	@docker-compose restart auth-service

docker-ps: ## Show running containers
	@docker-compose ps

docker-shell: ## Open shell in running container
	@docker-compose exec auth-service sh

systemd-install: ## Install as systemd service (requires root)
	@echo "Installing auth-service as systemd service..."
	@cd deployments/systemd && sudo bash install.sh

systemd-uninstall: ## Uninstall systemd service (requires root)
	@echo "Uninstalling auth-service systemd service..."
	@cd deployments/systemd && sudo bash uninstall.sh

systemd-status: ## Show systemd service status
	@sudo systemctl status auth-service

systemd-logs: ## Show systemd service logs
	@sudo journalctl -u auth-service -f

# k6 Load Testing
install-k6: ## Install k6 (requires chocolatey on Windows, brew on macOS)
	@echo "Installing k6..."
	@if command -v choco > /dev/null 2>&1; then \
		choco install k6; \
	elif command -v brew > /dev/null 2>&1; then \
		brew install k6; \
	elif command -v apt-get > /dev/null 2>&1; then \
		sudo gpg -k; \
		sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69; \
		echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list; \
		sudo apt-get update; \
		sudo apt-get install k6; \
	else \
		echo "Please install k6 manually from https://github.com/grafana/k6/releases"; \
		exit 1; \
	fi
	@echo "k6 installed successfully"

load-test-validate: ## Run k6 validate token test (cache hit performance)
	@cd tests/k6 && k6 run validate-token.js

load-test-login: ## Run k6 login test
	@cd tests/k6 && k6 run login.js

load-test-register: ## Run k6 register test
	@cd tests/k6 && k6 run register.js

load-test-mixed: ## Run k6 mixed load test (realistic scenario)
	@cd tests/k6 && k6 run mixed-load.js

load-test-all: ## Run all k6 tests
	@echo "Running all k6 load tests..."
	@if command -v bash > /dev/null 2>&1; then \
		cd tests/k6 && bash run-all-tests.sh; \
	else \
		cd tests/k6 && run-all-tests.bat; \
	fi

# CI/CD helpers
ci-lint: ## Run CI linting (same as CI pipeline)
	@echo "Running CI linting checks..."
	@golangci-lint run --timeout=5m
	@gofmt -l . | grep . && echo "Code is not formatted. Run 'make fmt'" && exit 1 || echo "✅ Code is formatted"
	@go mod tidy && git diff --exit-code go.mod go.sum || (echo "go.mod or go.sum is not tidy. Run 'go mod tidy'" && exit 1)

ci-test: ## Run CI tests (unit + integration)
	@echo "Running CI tests..."
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

ci-build: ## Build binary (CI style)
	@echo "Building binary (CI mode)..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags="-s -w -X main.version=$$(git describe --tags --always) -X main.buildTime=$$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
		-o auth-service \
		./cmd/auth-service

# Security scans
security-scan: ## Run security scans (gosec + govulncheck)
	@echo "Running security scans..."
	@command -v gosec > /dev/null || (echo "Installing gosec..." && go install github.com/securego/gosec/v2/cmd/gosec@latest)
	@command -v govulncheck > /dev/null || (echo "Installing govulncheck..." && go install golang.org/x/vuln/cmd/govulncheck@latest)
	@echo "Running gosec..."
	@gosec -fmt=text ./...
	@echo "Running govulncheck..."
	@govulncheck ./...

# Git helpers
git-tag: ## Create and push a new tag (usage: make git-tag VERSION=v1.0.0)
	@if [ -z "$$VERSION" ]; then \
		echo "Error: VERSION not specified. Usage: make git-tag VERSION=v1.0.0"; \
		exit 1; \
	fi
	@echo "Creating tag $$VERSION..."
	@git tag -a $$VERSION -m "Release $$VERSION"
	@git push origin $$VERSION
	@echo "✅ Tag $$VERSION created and pushed"

# Development helpers
dev-setup: deps keys ## Setup development environment
	@echo "Setting up development environment..."
	@command -v golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@command -v swag > /dev/null || make install-swag
	@command -v migrate > /dev/null || make install-migrate
	@echo "✅ Development environment ready!"

dev-db: ## Start only database services
	@docker-compose up -d postgres redis

dev-reset: docker-clean dev-setup docker-run migrate-up ## Reset development environment
	@echo "✅ Development environment reset complete!"

.DEFAULT_GOAL := help
