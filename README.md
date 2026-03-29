![Watcher in the Water](./assets/watcher-in-the-water-classic.webp)

# Watcher in the Water

This repository contains a Go server that monitors IP address assignment changes from your internet service provider (ISP) via dynamic host configuration protocol (DHCP).

You can then have it make callbacks via webhooks to any URLs that you want.

## Project Structure

```text
├── cmd/server/          # Process entrypoint (HTTP health + watcher loop)
├── internal/            #
│   ├── config/          # Environment-backed settings
│   ├── logger/          # Slog setup (JSON/text, LOG_LEVEL)
│   ├── ipstate/         # Persisted JSON state on disk
│   ├── publicip/        # Fetch and validate public IP from HTTP endpoints
│   ├── watcher/         # Polling loop and change detection
│   └── webhook/         # Sends new ip address payload to webhook urls
├── helm/                # Kubernetes chart (Deployment, PVC, probes, …)
├── scripts/start.sh     # Load env file and exec the binary
├── env/*.env.example    # Example .env files to be modified before running server
```

## Build and Run Locally

| Command          | Description                                                            |
| ---------------- | ---------------------------------------------------------------------- |
| `make build`     | Compile to `bin/server`                                                |
| `make run-local` | Ensure `env/local.env` exists (and copy if not) before starning server |
| `make run-eng`   | Ensure `env/eng.env` exists (and copy if not) before starning server   |
| `make run-prod`  | Ensure `env/prod.env` exists (and copy if not) before starning server  |

You can configure the `.env` files in `env/*.env.example` to create corresponding `*.env` files and customize the variables to your heart's content.

## Testing

| Command              | Description                                                       |
| -------------------- | ----------------------------------------------------------------- |
| `make test`          | Run all Go tests                                                  |
| `make test-coverage` | Tests plus `coverage.out` / `coverage.html`                       |
| `make lint`          | `golangci-lint` (same family of checks as CI when versions align) |
| `make fmt`           | `go fmt ./...`                                                    |
