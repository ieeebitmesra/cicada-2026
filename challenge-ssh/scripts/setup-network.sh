#!/bin/bash
# challenge-ssh/scripts/setup-network.sh
# Bridge + TAP + firewall rules for the challenge microVM.
# (The Go orchestrator performs the same steps natively via `orchestrator setup-net`.)

set -euo pipefail

BRIDGE="br0"
BRIDGE_IP="172.16.0.1/24"
TAP_NAME="tap-challenge"
VM_IP="172.16.0.2"
HOST_PORT="2223"
EXT_IFACE="${EXT_IFACE:-eth0}"   # change to your external interface

[ "$(id -u)" -eq 0 ] || { echo "must run as root"; exit 1; }

echo "=== Setting up Firecracker network ==="

# Enable IP forwarding
sysctl -w net.ipv4.ip_forward=1

# Create bridge
ip link add name "$BRIDGE" type bridge 2>/dev/null || true
ip addr add "$BRIDGE_IP" dev "$BRIDGE" 2>/dev/null || true
ip link set "$BRIDGE" up

# NAT for outbound is intentionally NOT configured: VMs have no egress.

echo "=== Installing firewall rules ==="
# BLOCK inter-VM traffic
iptables -C FORWARD -i "$BRIDGE" -o "$BRIDGE" -j DROP 2>/dev/null || \
    iptables -A FORWARD -i "$BRIDGE" -o "$BRIDGE" -j DROP

# BLOCK all egress FROM VMs (prevent reverse shells)
iptables -C FORWARD -i "$BRIDGE" -j DROP 2>/dev/null || \
    iptables -A FORWARD -i "$BRIDGE" -j DROP

echo "=== Creating TAP device for challenge VM ==="
ip tuntap add dev "$TAP_NAME" mode tap 2>/dev/null || true
ip link set dev "$TAP_NAME" master "$BRIDGE"
ip link set dev "$TAP_NAME" up

echo "=== Port forward host:${HOST_PORT} → VM:22 ==="
iptables -t nat -C PREROUTING -p tcp --dport "$HOST_PORT" \
    -j DNAT --to-destination "${VM_IP}:22" 2>/dev/null || \
    iptables -t nat -A PREROUTING -p tcp --dport "$HOST_PORT" \
    -j DNAT --to-destination "${VM_IP}:22"

iptables -C FORWARD -p tcp -d "$VM_IP" --dport 22 -j ACCEPT 2>/dev/null || \
    iptables -I FORWARD 1 -p tcp -d "$VM_IP" --dport 22 -j ACCEPT

echo "=== Done ==="
echo "Bridge : $BRIDGE ($BRIDGE_IP)"
echo "TAP    : $TAP_NAME"
echo "VM IP  : $VM_IP"
echo "Hosts connect on port $HOST_PORT (DNAT → ${VM_IP}:22)"
