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

# Create data directory. Note that in scratch we can't use RUN, 
# so we need to shift to an intermediate stage
WORKDIR /

# We need to create the directory structure in the intermediate stage
# and copy it to scratch. Let's use a small Alpine image for this.
FROM alpine:3.19 as data-builder

# Install OpenSSH to generate keys
RUN apk add --no-cache openssh-keygen

# Create data directories with proper permissions
RUN mkdir -p /data/todos /data/users && \
    # Create users.json with specific user ownership
    touch /data/users.json && \
    # Set ownership to user 10001
    chown -R 10001:10001 /data && \
    # Set more restrictive permissions
    chmod 700 /data && \
    chmod 700 /data/todos /data/users && \
    chmod 600 /data/users.json

# Generate an SSH host key with proper permissions
RUN ssh-keygen -t rsa -f /data/id_rsa -N "" && \
    chown 10001:10001 /data/id_rsa /data/id_rsa.pub && \
    chmod 600 /data/id_rsa && \
    chmod 644 /data/id_rsa.pub

# Back to our scratch image
FROM scratch

# Copy CA certificates, timezone data, and binary from the builder stage
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/todoissh /todoissh

# Copy the data directory from the data-builder stage
COPY --from=data-builder /data /data

# Set the DATA_DIR environment variable to the absolute path
ENV DATA_DIR=/data

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