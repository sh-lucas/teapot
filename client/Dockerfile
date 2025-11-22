# Stage 1: Builder
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy source code
COPY main.go .

# Build static binary
# CGO_ENABLED=0 for static binary
# -ldflags="-w -s" to reduce size
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /teapot main.go

# Stage 2: Runtime
FROM debian:trixie-slim

# Install ca-certificates for HTTPS support
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

# Copy binary to /usr/local/bin so it's in PATH
COPY --from=builder /teapot /usr/local/bin/teapot

# Ensure it's executable (should be by default, but good practice)
RUN chmod +x /usr/local/bin/teapot

# No default entrypoint/cmd, so user can run "teapot ..." or "bash"
