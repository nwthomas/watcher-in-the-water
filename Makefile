.PHONY: build run run-local run-eng run-prod test test-coverage lint fmt clean deps docker-build docker-run

build:
	go build -o bin/server ./cmd/server

run: run-local

run-local: build
	@test -f env/local.env || cp env/local.env.example env/local.env
	./scripts/start.sh local

run-eng: build
	./scripts/start.sh eng

run-prod: build
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

clean:
	rm -rf bin/

deps:
	go mod download

docker-build:
	docker build -t golang-server-boilerplate .

docker-run: docker-build
	docker run --rm -p 8080:8080 \
		-e PORT=8080 -e LOG_FORMAT=text \
		-e CHECK_INTERVAL=5m \
		-e STATE_PATH=/var/lib/watcher/state.json \
		-v watcher-state:/var/lib/watcher \
		golang-server-boilerplate
