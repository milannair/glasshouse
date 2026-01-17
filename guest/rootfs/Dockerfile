# Build stage - compile guest init
FROM golang:1.23-alpine AS builder
WORKDIR /build
COPY guest/init/main.go .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o guest-init main.go

# Runtime stage - minimal Python environment
FROM python:3.12-alpine

# Install only essential packages
RUN apk add --no-cache busybox

# Copy the guest init binary
COPY --from=builder /build/guest-init /sbin/init

# Ensure init is executable
RUN chmod +x /sbin/init

# Create workspace mount point
RUN mkdir -p /workspace
