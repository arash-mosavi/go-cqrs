# CQRS Production-Grade Makefile
# Provides convenient commands for development, building, testing, and deployment

.PHONY: help build test benchmark clean docker deploy k8s monitoring docs dev

# Default target
.DEFAULT_GOAL := help

# Variables
APP_NAME := cqrs-app
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
DOCKER_REGISTRY := ghcr.io/your-org
IMAGE_NAME := $(DOCKER_REGISTRY)/$(APP_NAME)
ENVIRONMENT := production

# Go variables
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt
BINARY_NAME := $(APP_NAME)

# Build flags
LDFLAGS := -ldflags="-w -s -X main.version=$(VERSION) -X main.buildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"
BUILD_FLAGS := -a -installsuffix cgo $(LDFLAGS)

## help: Show this help message
help:
	@echo "CQRS Production-Grade Application"
	@echo "================================="
	@echo ""
	@echo "Available commands:"
	@echo ""
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
	@echo ""
	@echo "Examples:"
	@echo "  make dev                    # Start development environment"
	@echo "  make test                   # Run all tests"
	@echo "  make docker                 # Build Docker image"
	@echo "  make deploy ENV=staging     # Deploy to staging"
	@echo ""

## dev: Start development environment with hot reload
dev:
	@echo "🚀 Starting development environment..."
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "Installing air for hot reload..."; \
		go install github.com/cosmtrek/air@latest; \
		air; \
	fi

## build: Build the application binary
build:
	@echo "🔨 Building application..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_NAME) ./cmd/complete-production-demo/
	@echo "✅ Build completed: $(BINARY_NAME)"

