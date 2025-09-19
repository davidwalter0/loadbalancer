FROM golang:1.24-bullseye AS builder

WORKDIR /app

# Install required build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    build-essential \
    linux-headers-amd64 \
    && rm -rf /var/lib/apt/lists/*

# Copy the whole source first since we need to handle the dependency properly
COPY . .

# Download dependencies
RUN go mod tidy && go mod download

# Build the loadbalancer binary and support utilities
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o loadbalancer \
    && mkdir -p bin \
    && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/ ./cmd/...

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
    apt-transport-https \
    gnupg \
    iw \
    procps \
    && mkdir -p /etc/apt/keyrings \
    && curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.30/deb/Release.key | gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg \
    && echo "deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.30/deb/ /" > /etc/apt/sources.list.d/kubernetes.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends kubectl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy all binaries to /usr/local/bin for easy access in PATH
COPY --from=builder /app/loadbalancer /usr/local/bin/
COPY --from=builder /app/bin/* /usr/local/bin/

# Set Linux capabilities on the loadbalancer binary
RUN apt-get update && apt-get install -y --no-install-recommends libcap2-bin \
    && setcap 'cap_net_admin,cap_net_raw,cap_net_bind_service=+ep' /usr/local/bin/loadbalancer \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Copy entrypoint script
COPY scripts/entrypoint.sh /app/scripts/
RUN chmod +x /app/scripts/entrypoint.sh

ENTRYPOINT ["/app/scripts/entrypoint.sh"]
