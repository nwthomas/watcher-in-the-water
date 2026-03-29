.PHONY: build run run-local run-eng run-prod test test-coverage lint fmt clean deps docker-build docker-run

build:
	go build -o bin/server ./cmd/server

run: run-local

run-local: build
	@test -f env/local.env || cp env/local.env.example env/local.env
	./scripts/start.sh local

run-eng: build
	@test -f env/eng.env || cp env/eng.env.example env/eng.env
	./scripts/start.sh eng

run-prod: build
	@test -f env/prod.env || cp env/prod.env.example env/prod.env
	./scripts/start.sh prod

test:
	go test ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...

docker-build:
	docker build -t watcher-in-the-water .

docker-run: docker-build
	docker run --rm -p 8080:8080 \
		-e PORT=8080 -e LOG_FORMAT=text \
		-e CHECK_INTERVAL=30s \
		-e STATE_PATH=/var/lib/watcher/state.json \
		-v watcher-state:/var/lib/watcher \
		watcher-in-the-water