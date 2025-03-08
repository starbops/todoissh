# Build stage
FROM golang:1.24-alpine AS builder

# Install necessary packages for build and copy
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with security flags
RUN CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w" -o todoissh

# Create minimal scratch image with only the binary
FROM scratch

# Add CA certificates for TLS validation
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Add timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary from the builder stage
COPY --from=builder /app/todoissh /todoissh

# No shell in scratch image - safer by default
# No package vulnerabilities - contains only our binary
# No need for user management (no users in scratch)

# Expose SSH port
EXPOSE 2222

# Security metadata labels
LABEL org.opencontainers.image.vendor="ZPCC" \
      org.opencontainers.image.authors="Zespre Chang" \
      org.opencontainers.image.title="TodoiSSH" \
      org.opencontainers.image.description="A secure SSH-based todo application" \
      org.opencontainers.image.version="0.1.0" \
      org.opencontainers.image.created="2025-03-08" \
      org.opencontainers.image.source="https://github.com/starbops/todoissh" \
      io.artifacthub.package.security.scannerDisabled="false" \
      io.artifacthub.package.security.provenance="true"

# Run as non-root (any UID above 10000 is considered non-root)
USER 10001

# Run the application
ENTRYPOINT ["/todoissh"]
CMD [] 