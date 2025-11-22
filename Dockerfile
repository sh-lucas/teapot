# Stage 1: Builder
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install git and ca-certificates
# git might be needed for go mod, ca-certificates for HTTPS
RUN apk add --no-cache git ca-certificates

# Create non-root user
# -D: Don't assign a password
# -u 1001: UID
# -h /app: Home directory
RUN adduser -D -u 1001 -h /app teapot

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build static binary
# -ldflags="-w -s" reduces binary size by stripping debug info
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /teapot main.go

# Ensure /app is owned by teapot (so we can copy it with permissions)
RUN chown -R teapot:teapot /app

# Stage 2: Runtime
FROM scratch

# Copy user/group info
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy CA certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /teapot /teapot

# Copy the app directory (for logs)
# We copy the directory itself to ensure it exists and has the right permissions
COPY --from=builder --chown=1001:1001 /app /app

WORKDIR /app

# Switch to non-root user
USER teapot

# Expose the application port
EXPOSE 9191

# Run the application
CMD ["/teapot"]
