# Build stage
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with security flags
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server

# Final stage
FROM alpine:3.21

# Add non-root user for security
RUN adduser -D -g '' appuser

# Install CA certificates for HTTPS connections
RUN apk add --no-cache ca-certificates

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Change ownership of the binary
RUN chown appuser:appuser /app/server

# Switch to non-root user
USER appuser

# Expose the application port
EXPOSE 8080

# Set environment variables
ENV PORT=8080

# Run the application
CMD ["/app/server"]
