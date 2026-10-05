#!/bin/sh
set -eu
umask 077
mkdir -p /data/acceptances
if [ ! -s /data/master.hex ]; then
  if [ -s /data/vpn.db ]; then echo "Persistent database present but master key absent; refusing regeneration" >&2; exit 1; fi
  od -An -N32 -tx1 /dev/urandom | tr -d ' \n' > /data/master.hex
fi
export VPN_MASTER_KEY_HEX="$(cat /data/master.hex)"
if [ "${VPN_MODE:-local}" != container-private-test ]; then
export VPN_TERMS_SHA256="$(sha256sum "${VPN_TERMS_PATH:-/app/terms.txt}" | cut -d ' ' -f1)"
fi
exec /app/api
