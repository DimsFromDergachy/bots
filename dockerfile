# Stage 1: Build the Go binary
FROM golang:1.26-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata gcc musl-dev

# Set working directory
WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application
ARG TARGET_ARCH
RUN CGO_ENABLED=1 GOOS=linux GOARCH=${TARGET_ARCH} \
    go build -ldflags="-w -s" -o bible-bot .

# Stage 2: Create minimal runtime image
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Create data directory and set permissions
RUN mkdir -p /app/data && \
    chown -R appuser:appgroup /app

# Copy binary and set ownership
COPY --from=builder --chown=appuser:appgroup /build/templates /app/templates
COPY --from=builder --chown=appuser:appgroup /build/bible-bot /app/bible-bot

# Switch to non-root user
USER appuser

# Set working directory
WORKDIR /app

# Expose port (adjust to your app's port)
EXPOSE 8080

# Run the application
CMD ["/app/bible-bot"]