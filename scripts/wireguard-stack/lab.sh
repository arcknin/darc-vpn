#!/bin/sh
set -eu
ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$ROOT"
[ -f .lab.env ] && [ -f .lab-owner ] || { echo 'Missing task ownership files' >&2; exit 1; }
# Read data without evaluating .env as shell code.
LAB_ID=$(sed -n 's/^LAB_ID=//p' .lab.env)
case "$LAB_ID" in *[!a-z0-9-]*|'') echo 'Invalid project characters' >&2; exit 1;; esac
case "$LAB_ID" in darc-vpn-stack-[0-9]*) ;; *) echo 'Invalid project ID' >&2; exit 1;; esac
[ "$(cat .lab-owner)" = "$LAB_ID" ] || { echo 'Ownership mismatch' >&2; exit 1; }
compose(){ docker compose --env-file .lab.env -p "$LAB_ID" -f infra/wireguard-stack/compose.yaml "$@"; }
case "${1:-}" in
 build) compose config --quiet; compose build ;;
 up) compose up -d --wait --wait-timeout 150 ;;
 recreate) compose up -d --force-recreate --wait --wait-timeout 150 ;;
 stop-api) compose stop api ;;
 start-api) compose up -d api --wait --wait-timeout 150 ;;
 restart-wg) compose restart wireguard; compose up -d --force-recreate api --wait --wait-timeout 150 ;;
 check) python3 scripts/wireguard-stack/acceptance.py ;;
 down)
  # Compose only removes resources owned by this exact project. Preserve directory for evidence.
  compose down --volumes --remove-orphans
  if docker ps -aq --filter "label=com.docker.compose.project=$LAB_ID" | read -r ignored; then echo 'Containers remain' >&2; exit 1; fi
  for suffix in api wireguard; do
   image="$LAB_ID-$suffix:1"
   docker image rm "$image" >/dev/null 2>&1 || { echo 'Task image cleanup failed' >&2; exit 1; }
  done
  ;;
 *) echo 'Use build|up|check|recreate|stop-api|start-api|restart-wg|down' >&2; exit 2 ;;
esac
