#!/bin/bash
# Generate SSL certificates for Nginx API Gateway
# This script generates self-signed certificates for development/testing
# For production, use certificates from a trusted CA

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SSL_DIR="${SCRIPT_DIR}/../nginx/ssl"
CA_DIR="${SSL_DIR}/ca"

# Create directories
mkdir -p "${SSL_DIR}" "${CA_DIR}"

echo "Generating SSL certificates for Nginx API Gateway..."

# Generate CA private key
echo "1. Generating CA private key..."
openssl genrsa -out "${CA_DIR}/ca-key.pem" 4096

# Generate CA certificate
echo "2. Generating CA certificate..."
openssl req -new -x509 -days 3650 -key "${CA_DIR}/ca-key.pem" \
    -out "${CA_DIR}/ca-cert.pem" \
    -subj "/C=US/ST=State/L=City/O=LogChain Consortium/CN=LogChain CA"

# Generate server private key
echo "3. Generating server private key..."
openssl genrsa -out "${SSL_DIR}/key.pem" 2048

# Generate server certificate signing request
echo "4. Generating server certificate signing request..."
openssl req -new -key "${SSL_DIR}/key.pem" \
    -out "${SSL_DIR}/server.csr" \
    -subj "/C=US/ST=State/L=City/O=LogChain/CN=api-gateway"

# Generate server certificate signed by CA
echo "5. Generating server certificate..."
openssl x509 -req -days 365 -in "${SSL_DIR}/server.csr" \
    -CA "${CA_DIR}/ca-cert.pem" \
    -CAkey "${CA_DIR}/ca-key.pem" \
    -CAcreateserial \
    -out "${SSL_DIR}/cert.pem" \
    -extensions v3_req \
    -extfile <(cat <<EOF
[req]
distinguished_name = req_distinguished_name
[v3_req]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names
[alt_names]
DNS.1 = localhost
DNS.2 = api-gateway
DNS.3 = *.localhost
IP.1 = 127.0.0.1
IP.2 = ::1
EOF
)

# Copy CA certificate for mTLS client verification
cp "${CA_DIR}/ca-cert.pem" "${SSL_DIR}/ca-cert.pem"

# Set proper permissions
chmod 600 "${SSL_DIR}/key.pem"
chmod 644 "${SSL_DIR}/cert.pem"
chmod 644 "${SSL_DIR}/ca-cert.pem"
chmod 600 "${CA_DIR}/ca-key.pem"
chmod 644 "${CA_DIR}/ca-cert.pem"

# Clean up
rm -f "${SSL_DIR}/server.csr"

echo ""
echo "SSL certificates generated successfully!"
echo "Certificate files:"
echo "  - Server cert: ${SSL_DIR}/cert.pem"
echo "  - Server key: ${SSL_DIR}/key.pem"
echo "  - CA cert (for mTLS): ${SSL_DIR}/ca-cert.pem"
echo ""
echo "For production, replace these with certificates from a trusted CA."

