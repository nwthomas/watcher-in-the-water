.PHONY: build
build: ## Build the server binary to bin/server
	go build -o bin/server ./cmd/server

.PHONY: docker-build
docker-build: ## Build the Docker image
	docker build -t watcher-in-the-water .

.PHONY: docker-run
docker-run: docker-build ## Run the Docker image locally on port 8080
	docker run --rm -p 8080:8080 \
		-e PORT=8080 -e LOG_FORMAT=text \
		-e CHECK_INTERVAL=30s \
		-e STATE_PATH=/var/lib/watcher/state.json \
		-v watcher-state:/var/lib/watcher \
		watcher-in-the-water

.PHONY: fmt
fmt: ## Format Go code
	go fmt ./...

.PHONY: help
help: ## Show available targets
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run ./...

.PHONY: run
run: run-local ## Alias for run-local

.PHONY: run-eng
run-eng: build ## Build and run with env/eng.env
	@test -f env/eng.env || cp env/eng.env.example env/eng.env
	./scripts/start.sh eng

.PHONY: run-local
run-local: build ## Build and run with env/local.env
	@test -f env/local.env || cp env/local.env.example env/local.env
	./scripts/start.sh local

.PHONY: run-prod
run-prod: build ## Build and run with env/prod.env
	@test -f env/prod.env || cp env/prod.env.example env/prod.env
	./scripts/start.sh prod

.PHONY: test
test: ## Run all Go tests
	go test ./...

.PHONY: test-coverage
test-coverage: ## Run tests and write coverage.out and coverage.html
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

.PHONY: tidy
tidy: ## Run go mod tidy
	go mod tidy
