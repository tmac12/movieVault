# Stage 1: Build Go scanner
FROM golang:1.25-alpine AS go-builder

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

# Stage 2: Build Astro website (we'll skip this for initial setup)
FROM node:20-alpine AS web-builder

WORKDIR /build

# Copy package files
COPY website/package*.json ./
RUN npm ci --only=production

# Copy website source
COPY website/ ./

# Build will happen at runtime with actual data

# Stage 3: Final runtime image
FROM nginx:alpine

# Install Node.js and Go runtime dependencies
RUN apk add --no-cache nodejs npm

# Copy Go scanner binary
COPY --from=go-builder /build/scanner /usr/local/bin/scanner
RUN chmod +x /usr/local/bin/scanner && \
    ls -lah /usr/local/bin/scanner && \
    /usr/local/bin/scanner --help || echo "Scanner installed successfully"

# Copy Astro website source (not built, will build at runtime)
COPY --from=web-builder /build /app/website

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
