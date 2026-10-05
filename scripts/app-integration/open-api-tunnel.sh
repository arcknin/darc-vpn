#!/bin/bash
set -Eeuo pipefail
exec ssh -o BatchMode=yes -o ExitOnForwardFailure=yes -o ServerAliveInterval=30 \
  -N -L 127.0.0.1:18082:127.0.0.1:18082 darc-vpn
