![Watcher in the Water](./assets/watcher-in-the-water-classic.webp)

# Watcher in the Water

This repository contains a Go server that monitors IP address assignment changes from your internet service provider (ISP) via dynamic host configuration protocol (DHCP).

When the public IP changes, it sends an email through SMTP with the previous and current IP address.

## Project Structure

```text
├── cmd/server/          # Process entrypoint (HTTP health + watcher loop)
├── internal/            #
│   ├── config/          # Environment-backed settings
│   ├── logger/          # Slog setup (JSON/text, LOG_LEVEL)
│   ├── ipstate/         # Persisted JSON state on disk
│   ├── publicip/        # Fetch and validate public IP from HTTP endpoints
│   ├── watcher/         # Polling loop and change detection
│   └── email/           # Sends IP change email notifications through SMTP
├── helm/                # Kubernetes chart (Deployment, PVC, probes, …)
├── scripts/start.sh     # Load env file and exec the binary
├── env/*.env.example    # Example .env files to be modified before running server
```

## Configuration

Runtime configuration is driven by:

- `PORT`
- `LOG_FORMAT`
- `LOG_LEVEL`
- `STATE_PATH`
- `CHECK_INTERVAL`
- `IP_URLS`
- `EMAIL_HOST_NAME`
- `EMAIL_PASSWORD`
- `EMAIL_PERSONAL_EMAIL`
- `EMAIL_PORT`
- `EMAIL_TLS`
- `EMAIL_USERNAME`

Email configuration is required at startup. If any `EMAIL_*` variable is missing or invalid, the server exits immediately so deployments fail loudly instead of monitoring without notifications.

Configure the SMTP settings with your provider's values:

```text
EMAIL_HOST_NAME=smtp.example.com
EMAIL_PORT=587
EMAIL_TLS=true
EMAIL_USERNAME=sender@example.com
EMAIL_PASSWORD=your-smtp-password
EMAIL_PERSONAL_EMAIL=you@example.com
```

`EMAIL_USERNAME` is the account used to authenticate and send the message. `EMAIL_PASSWORD` is the SMTP password for that account. `EMAIL_PERSONAL_EMAIL` is the destination address.

## Build and Run Locally

| Command          | Description                                                            |
| ---------------- | ---------------------------------------------------------------------- |
| `make build`     | Compile to `bin/server`                                                |
| `make run-local` | Ensure `env/local.env` exists (and copy if not) before starning server |
| `make run-eng`   | Ensure `env/eng.env` exists (and copy if not) before starning server   |
| `make run-prod`  | Ensure `env/prod.env` exists (and copy if not) before starning server  |

You can configure the `.env` files in `env/*.env.example` to create corresponding `*.env` files and customize the variables.

## Kubernetes and Sealed Secrets

The Helm chart reads all email configuration from a Kubernetes Secret:

- Secret name: `watcher-email` by default
- Secret keys: `EMAIL_HOST_NAME`, `EMAIL_PASSWORD`, `EMAIL_PERSONAL_EMAIL`, `EMAIL_PORT`, `EMAIL_TLS`, and `EMAIL_USERNAME`

The intended flow is:

1. Create a `SealedSecret` containing encrypted values for every required `EMAIL_*` key.
2. Let the Sealed Secrets controller materialize the backing Kubernetes `Secret`.
3. Deploy this chart so the pod consumes those keys via `secretKeyRef`.

If the Secret or any required key is missing, Kubernetes will not start the container. If a value is present but empty or invalid, the server exits during startup.

An example manifest is provided at [helm/examples/email.sealedsecret.example.yaml](/Users/nathanthomas/Developer/watcher-in-the-water/helm/examples/email.sealedsecret.example.yaml).

## Testing

| Command              | Description                                                       |
| -------------------- | ----------------------------------------------------------------- |
| `make test`          | Run all Go tests                                                  |
| `make test-coverage` | Tests plus `coverage.out` / `coverage.html`                       |
| `make lint`          | `golangci-lint` (same family of checks as CI when versions align) |
| `make fmt`           | `go fmt ./...`                                                    |
