# Repository Guidelines

## Project Structure & Module Organization

`cmd/server` contains the application entrypoint. Reusable internal packages live under `internal/`, currently `internal/config` for environment lookups and `internal/logger` for `slog` setup. Environment templates are in `env/*.env.example`, startup glue is in `scripts/start.sh`, and deployment manifests live in `helm/`. Build output goes to `bin/server`; treat `bin/` as generated.

## Build, Test, and Development Commands

Use `make build` to compile the server to `bin/server`. Use `make run-local` for normal local startup; it copies `env/local.env.example` to `env/local.env` if needed, then runs `scripts/start.sh local`. Use `make test` to run all Go tests and `make test-coverage` to generate `coverage.out` and `coverage.html`. Use `make fmt` for `go fmt ./...` and `make lint` for `golangci-lint run ./...`. Container flows are `make docker-build` and `make docker-run`.

## Coding Style & Naming Conventions

Follow standard Go formatting and keep code `gofmt`-clean. Use tabs as produced by `go fmt`; do not manually align with spaces. Package names should stay short and lowercase (`config`, `logger`). Exported identifiers use `CamelCase`; unexported helpers use `camelCase`. Prefer descriptive names over abbreviations, except for common Go receiver and error patterns. Keep HTTP wiring in `cmd/server` and move reusable logic into `internal/`.

## Testing Guidelines

Tests live beside the code they cover, using the `*_test.go` pattern already present in `internal/config` and `internal/logger`. Name tests `TestXxx` and prefer table-driven cases for small behavioral branches. Run `make test` before opening a PR; run `make test-coverage` when changing control flow or configuration behavior. Add coverage for new env handling, logging behavior, or HTTP middleware branches.

## Commit & Pull Request Guidelines

History currently starts with a single `Initial commit`, so use short imperative subjects going forward, for example `Add panic recovery logging` or `Validate local env file`. Keep commits focused and reviewable. PRs should include: a brief summary, the commands you ran (`make test`, `make fmt`, etc.), linked issues if applicable, and screenshots or sample logs when changing runtime behavior or Helm configuration.

## Configuration & Deployment Notes

Do not commit filled-in `.env` files or secrets. Start from `env/local.env.example`, `env/staging.env.example`, or `env/prod.env.example`. When changing runtime configuration, update the matching examples and Helm templates together so local, container, and cluster deployments stay aligned.