## build-all: Build all demo applications
build-all:
	@echo "🔨 Building all applications..."
	@for dir in cmd/*/; do \
		if [ -f "$$dir/main.go" ]; then \
			name=$$(basename "$$dir"); \
			echo "Building $$name..."; \
			CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o "bin/$$name" "./$$dir/"; \
		fi \
	done
	@echo "✅ All builds completed"

## test: Run all tests
test:
	@echo "🧪 Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@echo "✅ Tests completed"

## test-coverage: Run tests with coverage report
test-coverage: test
	@echo "📊 Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## benchmark: Run performance benchmarks
benchmark:
	@echo "⚡ Running benchmarks..."
	$(GOTEST) -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof ./...
	@echo "✅ Benchmarks completed"

## benchmark-suite: Run comprehensive benchmark suite
benchmark-suite:
	@echo "🏃 Running comprehensive benchmark suite..."
	$(GOBUILD) -o benchmark-suite ./cmd/benchmark-suite/
	./benchmark-suite
	@rm -f benchmark-suite
	@echo "✅ Benchmark suite completed"

## lint: Run code linting
lint:
	@echo "🔍 Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.54.2; \
		golangci-lint run; \
	fi
	@echo "✅ Linting completed"

## format: Format Go code
format:
	@echo "🎨 Formatting code..."
	$(GOFMT) -s -w .
	$(GOCMD) mod tidy
	@echo "✅ Code formatted"

## security: Run security checks
security:
	@echo "🔒 Running security checks..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "Installing gosec..."; \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
		gosec ./...; \
	fi
	@echo "✅ Security checks completed"

## clean: Clean build artifacts
clean:
	@echo "🧹 Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -f cpu.prof mem.prof
	rm -f benchmark_results_*.txt
	docker system prune -f --volumes
	@echo "✅ Cleanup completed"

## deps: Download and verify dependencies
deps:
	@echo "📦 Managing dependencies..."
	$(GOMOD) download
	$(GOMOD) verify
	$(GOMOD) tidy
	@echo "✅ Dependencies updated"

## docker: Build Docker image
docker:
	@echo "🐳 Building Docker image..."
	docker build -t $(IMAGE_NAME):$(VERSION) .
	docker tag $(IMAGE_NAME):$(VERSION) $(IMAGE_NAME):latest
	@echo "✅ Docker image built: $(IMAGE_NAME):$(VERSION)"

## docker-push: Push Docker image to registry
docker-push: docker
	@echo "📤 Pushing Docker image..."
	docker push $(IMAGE_NAME):$(VERSION)
	docker push $(IMAGE_NAME):latest
	@echo "✅ Docker image pushed"

## compose-up: Start services with Docker Compose
compose-up:
	@echo "🚀 Starting services with Docker Compose..."
	docker-compose up -d
	@echo "✅ Services started"

## compose-down: Stop services with Docker Compose
compose-down:
	@echo "🛑 Stopping services with Docker Compose..."
	docker-compose down
	@echo "✅ Services stopped"

## compose-logs: Show Docker Compose logs
compose-logs:
	docker-compose logs -f

## k8s-deploy: Deploy to Kubernetes
k8s-deploy:
	@echo "☸️  Deploying to Kubernetes ($(ENVIRONMENT))..."
	kubectl apply -f k8s/
	kubectl rollout status deployment/$(APP_NAME) -n cqrs-$(ENVIRONMENT) --timeout=300s
	@echo "✅ Kubernetes deployment completed"

## k8s-status: Check Kubernetes deployment status
k8s-status:
	@echo "📊 Checking Kubernetes status..."
	kubectl get all -n cqrs-$(ENVIRONMENT)
	@echo ""
	kubectl top pods -n cqrs-$(ENVIRONMENT) 2>/dev/null || echo "Metrics server not available"

## k8s-logs: Show Kubernetes logs
k8s-logs:
	kubectl logs deployment/$(APP_NAME) -n cqrs-$(ENVIRONMENT) -f --tail=100

## deploy: Full deployment pipeline
deploy: test docker-push k8s-deploy
	@echo "🎉 Deployment completed successfully!"

## deploy-staging: Deploy to staging environment
deploy-staging:
	@$(MAKE) deploy ENVIRONMENT=staging

## deploy-production: Deploy to production environment
deploy-production:
	@$(MAKE) deploy ENVIRONMENT=production

## monitoring: Deploy monitoring stack
monitoring:
	@echo "📊 Deploying monitoring stack..."
	kubectl apply -f monitoring/
	@echo "✅ Monitoring stack deployed"

## docs: Generate documentation
docs:
	@echo "📚 Generating documentation..."
	@if command -v godoc >/dev/null 2>&1; then \
		echo "Starting godoc server at http://localhost:6060"; \
		godoc -http=:6060; \
	else \
		echo "Installing godoc..."; \
		go install golang.org/x/tools/cmd/godoc@latest; \
		echo "Starting godoc server at http://localhost:6060"; \
		godoc -http=:6060; \
	fi

## install: Install development tools
install:
	@echo "🔧 Installing development tools..."
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	go install golang.org/x/tools/cmd/godoc@latest
	@echo "✅ Development tools installed"

## init: Initialize development environment
init: install deps
	@echo "🎯 Initializing development environment..."
	@mkdir -p bin logs
	@echo "✅ Development environment initialized"

## ci: Run CI pipeline locally
ci: format lint security test benchmark
	@echo "✅ CI pipeline completed successfully"

## release: Create a new release
release:
	@echo "🚀 Creating release..."
	@if [ -z "$(TAG)" ]; then \
		echo "❌ TAG is required. Usage: make release TAG=v1.0.0"; \
		exit 1; \
	fi
	git tag -a $(TAG) -m "Release $(TAG)"
	git push origin $(TAG)
	@echo "✅ Release $(TAG) created"

## health: Check application health
health:
	@echo "🏥 Checking application health..."
	@curl -f http://localhost:8080/health || echo "❌ Health check failed"
	@curl -f http://localhost:8080/ready || echo "❌ Readiness check failed"

## load-test: Run load testing
load-test:
	@echo "🔥 Running load tests..."
	@if command -v hey >/dev/null 2>&1; then \
		hey -n 10000 -c 100 http://localhost:8080/health; \
	else \
		echo "Installing hey..."; \
		go install github.com/rakyll/hey@latest; \
		hey -n 10000 -c 100 http://localhost:8080/health; \
	fi

## profile: Run performance profiling
profile:
	@echo "📈 Running performance profiling..."
	$(GOTEST) -cpuprofile=cpu.prof -memprofile=mem.prof -bench=. ./...
	@echo "Profiles generated: cpu.prof, mem.prof"
	@echo "View with: go tool pprof cpu.prof"

## version: Show version information
version:
	@echo "Version: $(VERSION)"
	@echo "Image: $(IMAGE_NAME):$(VERSION)"
	@echo "Go version: $$(go version)"
	@echo "Docker version: $$(docker --version 2>/dev/null || echo 'Not installed')"
	@echo "Kubectl version: $$(kubectl version --client --short 2>/dev/null || echo 'Not installed')"

# Advanced targets for specific scenarios

## debug: Build and run with debugging enabled
debug:
	@echo "🐛 Building with debug info..."
	$(GOBUILD) -gcflags="all=-N -l" -o $(BINARY_NAME)-debug ./cmd/complete-production-demo/
	@echo "Debug binary: $(BINARY_NAME)-debug"

## race: Run tests with race detection
race:
	@echo "🏃‍♂️ Running race detection..."
	$(GOTEST) -race -short ./...

## integration: Run integration tests
integration:
	@echo "🔗 Running integration tests..."
	$(GOTEST) -tags=integration -v ./...

## e2e: Run end-to-end tests
e2e:
	@echo "🎭 Running end-to-end tests..."
	$(GOTEST) -tags=e2e -v ./...

## stress: Run stress tests
stress:
	@echo "💪 Running stress tests..."
	$(GOTEST) -tags=stress -timeout=30m -v ./...

# Demo targets
.PHONY: demo demo-basic demo-simplified demo-production examples

demo: demo-basic demo-simplified demo-production  ## Run all demo examples

demo-basic:  ## Run the basic CQRS demo
	@echo "Running Basic CQRS Demo..."
	@go run cmd/demo/main.go

demo-simplified:  ## Run the simplified middleware demo
	@echo "Running Simplified Production Demo..."
	@go run cmd/simplified-demo/main.go

demo-production:  ## Run the complete production demo
	@echo "Running Complete Production Demo..."
	@go run cmd/complete-production-demo/main.go

examples: demo  ## Alias for demo target
