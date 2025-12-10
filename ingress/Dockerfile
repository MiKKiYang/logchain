# Nginx API Gateway Dockerfile
# Based on official Nginx image with OpenResty for Lua support

FROM openresty/openresty:alpine

# Install additional dependencies
RUN apk add --no-cache \
    curl \
    ca-certificates \
    lua-resty-http \
    lua-resty-openssl \
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

# Expose ports
EXPOSE 80 443 50052

# Health check
HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
    CMD curl -f http://localhost/health || exit 1

# Start Nginx with our config path
CMD ["/usr/local/openresty/bin/openresty", "-c", "/etc/nginx/nginx.conf", "-g", "daemon off;"]

