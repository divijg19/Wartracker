# Multi-stage Dockerfile for Wartracker bot
# Builder: Debian-based so sqlite3 CGO build succeeds
# Use the official Go builder image. If your environment cannot pull this tag
# try another official tag (for example `golang:1.25` or `golang:1.25.1`) or a
# pinned digest. The exact available tags depend on the registry mirror.
# Use Microsoft Container Registry devcontainers image to avoid Docker Hub DNS/CDN issues
FROM mcr.microsoft.com/devcontainers/go AS builder

# Install sqlite3 dev and build tools for mattn/go-sqlite3
RUN apt-get update && \
    apt-get install -y --no-install-recommends build-essential libsqlite3-dev ca-certificates git && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Build with CGO enabled for sqlite driver
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -trimpath -o /bin/wartracker ./cmd/bot

## Use a minimal, maintained runtime (distroless) to reduce attack surface.
## Distroless images don't include a shell; we only need CA certs to call Discord.
FROM mcr.microsoft.com/devcontainers/base:bookworm

COPY --from=builder /bin/wartracker /usr/local/bin/wartracker

# Working dir and data mount
WORKDIR /app
VOLUME ["/app/data"]

# Default DB path (can be overridden with DB_PATH env)
ENV DB_PATH=/app/data/guild_data.db

ENTRYPOINT ["/usr/local/bin/wartracker"]
