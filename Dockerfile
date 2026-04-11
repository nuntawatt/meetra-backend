# ——— Multi-stage build ———————————————————————————————————————————————————————

# Stage 1: Build
FROM golang:1.22-alpine AS builder

# Install required system deps for CGO sqlite/postgres builds
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Cache module downloads separately from source code
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build a statically linked binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.Version=$(git describe --tags --always 2>/dev/null || echo dev)" \
    -o /app/wego ./cmd/server

# ——————————————————————————————————————————————————————————————————————————————
# Stage 2: Runtime (minimal scratch image)
FROM gcr.io/distroless/static-debian12

WORKDIR /app

# Copy binary, migrations, and default uploads dir
COPY --from=builder /app/wego        /app/wego
COPY --from=builder /app/migrations  /app/migrations

# Run as non-root
USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/app/wego"]
