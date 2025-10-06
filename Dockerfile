FROM golang:1.24-bullseye AS builder

# Build arguments for multi-architecture support
ARG TARGETOS=linux
ARG TARGETARCH

WORKDIR /app

# Install required build dependencies (architecture-agnostic packages)
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Copy the whole source first since we need to handle the dependency properly
COPY . .

# Download dependencies
RUN go mod tidy && go mod download

# Build binaries for BOTH amd64 and arm64 architectures
# This allows the image to run on either architecture
RUN mkdir -p bin/amd64 bin/arm64 && \
    echo "Building for amd64..." && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=amd64 go build -mod=mod -o bin/amd64/loadbalancer && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=amd64 go build -mod=mod -o bin/amd64/ ./cmd/... && \
    echo "Building for arm64..." && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=arm64 go build -mod=mod -o bin/arm64/loadbalancer && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=arm64 go build -mod=mod -o bin/arm64/ ./cmd/... && \
    echo "Build complete for both architectures"

# Use Debian slim for the final container
FROM debian:bullseye-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    iproute2 \
    net-tools \
    bash \
    dnsutils \
    traceroute \
    iputils-ping \
    curl \
    iw \
    procps \
    libcap2-bin \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binaries for BOTH architectures
# The entrypoint script will select the correct one based on runtime architecture
COPY --from=builder /app/bin/amd64/ /usr/local/bin/amd64/
COPY --from=builder /app/bin/arm64/ /usr/local/bin/arm64/

# Set Linux capabilities on both architecture binaries (if supported)
# This may fail on some platforms (e.g., macOS Docker) but that's okay
# The container will need to run with --privileged or --cap-add in those cases
RUN setcap 'cap_net_admin,cap_net_raw,cap_net_bind_service=+ep' /usr/local/bin/amd64/loadbalancer || \
    echo "Warning: Could not set capabilities on amd64 binary" && \
    setcap 'cap_net_admin,cap_net_raw,cap_net_bind_service=+ep' /usr/local/bin/arm64/loadbalancer || \
    echo "Warning: Could not set capabilities on arm64 binary"

# Create architecture detection script and symlinks
# This will be run at build time to set up symlinks for the current architecture
RUN echo '#!/bin/bash' > /usr/local/bin/setup-arch-links.sh && \
    echo 'ARCH=$(uname -m)' >> /usr/local/bin/setup-arch-links.sh && \
    echo 'case "$ARCH" in' >> /usr/local/bin/setup-arch-links.sh && \
    echo '  x86_64|amd64) BINARY_ARCH="amd64" ;;' >> /usr/local/bin/setup-arch-links.sh && \
    echo '  aarch64|arm64) BINARY_ARCH="arm64" ;;' >> /usr/local/bin/setup-arch-links.sh && \
    echo '  *) BINARY_ARCH="amd64" ;;' >> /usr/local/bin/setup-arch-links.sh && \
    echo 'esac' >> /usr/local/bin/setup-arch-links.sh && \
    echo 'for bin in /usr/local/bin/${BINARY_ARCH}/*; do' >> /usr/local/bin/setup-arch-links.sh && \
    echo '  ln -sf "$bin" "/usr/local/bin/$(basename $bin)"' >> /usr/local/bin/setup-arch-links.sh && \
    echo 'done' >> /usr/local/bin/setup-arch-links.sh && \
    chmod +x /usr/local/bin/setup-arch-links.sh

# Copy entrypoint script
COPY scripts/entrypoint.sh /app/scripts/
RUN chmod +x /app/scripts/entrypoint.sh

ENTRYPOINT ["/app/scripts/entrypoint.sh"]
