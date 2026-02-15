# Build stage
# We use golang:1.22-alpine for a small build footprint.
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Download dependencies first to leverage Docker cache
COPY go.mod ./
RUN go mod download

# Copy source code and build the binary
COPY . .
# CGO_ENABLED=0 ensures a static binary, crucial for running in distroless/scratch
RUN CGO_ENABLED=0 GOOS=linux go build -o golangzakhireh ./cmd/server

# Run stage
# We use distroless static-debian12 for security (no shell, non-root by default support)
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /

# Copy the binary from builder
COPY --from=builder /app/golangzakhireh /golangzakhireh
# Copy templates for the dashboard
COPY --from=builder /app/internal/dashboard/templates /internal/dashboard/templates

# Volume configuration
# The /data/modules directory is used to persist cached modules.
# Ensure this directory is writable by the non-root user (UID 65532).
# When mounting a volume, you may need to set permissions on the host directory:
# chown -R 65532:65532 ./data/modules

USER 65532:65532

# Expose the default port
EXPOSE 8811

# Environment Variables
ENV GOLANGZAKHIREH_PORT=:8811
ENV GOLANGZAKHIREH_DATA_DIR=/data/modules

ENTRYPOINT ["/golangzakhireh"]
