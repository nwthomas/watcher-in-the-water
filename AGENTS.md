# Repository Guidelines

## Project Structure & Module Organization

`cmd/server` contains the application entrypoint, HTTP health handlers, and process lifecycle wiring. Reusable code lives under `internal/`: `config` loads environment-backed settings, `logger` initializes `slog`, `publicip` fetches and validates the current public IP, `ipstate` persists the last observed IP to disk, `watcher` runs the polling loop and change detection, and `webhook` delivers change notifications. Environment templates live in `env/*.env.example`, startup glue is in `scripts/start.sh`, static assets are in `assets/`, and deployment manifests live in `helm/`. Build output goes to `bin/server`; treat `bin/`, `coverage.out`, and `coverage.html` as generated artifacts.

## Build, Test, and Development Commands

Use `make build` to compile the server to `bin/server`. `make run-local` is the default local entrypoint; it creates `env/local.env` from `env/local.env.example` on first run, then starts the binary through `scripts/start.sh local`. `make run-eng` and `make run-prod` follow the same pattern for alternate environments, but keep the checked-in env templates in sync with those targets before relying on them. Use `make test` to run all Go tests and `make test-coverage` to generate `coverage.out` and `coverage.html`. Use `make fmt` for `go fmt ./...` and `make lint` for `golangci-lint run ./...`. Container flows are `make docker-build` and `make docker-run`.

## Coding Style & Naming Conventions

Follow standard Go formatting and keep code `gofmt`-clean. Use tabs as produced by `go fmt`; do not manually align with spaces. Package names should stay short and lowercase. Exported identifiers use `CamelCase`; unexported helpers use `camelCase`. Prefer descriptive names over abbreviations, except for common Go receiver and error patterns. Keep process bootstrapping, signal handling, and HTTP health wiring in `cmd/server`; move reusable polling, persistence, HTTP client, and notification behavior into `internal/`. Preserve the repository’s current convention of uppercase package-level constants when extending an existing block.

## Testing Guidelines

Tests live beside the code they cover, using the `*_test.go` pattern already present across `cmd/server` and `internal/*`. Name tests `TestXxx` and prefer table-driven cases for parsing, config, and branch-heavy logic. Run `make test` before opening a PR; run `make test-coverage` when changing control flow, persistence behavior, watcher polling, or HTTP request handling. Add coverage for new env handling, logging behavior, IP fetch fallback behavior, webhook delivery branches, and readiness or shutdown paths.

## Commit & Pull Request Guidelines

Use short imperative commit subjects, optionally with a conventional prefix when it adds clarity, for example `Add panic recovery logging` or `fix: handle empty webhook URL lists`. Keep commits focused and reviewable. PRs should include a brief summary, the commands you ran (`make test`, `make fmt`, etc.), linked issues if applicable, and sample logs or manifest snippets when changing runtime behavior, health checks, container settings, or Helm configuration.

## Configuration & Deployment Notes

Do not commit filled-in `.env` files or secrets. Start from the matching file in `env/*.env.example`, and keep `scripts/start.sh`, `Makefile` targets, and committed env examples aligned when adding or renaming environments. Runtime configuration is driven by `PORT`, `LOG_FORMAT`, `LOG_LEVEL`, `STATE_PATH`, `CHECK_INTERVAL`, `IP_URLS`, and `WEBHOOK_URLS`; update env examples, Docker defaults, and Helm values/templates together when any of those change. Persistence matters to behavior: the watcher stores state on disk, and the Helm chart is intentionally configured around a single replica plus a writable volume, so document any change that affects state compatibility or readiness semantics.
