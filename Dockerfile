# Build stage
FROM golang:1.23-alpine AS builder

# Install git for go mod download
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='-w -s' -o greens-marketplace ./cmd/server

# Runtime stage
FROM alpine:3.19

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Set timezone
ENV TZ=UTC

# Create app user
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/greens-marketplace .

# Copy migrations directory
COPY --from=builder /app/migrations ./migrations

# Change ownership to appuser
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the binary
CMD ["./greens-marketplace", "-config", "config.yaml"]