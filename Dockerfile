# Stage 1: Build Go scanner
FROM golang:1.21-alpine AS go-builder

WORKDIR /build

# Copy go module files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Build scanner binary
RUN go build -o scanner cmd/scanner/main.go

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
RUN chmod +x /usr/local/bin/scanner

# Copy Astro website source (not built, will build at runtime)
COPY --from=web-builder /build /app/website

# Create data directories
RUN mkdir -p /data/movies /data/covers /config

# Copy nginx config
COPY docker/nginx.conf /etc/nginx/conf.d/default.conf

# Copy entrypoint script
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 80

ENTRYPOINT ["/entrypoint.sh"]
