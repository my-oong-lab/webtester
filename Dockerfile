# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o webtester ./cmd/app

# Final stage
FROM alpine:3.20

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata sqlite-libs

# Copy binary from builder
COPY --from=builder /app/webtester .

# Copy example config (used as default if config.yaml not mounted)
COPY config.example.yaml ./config.example.yaml

# Create non-root user
RUN adduser -D -u 1000 appuser && chown -R appuser:appuser /app
USER appuser

# Expose HTTP port
EXPOSE 9090

# Run with config fallback: use mounted config.yaml or copy from example
ENTRYPOINT ["/bin/sh", "-c", "if [ ! -f config.yaml ]; then cp config.example.yaml config.yaml; fi; ./webtester"]