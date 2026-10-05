#!/bin/bash
# Prepare/import/cold-restore only. Never stops or changes the host wg0.
set +x
set -Eeuo pipefail
umask 077
export PATH=/usr/sbin:/usr/bin:/sbin:/bin
export PYTHONDONTWRITEBYTECODE=1
HERE=$(cd -P -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
ROOT=/opt/darc-vpn-production
STATE=/var/lib/darc-vpn-migration
PRIVATE_TEST=0
SETTINGS=
case "${1:-}" in --help|-h) echo '02b-build-target.sh [--private-test] [--settings /root/production-settings.json]: builds images, imports old peers, tests two cold restores in isolated namespace. Does not stop host VPN.'; exit 0;; esac
while [[ $# -gt 0 ]]; do
  case "$1" in
    --private-test) PRIVATE_TEST=1; shift ;;
    --settings) SETTINGS=${2:?}; shift 2 ;;
    *) echo 'Use --private-test or --settings /root/production-settings.json' >&2; exit 2 ;;
  esac
done
[[ $EUID -eq 0 ]] || { echo 'Root required. No operation performed.' >&2; exit 1; }
[[ $(uname -s) == Linux && $(hostname -s) == srv1519215 ]] || { echo 'Unexpected host' >&2; exit 1; }
python3 "$HERE/inventory.py" secure "$HERE" directory
(cd "$HERE" && sha256sum -c SHA256SUMS >/dev/null)
python3 "$HERE/inventory.py" secure "$STATE" directory
exec 9>"$STATE/operation.lock"; flock -n 9 || { echo 'Another operation active' >&2; exit 1; }
[[ ! -f "$STATE/rollback-armed" ]] || { echo 'Resolve pending cutover first' >&2; exit 1; }
RUN=$(cat "$STATE/run-id"); [[ $RUN =~ ^[0-9]{8}T[0-9]{6}Z-[0-9]+$ ]] || exit 1
BACKUP="$STATE/backups/$RUN"; python3 "$HERE/inventory.py" secure "$BACKUP" directory
(cd "$BACKUP" && sha256sum -c SHA256SUMS >/dev/null)
systemctl is-active --quiet wg-quick@wg0
wg showconf wg0 > "$STATE/02b-source.conf"
python3 "$HERE/inventory.py" compare "$BACKUP/live.conf" "$STATE/02b-source.conf"
if [[ -e "$ROOT" ]]; then python3 "$HERE/inventory.py" secure "$ROOT" directory; else install -d -m 700 "$ROOT"; fi
install -d -m 700 "$ROOT/data" "$ROOT/identity" "$ROOT/terms"
for dir in "$ROOT/data" "$ROOT/identity" "$ROOT/terms"; do python3 "$HERE/inventory.py" secure "$dir" directory; done
for file in "$ROOT/data/master.hex" "$ROOT/data/vpn.db" "$ROOT/identity/server.key"; do if [[ -e "$file" || -L "$file" ]]; then python3 "$HERE/inventory.py" secure "$file"; fi; done
python3 "$HERE/target.py" export "$BACKUP/live.conf" "$BACKUP/interface.json" "$ROOT"
if [[ ! -s "$ROOT/data/master.hex" ]]; then
  [[ ! -s "$ROOT/data/vpn.db" ]] || { echo 'Database exists without persistent master key; refusing regeneration' >&2; exit 1; }
  od -An -N32 -tx1 /dev/urandom | tr -d ' \n' > "$ROOT/data/master.hex"
fi
PROJECT=darc-vpn-prod-main
# Do not mutate a target already deployed. Builds have no production config/keys in context.
[[ -z $(docker ps -aq --filter "label=com.docker.compose.project=$PROJECT") ]] || { echo 'Target already has containers; refuse offline import' >&2; exit 1; }
CODE="$HERE/payload"
python3 "$HERE/inventory.py" secure "$CODE" directory
if ! docker build -f "$CODE/infra/wireguard-production/Dockerfile.api" -t "darc-vpn-prod-api:$RUN" "$CODE" > "$STATE/02b-build-api.log" 2>&1; then echo "Build failed; root-only log: $STATE/02b-build-api.log" >&2; exit 1; fi
if ! docker build -f "$CODE/infra/wireguard-production/Dockerfile.wireguard" -t "darc-vpn-prod-wg:$RUN" "$CODE" > "$STATE/02b-build-wg.log" 2>&1; then echo "Build failed; root-only log: $STATE/02b-build-wg.log" >&2; exit 1; fi
API_IMAGE=$(docker image inspect "darc-vpn-prod-api:$RUN" --format '{{.Id}}')
WG_IMAGE=$(docker image inspect "darc-vpn-prod-wg:$RUN" --format '{{.Id}}')
for image in "$API_IMAGE" "$WG_IMAGE"; do [[ $image =~ ^sha256:[a-f0-9]{64}$ ]] || exit 1; done
# Avoid existing tests/other containers that can write into data before offline import.
python3 "$HERE/target.py" check-mounts "$ROOT/data"
COLD="darc-vpn-cold-$RUN"
COLD_API="$COLD-api"
cleanup(){
  if docker inspect "$COLD_API" >/dev/null 2>&1; then
    [[ $(docker inspect "$COLD_API" --format '{{index .Config.Labels "darc.vpn.cold"}}') == "$RUN" ]] || return 1
    docker rm -f "$COLD_API" >/dev/null
  fi
  if docker inspect "$COLD" >/dev/null 2>&1; then
    [[ $(docker inspect "$COLD" --format '{{index .Config.Labels "darc.vpn.cold"}}') == "$RUN" ]] || return 1
    docker rm -f "$COLD" >/dev/null
  fi
  rm -f -- "$ROOT/legacy-import.json"
}
trap cleanup EXIT
# cli images contain binaries only, not source keys. Never pass PSKs through argv/env.
docker run --rm --network none --cap-drop ALL --entrypoint /app/legacy-import \
  -v "$ROOT/data:/data" -v "$ROOT/legacy-import.json:/input.json:ro" "$API_IMAGE" \
  --db /data/vpn.db --master-key-file /data/master.hex --manifest /input.json
