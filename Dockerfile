# Build stage
FROM golang:1.26-alpine AS builder
WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o server ./cmd/server

# Runtime stage
FROM alpine:3.20
WORKDIR /app

RUN apk add --no-cache ca-certificates wget

RUN adduser -D -u 1000 appuser

COPY --from=builder /app/server .
RUN chown appuser:appuser server

USER appuser

EXPOSE 8080

# Respects PORT at runtime (defaults to 8080)
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD sh -c 'wget -q -O- "http://127.0.0.1:${PORT:-8080}/health/live" || exit 1'

ENTRYPOINT ["./server"]
