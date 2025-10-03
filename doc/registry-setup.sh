#!/usr/bin/env bash
# Setup Script

# The following script sets up a Docker registry on the =kdc1= host:


# Configuration
REGISTRY_HOST="kdc1"
REGISTRY_DOMAIN="registry.ac0.net"
REGISTRY_PORT=5000
REGISTRY_DIR="/opt/docker-registry"
REGISTRY_DATA_DIR="${REGISTRY_DIR}/data"
REGISTRY_CERTS_DIR="${REGISTRY_DIR}/certs"
REGISTRY_AUTH_DIR="${REGISTRY_DIR}/auth"
NFS_SERVER="tnas.ac0.net"
NFS_EXPORT="/volume1/docker-registry"
LOCAL_CIDR="192.168.0.0/24"

# Set up directories
echo "Creating registry directories..."
sudo mkdir -p ${REGISTRY_CERTS_DIR} ${REGISTRY_AUTH_DIR} ${REGISTRY_DATA_DIR}

# TLS Certificate Generation

# Generate TLS certificates for secure communication:


echo "Generating TLS certificates..."
cd ${REGISTRY_CERTS_DIR}

# Generate CA key and certificate
sudo openssl genrsa -out ca-key.pem 4096
sudo openssl req -new -x509 -days 3650 -key ca-key.pem -sha256 -subj "/CN=Docker Registry CA" -out ca.pem

# Generate server key
sudo openssl genrsa -out server-key.pem 4096

# Create a CSR and include all your hostnames/IPs
sudo openssl req -subj "/CN=${REGISTRY_DOMAIN}" -sha256 -new -key server-key.pem -out server.csr

# Configure certificate extensions
cat > extfile.cnf <<EOF
subjectAltName = DNS:${REGISTRY_HOST},DNS:${REGISTRY_HOST}.ac0.net,DNS:*.ac0.net,DNS:${REGISTRY_DOMAIN},DNS:tnas.ac0.net,DNS:tnas.afs0.net,DNS:tnas.local,IP:192.168.0.107,IP:${LOCAL_CIDR%/*}
extendedKeyUsage = serverAuth
EOF

# Generate the certificate
sudo openssl x509 -req -days 3650 -sha256 -in server.csr -CA ca.pem -CAkey ca-key.pem \
  -CAcreateserial -out server-cert.pem -extfile extfile.cnf

# Set appropriate permissions
sudo chmod 0600 ${REGISTRY_CERTS_DIR}/ca-key.pem ${REGISTRY_CERTS_DIR}/server-key.pem

# NFS Storage Configuration

# Configure NFS storage for registry data persistence:


echo "Setting up NFS mount..."

# Add NFS mount to fstab if not already present
if ! grep -q "${NFS_SERVER}:${NFS_EXPORT}" /etc/fstab; then
  echo "${NFS_SERVER}:${NFS_EXPORT} ${REGISTRY_DATA_DIR} nfs defaults,soft,timeo=30,retry=2 0 0" | sudo tee -a /etc/fstab
fi

# Mount the NFS share
sudo mount ${REGISTRY_DATA_DIR} || echo "NFS mount already exists or failed (might need to verify NFS server)"

# systemd Service Configuration

# Create a systemd service for managing the registry container:


echo "Creating systemd service..."

# Create systemd service file
sudo bash -c "cat > /etc/systemd/system/docker-registry.service" << EOF
[Unit]
Description=Docker Registry Container
After=docker.service network-online.target
Requires=docker.service network-online.target

[Service]
Type=simple
TimeoutStartSec=0
Restart=always
ExecStartPre=-/usr/bin/docker stop %n
ExecStartPre=-/usr/bin/docker rm %n
ExecStart=/usr/bin/docker run --rm --name docker-registry \\
  -p ${REGISTRY_PORT}:5000 \\
  -v ${REGISTRY_DATA_DIR}:/var/lib/registry \\
  -v ${REGISTRY_CERTS_DIR}:/certs \\
  -e REGISTRY_HTTP_TLS_CERTIFICATE=/certs/server-cert.pem \\
  -e REGISTRY_HTTP_TLS_KEY=/certs/server-key.pem \\
  registry:2
ExecStop=/usr/bin/docker stop %n

[Install]
WantedBy=multi-user.target
EOF

# Enable and start the service
echo "Enabling and starting registry service..."
sudo systemctl daemon-reload
sudo systemctl enable docker-registry
sudo systemctl start docker-registry

# Client Setup Script

# Create a script to configure client hosts to use the registry:


echo "Creating client setup script..."
sudo bash -c "cat > ${REGISTRY_DIR}/setup-client.sh" << 'EOF'
#!/bin/bash
# Docker Registry Client Setup Script

REGISTRY_HOST="kdc1"
REGISTRY_DOMAIN="registry.ac0.net"
REGISTRY_PORT=5000
REGISTRY_ADDRESS="${REGISTRY_DOMAIN}:${REGISTRY_PORT}"
CERT_DIR="/etc/docker/certs.d/${REGISTRY_ADDRESS}"

# Create certificate directory
sudo mkdir -p ${CERT_DIR}

# Copy CA certificate
sudo cp /opt/docker-registry/certs/ca.pem ${CERT_DIR}/ca.crt

# Add registry entries to /etc/hosts if needed
if ! grep -q "${REGISTRY_DOMAIN}" /etc/hosts; then
  # Get registry server IP
  REGISTRY_IP=$(getent hosts ${REGISTRY_HOST} | awk '{ print $1 }')
  if [ -z "$REGISTRY_IP" ]; then
    echo "Could not resolve ${REGISTRY_HOST}, please enter the IP address:"
    read REGISTRY_IP
  fi
  echo "${REGISTRY_IP} ${REGISTRY_DOMAIN}" | sudo tee -a /etc/hosts
fi

# Add NFS server to /etc/hosts if needed
if ! grep -q "tnas.ac0.net" /etc/hosts; then
  echo "192.168.0.107 tnas.ac0.net tnas.afs0.net tnas.local" | sudo tee -a /etc/hosts
fi

# Restart Docker daemon
sudo systemctl restart docker

echo "Client setup complete. You can now use:"
echo "  docker pull ${REGISTRY_ADDRESS}/your-image"
echo "  docker push ${REGISTRY_ADDRESS}/your-image"
EOF

sudo chmod +x ${REGISTRY_DIR}/setup-client.sh

echo "====================================================="
echo "Docker Registry setup complete!"
echo "Registry is running at: ${REGISTRY_DOMAIN}:${REGISTRY_PORT}"
echo "To configure clients, run: ${REGISTRY_DIR}/setup-client.sh"
echo "====================================================="
