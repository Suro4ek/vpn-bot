#!/bin/bash
set -e

# Create directories
mkdir -p /etc/wireguard
mkdir -p /app/data

# Check if WireGuard is already configured
if [ ! -f "/etc/wireguard/params" ]; then
    echo "Initializing WireGuard configuration..."
    
    # Set default values if not provided
    SERVER_PUB_IP=${SERVER_PUB_IP:-"auto"}
    SERVER_PUB_NIC=${SERVER_PUB_NIC:-"eth0"}
    SERVER_WG_NIC=${SERVER_WG_NIC:-"wg0"}
    SERVER_WG_IPV4=${SERVER_WG_IPV4:-"10.66.66.1"}
    SERVER_WG_IPV6=${SERVER_WG_IPV6:-"fd42:42:42::1"}
    SERVER_PORT=${SERVER_PORT:-"51820"}
    CLIENT_DNS_1=${CLIENT_DNS_1:-"1.1.1.1"}
    CLIENT_DNS_2=${CLIENT_DNS_2:-"1.0.0.1"}
    ALLOWED_IPS=${ALLOWED_IPS:-"0.0.0.0/0,::/0"}
    
    # Auto-detect public IP if set to auto
    if [ "$SERVER_PUB_IP" = "auto" ]; then
        SERVER_PUB_IP=$(curl -s https://ipinfo.io/ip || echo "127.0.0.1")
        echo "Auto-detected public IP: $SERVER_PUB_IP"
    fi
    
    # Generate WireGuard keys
    SERVER_PRIV_KEY=$(wg genkey)
    SERVER_PUB_KEY=$(echo "${SERVER_PRIV_KEY}" | wg pubkey)
    
    # Create WireGuard params file
    cat > /etc/wireguard/params << EOF
SERVER_PUB_IP=${SERVER_PUB_IP}
SERVER_PUB_NIC=${SERVER_PUB_NIC}
SERVER_WG_NIC=${SERVER_WG_NIC}
SERVER_WG_IPV4=${SERVER_WG_IPV4}
SERVER_WG_IPV6=${SERVER_WG_IPV6}
SERVER_PORT=${SERVER_PORT}
SERVER_PRIV_KEY=${SERVER_PRIV_KEY}
SERVER_PUB_KEY=${SERVER_PUB_KEY}
CLIENT_DNS_1=${CLIENT_DNS_1}
CLIENT_DNS_2=${CLIENT_DNS_2}
ALLOWED_IPS=${ALLOWED_IPS}
EOF

    # Create WireGuard server configuration
    cat > "/etc/wireguard/${SERVER_WG_NIC}.conf" << EOF
[Interface]
Address = ${SERVER_WG_IPV4}/24,${SERVER_WG_IPV6}/64
ListenPort = ${SERVER_PORT}
PrivateKey = ${SERVER_PRIV_KEY}
PostUp = iptables -A FORWARD -i ${SERVER_WG_NIC} -j ACCEPT; iptables -t nat -A POSTROUTING -o ${SERVER_PUB_NIC} -j MASQUERADE
PostDown = iptables -D FORWARD -i ${SERVER_WG_NIC} -j ACCEPT; iptables -t nat -D POSTROUTING -o ${SERVER_PUB_NIC} -j MASQUERADE
EOF

    echo "WireGuard configuration created."
else
    echo "WireGuard configuration already exists."
    source /etc/wireguard/params
fi

# Enable IP forwarding
echo 1 > /proc/sys/net/ipv4/ip_forward
echo 1 > /proc/sys/net/ipv6/conf/all/forwarding

# Load WireGuard module if available
modprobe wireguard 2>/dev/null || echo "WireGuard module not available, using userspace implementation"

# Check if qrencode is available
if ! command -v qrencode &> /dev/null; then
    echo "Warning: qrencode not found, QR codes will not be generated"
fi

# Start WireGuard
echo "Starting WireGuard interface: ${SERVER_WG_NIC}"
wg-quick up "${SERVER_WG_NIC}" 2>/dev/null || echo "WireGuard interface already up or failed to start"

# Execute the main command
exec "$@"