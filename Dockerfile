# Stage 1: Build Go scanner
FROM golang:1.25-alpine AS go-builder

# Install file command for binary verification
RUN apk add --no-cache file

# Build arguments for multi-platform support
ARG TARGETOS
ARG TARGETARCH

WORKDIR /build

# Copy go module files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Build scanner binary with cross-compilation support
# Note: Build entire package (./cmd/scanner) not just main.go to include scan.go and scheduler.go
RUN GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -v -o scanner ./cmd/scanner && \
    ls -lah scanner && \
    file scanner

# Stage 2: Prepare Astro website source (no build yet)
FROM node:20-alpine AS web-source

WORKDIR /build

# Just copy source files, don't install dependencies yet
COPY website/ ./

# Stage 3: Final runtime image
FROM nginx:alpine

# Install Node.js and npm
RUN apk add --no-cache nodejs npm

# Copy Go scanner binary
COPY --from=go-builder /build/scanner /usr/local/bin/scanner
RUN chmod +x /usr/local/bin/scanner && \
    ls -lah /usr/local/bin/scanner && \
    /usr/local/bin/scanner --help || echo "Scanner installed successfully"

# Copy Astro website source (without node_modules)
COPY --from=web-source /build /app/website

# Install npm dependencies in the final image to ensure compatibility
# This fixes esbuild native binary issues
RUN cd /app/website && npm ci

# Create data directories
RUN mkdir -p /data/movies /data/covers /config

# Copy Docker-specific config
COPY config/config.docker.yaml /config/config.yaml

# Copy nginx config
COPY docker/nginx.conf /etc/nginx/conf.d/default.conf

# Copy entrypoint script
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Copy health check script
COPY docker/healthcheck.sh /healthcheck.sh
RUN chmod +x /healthcheck.sh

# Install curl for health check
RUN apk add --no-cache curl

# Add health check
HEALTHCHECK --interval=30s --timeout=3s --retries=3 CMD /healthcheck.sh

EXPOSE 80

ENTRYPOINT ["/entrypoint.sh"]
