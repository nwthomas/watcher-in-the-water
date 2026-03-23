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

RUN adduser -D -u 1000 appuser

COPY --from=builder /app/server .
RUN chown appuser:appuser server

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget -q -O- http://localhost:8080/health/live || exit 1

ENTRYPOINT ["./server"]
