# Multi-stage Dockerfile for Wartracker bot
# Builder: Debian-based so sqlite3 CGO build succeeds
FROM golang:1.25.1 AS builder

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

# Final runtime image
FROM debian:bookworm-slim
RUN apt-get update && \ 
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /bin/wartracker /usr/local/bin/wartracker

# Working dir and data mount
WORKDIR /app
VOLUME ["/app/data"]

# Default DB path (can be overridden with DB_PATH env)
ENV DB_PATH=/app/data/guild_data.db

ENTRYPOINT ["/usr/local/bin/wartracker"]
