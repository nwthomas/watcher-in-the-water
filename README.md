![Watcher in the Water](./assets/watcher-in-the-water-classic.webp)

# Watcher in the Water

A Go server that monitors your home network’s public IP address and detects when it changes. Use it to stay ahead of dynamic DNS drift, firewall rules, or anything else that needs to track the address your ISP assigns you.

## Project Structure

```text
.
├── cmd/server/          # process entrypoint (HTTP health + watcher loop)
├── internal/
│   ├── config/          # environment-backed settings
│   ├── logger/          # slog setup (JSON/text, LOG_LEVEL)
│   ├── ipstate/         # persisted JSON state on disk
│   ├── publicip/        # fetch and validate public IP from HTTP endpoints
│   └── watcher/         # polling loop and change detection
├── helm/                # Kubernetes chart (Deployment, PVC, probes, …)
├── scripts/start.sh     # load env file and exec the binary
├── env/*.env.example    # copy to env/<name>.env (e.g. local, staging, prod; not committed)
├── Dockerfile
├── Makefile
└── .github/workflows/   # tests, image build + push
```

## Build and Run Locally

| Command                       | Description                                                                                    |
| ----------------------------- | ---------------------------------------------------------------------------------------------- |
| `make build`                  | Compile to `bin/server`.                                                                       |
| `make run` / `make run-local` | Ensure `env/local.env` exists (from `env/local.env.example` if needed), then start the server. |
| `make run-eng`                | Loads `env/eng.env` (create it; you can start from `env/staging.env.example`).                 |
| `make run-prod`               | Run with `env/prod.env` (create from `env/prod.env.example` first).                            |

Run the binary without Make after `make build` and a filled-in `env/local.env`:

```bash
set -a && source env/local.env && set +a && ./bin/server
```

Configuration is via environment variables (see `internal/config` and the `env/*.env.example` files). Important keys include `PORT`, `LOG_FORMAT`, `LOG_LEVEL`, `CHECK_INTERVAL`, `STATE_PATH`, `IP_URLS`, and `WEBHOOK_URLS` (comma-separated URLs; each receives a JSON POST when the public IP changes).

## Tests

| Command              | Description                                                        |
| -------------------- | ------------------------------------------------------------------ |
| `make test`          | Run all Go tests.                                                  |
| `make test-coverage` | Tests plus `coverage.out` / `coverage.html`.                       |
| `make lint`          | `golangci-lint` (same family of checks as CI when versions align). |
| `make fmt`           | `go fmt ./...`                                                     |

## Docker

| Command             | Description                                                                                                                         |
| ------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| `make docker-build` | Build image `watcher-in-the-water`.                                                                                                 |
| `make docker-run`   | Run the image with port `8080`, a named volume for state under `/var/lib/watcher`, and a short poll interval for local smoke tests. |

Adjust `-e` flags on `docker run` as needed; `STATE_PATH` should stay under a directory that is writable by the non-root user in the image (see `Dockerfile`).
