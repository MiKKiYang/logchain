# Nginx API Gateway Dockerfile
# Based on official Nginx image with OpenResty for Lua support

FROM openresty/openresty:alpine

# Install additional dependencies
RUN apk add --no-cache \
    curl \
    ca-certificates \
    && rm -rf /var/cache/apk/*

# Create necessary directories
RUN mkdir -p /etc/nginx/conf.d \
    /etc/nginx/lua \
    /etc/nginx/ssl \
    /var/log/nginx \
    /var/cache/nginx

# Copy Nginx configuration files
COPY nginx/nginx.conf /etc/nginx/nginx.conf
COPY nginx/conf.d/ /etc/nginx/conf.d/
COPY nginx/lua/ /etc/nginx/lua/

# Set permissions
RUN chmod -R 755 /etc/nginx/lua \
    && chmod 644 /etc/nginx/nginx.conf \
    && chmod 644 /etc/nginx/conf.d/*.conf

# Create placeholder SSL certificates (should be replaced with real certificates in production)
# These are only used if SSL certificates are not mounted via volume
RUN if [ ! -f /etc/nginx/ssl/cert.pem ]; then \
        openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
            -keyout /etc/nginx/ssl/key.pem \
            -out /etc/nginx/ssl/cert.pem \
            -subj "/C=US/ST=State/L=City/O=LogChain/CN=api-gateway" \
        && chmod 600 /etc/nginx/ssl/key.pem \
        && chmod 644 /etc/nginx/ssl/cert.pem; \
    fi

# Create placeholder CA certificate for mTLS (should be replaced with real CA cert in production)
RUN if [ ! -f /etc/nginx/ssl/ca-cert.pem ]; then \
        openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
            -keyout /etc/nginx/ssl/ca-key.pem \
            -out /etc/nginx/ssl/ca-cert.pem \
            -subj "/C=US/ST=State/L=City/O=LogChain Consortium/CN=CA" \
        && chmod 600 /etc/nginx/ssl/ca-key.pem \
        && chmod 644 /etc/nginx/ssl/ca-cert.pem; \
    fi

# Create placeholder API keys file (should be replaced with real API keys)
COPY nginx/conf.d/api-keys.json.example /etc/nginx/conf.d/api-keys.json

# Create placeholder consortium config files
COPY nginx/conf.d/consortium-ip-whitelist.json.example /etc/nginx/conf.d/consortium-ip-whitelist.json

# Expose ports
EXPOSE 80 443 50052

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost/health || exit 1

# Start Nginx
CMD ["/usr/local/openresty/bin/openresty", "-g", "daemon off;"]

