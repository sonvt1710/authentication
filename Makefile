# Service-specific configuration
APP_NAME := authentication-service
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v1.0.0")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DOCKER_IMAGE := $(APP_NAME):$(VERSION)
LDFLAGS := -X main.Version=$(VERSION) -X main.BuildDate=$(BUILD_DATE) -X main.GitCommit=$(GIT_COMMIT)

# Include shared Makefile from core
# TODO: Clone from Lee-Tech/core if not present to this directory instead of relative path
include ../core/Makefile.common

# Service-specific targets (database, docker, etc.)

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

.PHONY: test-api
test-api: ## Test API endpoints
	@echo "$(GREEN)Testing API endpoints...$(NC)"
	@if [ -f test_authentication.sh ]; then \
		./test_authentication.sh; \
	else \
		echo "$(RED)test_authentication.sh not found$(NC)"; \
	fi


# Database commands
.PHONY: db-create
db-create: ## Create the database
	@echo "$(GREEN)Creating database from $(DATABASE_URL)...$(NC)"
	@psql $(DATABASE_URL) -c "CREATE DATABASE $(shell echo $(DATABASE_URL) | sed -n 's/.*\/\([^?]*\).*/\1/p');" 2>/dev/null || echo "$(YELLOW)Database already exists or error occurred$(NC)"

.PHONY: db-drop
db-drop: ## Drop the database
	@echo "$(RED)Dropping database from $(DATABASE_URL)...$(NC)"
	@read -p "Are you sure? [y/N]: " confirm && [ "$$confirm" = "y" ] || { echo "$(YELLOW)Cancelled$(NC)"; exit 1; }
	@psql $(DATABASE_URL) -c "DROP DATABASE IF EXISTS $(shell echo $(DATABASE_URL) | sed -n 's/.*\/\([^?]*\).*/\1/p');"

.PHONY: db-migrate
db-migrate: ## Run database migrations
	@echo "$(GREEN)Running database migrations...$(NC)"
	@echo "$(YELLOW)Migrations are auto-run on service start$(NC)"

.PHONY: db-reset
db-reset: db-drop db-create ## Reset the database
	@echo "$(GREEN)Database reset complete$(NC)"

# Docker commands
.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "$(GREEN)Building Docker image...$(NC)"
	docker build -t $(DOCKER_IMAGE) .
	@echo "$(GREEN)Docker image built: $(DOCKER_IMAGE)$(NC)"

.PHONY: docker-run
docker-run: ## Run in Docker container
	@echo "$(GREEN)Running in Docker...$(NC)"
	docker run -p 8081:8081 \
		-e DATABASE_URL="$(DATABASE_URL)" \
		-e DB_READ_DSNS="$(DB_READ_DSNS)" \
		-e JWT_SECRET="$(JWT_SECRET)" \
		-e VAULT_ADDR="$(VAULT_ADDR)" \
		-e VAULT_TOKEN="$(VAULT_TOKEN)" \
		--rm $(DOCKER_IMAGE)

.PHONY: docker-compose-up
docker-compose-up: ## Start services with docker-compose
	@echo "$(GREEN)Starting services with docker-compose...$(NC)"
	docker-compose up -d

.PHONY: docker-compose-down
docker-compose-down: ## Stop services with docker-compose
	@echo "$(GREEN)Stopping services...$(NC)"
	docker-compose down

.PHONY: docker-compose-logs
docker-compose-logs: ## View docker-compose logs
	docker-compose logs -f

# Development shortcuts (service-specific)
.PHONY: setup
setup: deps db-create ## Setup development environment
	@echo "$(GREEN)Development environment setup complete$(NC)"
	@echo "$(BLUE)Run 'make dev' to start with hot reload$(NC)"

.PHONY: restart
restart: clean build run ## Clean, rebuild and run

# Show current configuration
.PHONY: info
info: ## Show current configuration
	@echo "$(BLUE)====== Service Info ======$(NC)"
	@echo "App Name:      $(APP_NAME)"
	@echo "Version:       $(VERSION)"
	@echo "Git Commit:    $(GIT_COMMIT)"
	@echo "Build Date:    $(BUILD_DATE)"
	@echo "Database URL:  $(DATABASE_URL)"
	@echo "DB Read DSNs:  $(DB_READ_DSNS)"
	@echo "Vault Addr:    $(VAULT_ADDR)"
	@echo "$(BLUE)==============================$(NC)"