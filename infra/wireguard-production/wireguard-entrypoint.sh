#!/bin/sh
set -eu
umask 077
: "${WG_ADDRESS4:?}" "${WG_ADDRESS6:?}" "${WG_POOL4:?}" "${WG_POOL6:?}" "${WG_PORT:?}"
mkdir -p /identity
if [ ! -s /identity/server.key ]; then echo "Existing server identity required; refusing generation" >&2; exit 1; fi
# Startup owns the empty interface; peer state is written only by the Go API.
ip link add wg0 type wireguard
wg set wg0 private-key /identity/server.key listen-port "$WG_PORT"
ip address add "$WG_ADDRESS4" dev wg0
ip -6 address add "$WG_ADDRESS6" dev wg0
ip link set wg0 mtu 1420 up
if [ "${WG_COLD_TEST:-0}" != 1 ]; then
iptables -A FORWARD -i wg0 -o eth0 -j ACCEPT
iptables -A FORWARD -i eth0 -o wg0 -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
iptables -t nat -A POSTROUTING -s "$WG_POOL4" -o eth0 -j MASQUERADE
ip6tables -A FORWARD -i wg0 -o eth0 -j ACCEPT
ip6tables -A FORWARD -i eth0 -o wg0 -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
ip6tables -t nat -A POSTROUTING -s "$WG_POOL6" -o eth0 -j MASQUERADE
fi
finish(){ ip link delete wg0 2>/dev/null || true; exit 0; }
trap finish TERM INT
while :; do sleep 3600 & wait "$!" || true; done