for cycle in 1 2; do
  docker run -d --name "$COLD" --label "darc.vpn.cold=$RUN" --network none --cap-drop ALL --cap-add NET_ADMIN --sysctl net.ipv6.conf.all.disable_ipv6=0 --sysctl net.ipv6.conf.default.disable_ipv6=0 \
    -v "$ROOT/identity:/identity:ro" -e WG_ADDRESS4=10.66.66.1/24 -e WG_ADDRESS6=fd42:42:42::1/64 \
    -e WG_POOL4=10.66.66.0/24 -e WG_POOL6=fd42:42:42::/64 -e WG_PORT=51442 -e WG_COLD_TEST=1 "$WG_IMAGE" >/dev/null
  ready=0
  for _ in {1..30}; do if [[ -n $(docker exec "$COLD" ip -o link show up dev wg0 2>/dev/null) ]]; then ready=1; break; fi; sleep 1; done
  [[ $ready -eq 1 ]] || { echo 'Cold bootstrap did not create interface' >&2; exit 1; }
  for pass in 1 2; do
    docker run --rm --network "container:$COLD" --cap-drop ALL --cap-add NET_ADMIN --entrypoint /app/legacy-import \
      -v "$ROOT/data:/data" "$API_IMAGE" --operation reconcile --db /data/vpn.db --master-key-file /data/master.hex
    docker exec "$COLD" wg showconf wg0 > "$STATE/02b-restored-$cycle-$pass.conf"
    python3 "$HERE/inventory.py" compare "$BACKUP/live.conf" "$STATE/02b-restored-$cycle-$pass.conf"
  done
  docker rm -f "$COLD" >/dev/null
  echo "COLD_RESTORE_$cycle OK (identity, peers, PSKs, AllowedIPs, keepalive)"
done
python3 "$HERE/target.py" materialize "$HERE" "$ROOT" "$STATE" "$RUN" "$API_IMAGE" "$WG_IMAGE" "$SETTINGS" "$PRIVATE_TEST"
if [[ -n "$SETTINGS" || $PRIVATE_TEST -eq 1 ]]; then
  docker run -d --name "$COLD" --label "darc.vpn.cold=$RUN" --network none --cap-drop ALL --cap-add NET_ADMIN --sysctl net.ipv6.conf.all.disable_ipv6=0 --sysctl net.ipv6.conf.default.disable_ipv6=0 \
    -v "$ROOT/identity:/identity:ro" -e WG_ADDRESS4=10.66.66.1/24 -e WG_ADDRESS6=fd42:42:42::1/64 \
    -e WG_POOL4=10.66.66.0/24 -e WG_POOL6=fd42:42:42::/64 -e WG_PORT=51442 -e WG_COLD_TEST=1 "$WG_IMAGE" >/dev/null
  ready=0
  for _ in {1..30}; do if [[ -n $(docker exec "$COLD" ip -o link show up dev wg0 2>/dev/null) ]]; then ready=1; break; fi; sleep 1; done
  [[ $ready -eq 1 ]] || exit 1
  docker run -d --name "$COLD_API" --label "darc.vpn.cold=$RUN" --network "container:$COLD" --cap-drop ALL --cap-add NET_ADMIN \
    --env-file "$ROOT/runtime-api.env" -v "$ROOT/data:/data" -v "$ROOT/terms:/terms:ro" "$API_IMAGE" >/dev/null
  ready=0
  for _ in {1..60}; do if docker exec "$COLD_API" /app/api health >/dev/null 2>&1; then ready=1; break; fi; sleep 1; done
  [[ $ready -eq 1 ]] || { docker logs "$COLD_API" > "$STATE/02b-api-private.log" 2>&1; echo 'API runtime test failed; private log retained, no final restore proof' >&2; exit 1; }
  docker exec "$COLD" wg showconf wg0 > "$STATE/02b-api-restored.conf"
  python3 "$HERE/inventory.py" compare "$BACKUP/live.conf" "$STATE/02b-api-restored.conf"
  docker rm -f "$COLD_API" "$COLD" >/dev/null
  python3 "$HERE/target.py" finalize-proof "$ROOT"
fi

systemctl is-active --quiet wg-quick@wg0
wg showconf wg0 > "$STATE/02b-after.conf"
python3 "$HERE/inventory.py" compare "$BACKUP/live.conf" "$STATE/02b-after.conf"
echo '02b completed: legacy peers imported/encrypted and cold restore verified; host VPN still active.'
echo 'No cutover performed. Follow the target status for remaining production settings/review.'
