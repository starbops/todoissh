# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o todoissh

# Final stage only - we're using pre-built binary
FROM alpine:3.19

# Create a non-root user
RUN adduser -D -h /app todoapp

WORKDIR /app

# Copy the pre-built binary
COPY bin/todoissh .

# Set ownership
RUN chown -R todoapp:todoapp /app

# Switch to non-root user
USER todoapp

# Expose SSH port
EXPOSE 2222

# Run the application
CMD ["./todoissh"] 