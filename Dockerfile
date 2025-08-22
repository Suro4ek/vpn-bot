# Multi-stage build for Go application
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o vpn-bot ./cmd/main.go

# Final stage with WireGuard
FROM alpine:3.22

# Add community repository and install WireGuard and required packages
RUN echo "http://dl-cdn.alpinelinux.org/alpine/v3.22/community" >> /etc/apk/repositories && \
    apk update && \
    apk add --no-cache \
    wireguard-tools \
    iptables \
    ip6tables \
    bash \
    curl \
    libqrencode \
    openrc \
    supervisor

# Copy the built binary
COPY --from=builder /app/vpn-bot /usr/local/bin/vpn-bot

# Copy configuration files
COPY --from=builder /app/pkg/wireguard/templates/ /app/pkg/wireguard/templates/
COPY wireguard-install.sh /usr/local/bin/wireguard-install.sh
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

# Make scripts executable
RUN chmod +x /usr/local/bin/vpn-bot
RUN chmod +x /usr/local/bin/wireguard-install.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# Create necessary directories
RUN mkdir -p /etc/wireguard
RUN mkdir -p /var/log/supervisor
RUN mkdir -p /etc/supervisor/conf.d

# Create supervisor configuration
RUN echo "[supervisord]" > /etc/supervisor/conf.d/supervisord.conf && \
    echo "nodaemon=true" >> /etc/supervisor/conf.d/supervisord.conf && \
    echo "logfile=/var/log/supervisor/supervisord.log" >> /etc/supervisor/conf.d/supervisord.conf && \
    echo "pidfile=/var/run/supervisord.pid" >> /etc/supervisor/conf.d/supervisord.conf && \
    echo "" >> /etc/supervisor/conf.d/supervisord.conf && \
    echo "[program:vpn-bot]" >> /etc/supervisor/conf.d/supervisord.conf && \
    echo "command=/usr/local/bin/vpn-bot" >> /etc/supervisor/conf.d/supervisord.conf && \
    echo "autostart=true" >> /etc/supervisor/conf.d/supervisord.conf && \
    echo "autorestart=true" >> /etc/supervisor/conf.d/supervisord.conf && \
    echo "stderr_logfile=/var/log/supervisor/vpn-bot.err.log" >> /etc/supervisor/conf.d/supervisord.conf && \
    echo "stdout_logfile=/var/log/supervisor/vpn-bot.out.log" >> /etc/supervisor/conf.d/supervisord.conf

# Enable IP forwarding
RUN echo "net.ipv4.ip_forward = 1" >> /etc/sysctl.conf
RUN echo "net.ipv6.conf.all.forwarding = 1" >> /etc/sysctl.conf

# Expose WireGuard port (default)
EXPOSE 51820/udp

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]

# Use supervisor to run multiple services
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]