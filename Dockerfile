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

# Final stage
FROM alpine:3.19

# Install dependencies
RUN apk --no-cache add ca-certificates

# Create a non-root user
RUN adduser -D -h /app todoapp

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/todoissh .

# Set ownership
RUN chown -R todoapp:todoapp /app

# Switch to non-root user
USER todoapp

# Expose SSH port
EXPOSE 2222

# Run the application
ENTRYPOINT ["./todoissh"] 