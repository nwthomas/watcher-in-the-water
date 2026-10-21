![Watcher in the Water](./assets/watcher-in-the-water-classic.webp)

# Watcher in the Water

This repository contains a Go server that monitors IP address assignment changes from your internet service provider (ISP) via dynamic host configuration protocol (DHCP).

You can then have it make callbacks via webhooks to any URLs that you want which will include the new IP address in the request body, and optionally send you an email (for example through iCloud Mail).

## Project Structure

```text
├── cmd/server/          # Process entrypoint (HTTP health + watcher loop)
├── internal/            #
│   ├── config/          # Environment-backed settings
│   ├── logger/          # Slog setup (JSON/text, LOG_LEVEL)
│   ├── ipstate/         # Persisted JSON state on disk
│   ├── publicip/        # Fetch and validate public IP from HTTP endpoints
│   ├── email/           # Sends an SMTP email on ip address change
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

You can configure the `.env` files in `env/*.env.example` to create corresponding `*.env` files and customize the variables.

## Email Notifications

When the public IP changes, the server can send an email to `EMAIL_PERSONAL_EMAIL` (it is used as both sender and recipient). Email is off unless these variables are set; if only some are set, the server logs a warning at startup and leaves email off.

| Variable               | Description                                                              |
| ---------------------- | ------------------------------------------------------------------------ |
| `EMAIL_HOST_NAME`      | SMTP host, e.g. `smtp.mail.me.com` for iCloud                            |
| `EMAIL_PORT`           | SMTP port (default `587`)                                                |
| `EMAIL_TLS`            | `true` (default) requires STARTTLS; `false` only works against localhost |
| `EMAIL_USERNAME`       | SMTP login; for iCloud, your full iCloud address                         |
| `EMAIL_PASSWORD`       | For iCloud, an app-specific password (not your Apple Account password)   |
| `EMAIL_PERSONAL_EMAIL` | Sender and recipient; for iCloud, your address or one of its aliases     |

Delivery is best-effort like webhooks: failures are logged and not retried.

### Kubernetes with Sealed Secrets

Put the six `EMAIL_*` values in a local `KEY=value` file that you do not commit (e.g. `email.env`), then seal it for the namespace you deploy to:

```bash
kubectl create secret generic watcher-in-the-water-email --namespace <namespace> --from-env-file=email.env --dry-run=client -o yaml | kubeseal --format yaml > watcher-in-the-water-email.sealed.yaml
```

Apply the SealedSecret, then point the chart at the Secret it unseals into:

```yaml
extraEnvFrom:
  - secretRef:
      name: watcher-in-the-water-email
```

## Testing

| Command              | Description                                                       |
| -------------------- | ----------------------------------------------------------------- |
| `make test`          | Run all Go tests                                                  |
| `make test-coverage` | Tests plus `coverage.out` / `coverage.html`                       |
| `make lint`          | `golangci-lint` (same family of checks as CI when versions align) |
| `make fmt`           | `go fmt ./...`                                                    |
