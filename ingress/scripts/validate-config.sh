#!/bin/bash
# Validate Nginx API Gateway configuration
# This script validates configuration files and SSL certificates

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INGRESS_DIR="${SCRIPT_DIR}/.."
CONF_DIR="${INGRESS_DIR}/nginx/conf.d"
SSL_DIR="${INGRESS_DIR}/ssl"
NGINX_CONF="${INGRESS_DIR}/nginx/nginx.conf"

ERRORS=0
WARNINGS=0

echo "Validating Nginx API Gateway configuration..."
echo ""

# Check if nginx.conf exists
if [ ! -f "${NGINX_CONF}" ]; then
    echo "ERROR: nginx.conf not found at ${NGINX_CONF}"
    ERRORS=$((ERRORS + 1))
else
    echo "nginx.conf found"
    
    # Validate nginx configuration syntax (if nginx is available)
    if command -v nginx &> /dev/null; then
        if nginx -t -c "${NGINX_CONF}" 2>&1 | grep -q "syntax is ok"; then
            echo "nginx.conf syntax is valid"
        else
            echo "ERROR: nginx.conf syntax is invalid"
            nginx -t -c "${NGINX_CONF}" 2>&1 | grep -v "syntax is ok"
            ERRORS=$((ERRORS + 1))
        fi
    else
        echo "WARNING: nginx command not found, skipping syntax validation"
        WARNINGS=$((WARNINGS + 1))
    fi
fi

echo ""

# Check API keys file
if [ ! -f "${CONF_DIR}/api-keys.json" ]; then
    echo "WARNING: api-keys.json not found (using example file)"
    WARNINGS=$((WARNINGS + 1))
else
    echo "api-keys.json found"
    
    # Validate JSON syntax
    if command -v jq &> /dev/null; then
        if jq empty "${CONF_DIR}/api-keys.json" 2>/dev/null; then
            echo "api-keys.json is valid JSON"
        else
            echo "ERROR: api-keys.json is not valid JSON"
            ERRORS=$((ERRORS + 1))
        fi
    else
        echo "WARNING: jq not found, skipping JSON validation"
        WARNINGS=$((WARNINGS + 1))
    fi
fi

echo ""

# Check consortium IP whitelist file
if [ ! -f "${CONF_DIR}/consortium-ip-whitelist.json" ]; then
    echo "WARNING: consortium-ip-whitelist.json not found (using example file)"
    WARNINGS=$((WARNINGS + 1))
else
    echo "consortium-ip-whitelist.json found"
    
    # Validate JSON syntax
    if command -v jq &> /dev/null; then
        if jq empty "${CONF_DIR}/consortium-ip-whitelist.json" 2>/dev/null; then
            echo "consortium-ip-whitelist.json is valid JSON"
        else
            echo "ERROR: consortium-ip-whitelist.json is not valid JSON"
            ERRORS=$((ERRORS + 1))
        fi
    fi
fi

echo ""

# Check SSL certificates
if [ ! -f "${SSL_DIR}/cert.pem" ] || [ ! -f "${SSL_DIR}/key.pem" ]; then
    echo "WARNING: SSL certificates not found in ${SSL_DIR}"
    echo "Run ./scripts/generate-ssl-certs.sh to generate certificates"
    WARNINGS=$((WARNINGS + 1))
else
    echo "SSL certificates found"
    
    # Validate certificate
    if command -v openssl &> /dev/null; then
        if openssl x509 -in "${SSL_DIR}/cert.pem" -noout -text &> /dev/null; then
            echo "SSL certificate is valid"
            
            # Check expiration
            EXPIRY=$(openssl x509 -in "${SSL_DIR}/cert.pem" -noout -enddate | cut -d= -f2)
            EXPIRY_EPOCH=$(date -d "$EXPIRY" +%s 2>/dev/null || date -j -f "%b %d %H:%M:%S %Y" "$EXPIRY" +%s 2>/dev/null || echo "0")
            NOW_EPOCH=$(date +%s)
            DAYS_LEFT=$(( (EXPIRY_EPOCH - NOW_EPOCH) / 86400 ))
            
            if [ $DAYS_LEFT -lt 30 ]; then
                echo "WARNING: SSL certificate expires in $DAYS_LEFT days"
                WARNINGS=$((WARNINGS + 1))
            else
                echo "SSL certificate expires in $DAYS_LEFT days"
            fi
        else
            echo "ERROR: SSL certificate is invalid"
            ERRORS=$((ERRORS + 1))
        fi
    fi
fi

echo ""

# Check CA certificate for mTLS
if [ ! -f "${SSL_DIR}/ca-cert.pem" ]; then
    echo "WARNING: CA certificate not found in ${SSL_DIR}"
    echo "Required for mTLS authentication"
    WARNINGS=$((WARNINGS + 1))
else
    echo "CA certificate found"
    
    # Validate CA certificate
    if command -v openssl &> /dev/null; then
        if openssl x509 -in "${SSL_DIR}/ca-cert.pem" -noout -text &> /dev/null; then
            echo "CA certificate is valid"
        else
            echo "ERROR: CA certificate is invalid"
            ERRORS=$((ERRORS + 1))
        fi
    fi
fi

echo ""

# Check Lua scripts
LUA_SCRIPTS=(
    "api-key-auth.lua"
    "grpc-api-key-auth.lua"
    "mtls-ip-auth.lua"
)

for script in "${LUA_SCRIPTS[@]}"; do
    if [ -f "${INGRESS_DIR}/nginx/lua/${script}" ]; then
        echo "${script} found"
    else
        echo "ERROR: ${script} not found"
        ERRORS=$((ERRORS + 1))
    fi
done

echo ""
echo "=========================================="
echo "Validation Summary:"
echo "  Errors: ${ERRORS}"
echo "  Warnings: ${WARNINGS}"
echo "=========================================="

if [ $ERRORS -gt 0 ]; then
    echo ""
    echo "Configuration validation failed. Please fix the errors above."
    exit 1
elif [ $WARNINGS -gt 0 ]; then
    echo ""
    echo "Configuration has warnings. Review and update as needed."
    exit 0
else
    echo ""
    echo "Configuration validation passed!"
    exit 0
fi

