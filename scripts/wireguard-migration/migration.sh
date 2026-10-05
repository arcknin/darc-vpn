#!/bin/bash
# Operator-run only. Never run this bundle as root from a user-writable checkout.
set +x
set -Eeuo pipefail
umask 077
export PATH=/usr/sbin:/usr/bin:/sbin:/bin
export PYTHONDONTWRITEBYTECODE=1
BUNDLE=$(cd -P -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
SELF="$BUNDLE/migration.sh"
HELPER="$BUNDLE/inventory.py"
STATE=/var/lib/darc-vpn-migration
COMMAND=${1:-help}
[[ $# -eq 0 ]] || shift
PLAN=
CONFIRM=0
while [[ $# -gt 0 ]]; do
  case "$1" in
    --state) STATE=${2:?}; shift 2 ;;
    --plan) PLAN=${2:?}; shift 2 ;;
    --confirm-cutover) CONFIRM=1; shift ;;
    --help|-h) COMMAND=help; shift ;;
    *) printf 'Unknown option\n' >&2; exit 2 ;;
  esac
done
die(){ printf '%s\n' "$*" >&2; exit 1; }
help(){ cat <<'HELP'
Darc VPN migration bundle (root operator; no action by upload).
  migration.sh inspect
  migration.sh prepare [--plan /root/production-plan.json]
  migration.sh cutover --confirm-cutover
  migration.sh verify
  migration.sh rollback
inspect/prepare capture protected reports/backups only; they do not stop WG.
cutover requires a reviewed production Compose and matching restore evidence.
The current lab stack cannot import legacy peers/PSKs: do not use it for cutover.
First freeze this bundle in a root-owned directory and verify SHA256SUMS.
State defaults to /var/lib/darc-vpn-migration; only /root or /var/lib state allowed.
HELP
}
case "$COMMAND" in help|--help|-h) help; exit 0 ;; inspect|prepare|cutover|verify|rollback) ;; *) help; exit 2 ;; esac
[[ $EUID -eq 0 ]] || die 'Root required. No operation performed.'
[[ $(uname -s) == Linux ]] || die 'Linux VPS required.'
[[ $(hostname -s) == srv1519215 ]] || die 'Unexpected VPS; refusing operation.'
[[ ! -L "$SELF" && $(stat -c %u "$SELF") == 0 && $(stat -c %u "$HELPER") == 0 ]] || die 'Freeze/verify bundle under /root before running it; codex-owned scripts are refused.'
python3 "$HELPER" secure "$BUNDLE" directory
python3 "$HELPER" secure "$SELF"
python3 "$HELPER" secure "$HELPER"
(cd "$BUNDLE" && sha256sum -c SHA256SUMS >/dev/null) || die 'Bundle checksum verification failed.'
case "$STATE" in /var/lib/darc-vpn-migration|/root/darc-vpn-migration-state) ;; *) die 'Unsupported state path.' ;; esac
if [[ -e "$STATE" ]]; then python3 "$HELPER" secure "$STATE" directory; else install -d -m 700 "$STATE"; fi
chmod 700 "$STATE"
exec 9>"$STATE/operation.lock"
if [[ $COMMAND == rollback ]]; then flock -w 120 9 || die 'Rollback could not acquire lock; inspect state manually.'; else flock -n 9 || die 'Another migration operation is running.'; fi
for cmd in python3 wg ip systemctl docker sha256sum install timeout ss; do command -v "$cmd" >/dev/null || die "Missing dependency: $cmd"; done
stage(){ printf '%s\n' "$1" > "$STATE/stage.tmp"; mv -f -- "$STATE/stage.tmp" "$STATE/stage"; }
run_id(){ [[ -f "$STATE/run-id" ]] || die 'Run prepare first.'; RUN=$(cat "$STATE/run-id"); [[ $RUN =~ ^[0-9]{8}T[0-9]{6}Z-[0-9]+$ ]] || die 'Invalid run ID.'; BACKUP="$STATE/backups/$RUN"; python3 "$HELPER" secure "$BACKUP" directory; }
get(){ python3 "$HELPER" get "$STATE/plan.json" "$1"; }
compose_original_args(){ COMPOSE_ARGS=(docker compose --project-directory "$(dirname -- "$(get compose_file)")" --env-file "$(get env_file)" -p "$(get project)" -f "$(get compose_file)"); }
compose_original(){ compose_original_args; "${COMPOSE_ARGS[@]}" "$@"; }
compose_args(){ COMPOSE_ARGS=(docker compose --project-directory "$STATE" -p "$(get project)" -f "$STATE/runtime-model.json"); }
compose(){ compose_args; "${COMPOSE_ARGS[@]}" "$@"; }
private_log(){ printf 'Details retained root-only: %s\n' "$1" >&2; }
collect(){
  local out=$1
  systemctl is-active --quiet wg-quick@wg0 || die 'Host wg0 must be active for inspection/preparation.'
  for cmd in iptables-save ip6tables-save tar; do command -v "$cmd" >/dev/null || die "Missing dependency: $cmd"; done
  python3 "$HELPER" secure /etc/wireguard/wg0.conf
  install -d -m 700 "$out"
  install -m 600 /etc/wireguard/wg0.conf "$out/wg0.conf"
  wg showconf wg0 > "$out/live.conf"
  wg show wg0 dump > "$out/live.dump"
  ip -j address show wg0 > "$out/interface.json"
  ip -j route show table all > "$out/routes4.json"
  ip -6 -j route show table all > "$out/routes6.json"
  iptables-save > "$out/firewall4.rules"
  ip6tables-save > "$out/firewall6.rules"
  systemctl cat wg-quick@wg0 > "$out/service.txt"
  systemctl is-enabled wg-quick@wg0 > "$out/enabled.txt" 2>/dev/null || true
  docker network ls -q > "$out/network-ids.txt"
  if [[ -s "$out/network-ids.txt" ]]; then mapfile -t ids < "$out/network-ids.txt"; docker network inspect "${ids[@]}" > "$out/networks.json"; fi
  stat -c '%a %u %g' /etc/wireguard/wg0.conf > "$out/config-mode.txt"
  python3 "$HELPER" summary "$out/live.conf" "$out/wg0.conf" "$out/interface.json" > "$out/sanitized-summary.json"
}
render_plan(){
  python3 "$HELPER" plan "$STATE/plan.json" > /dev/null
  # An unresolved model catches additional env files before their values are read.
  compose_original config --no-env-resolution --format json > "$STATE/model-unresolved.json" 2> "$STATE/plan.log"
  python3 "$HELPER" model "$STATE/model-unresolved.json"
  compose_original config --format json > "$STATE/model.json" 2>> "$STATE/plan.log"
  python3 "$HELPER" model "$STATE/model.json"
  python3 "$HELPER" proof "$BACKUP/live.conf" "$(get restore_proof)" "$STATE/model.json"
  compose_original config --images > "$STATE/images.txt"
  while IFS= read -r image; do docker image inspect "$image" > /dev/null 2>> "$STATE/plan.log" || die 'Pinned target image missing; build/validate deployment separately.'; done < "$STATE/images.txt"
  python3 "$HELPER" freeze "$STATE/model.json" > "$STATE/runtime-model.json"
  sha256sum "$STATE/model.json" > "$STATE/model.sha256"
}
plan_drift_check(){
  python3 "$HELPER" plan "$STATE/plan.json" > /dev/null
  compose_original config --format json > "$STATE/model-check.json" 2>> "$STATE/plan.log"
  cmp -s "$STATE/model.json" "$STATE/model-check.json" || die 'Target Compose/environment changed; rerun preparation.'
  python3 "$HELPER" model "$STATE/model-check.json"
  python3 "$HELPER" proof "$BACKUP/live.conf" "$(get restore_proof)" "$STATE/model-check.json"
}
rollback_core(){
  run_id
  [[ -f "$STATE/rollback-armed" ]] || { printf 'No armed cutover; nothing to rollback.\n'; return 0; }
  local stage_now
  stage_now=$(cat "$STATE/stage" 2>/dev/null || true)
  [[ $stage_now != verified ]] || { printf 'Already verified; refusing timer rollback.\n'; return 0; }
  stage rolling-back
  # Keep target volumes/data for diagnosis. Never restore the global firewall snapshot.
  if ! compose stop > "$STATE/rollback.log" 2>&1; then private_log "$STATE/rollback.log"; die 'Target stop failed; host VPN was not started into a port conflict.'; fi
  if ! compose down --remove-orphans >> "$STATE/rollback.log" 2>&1; then
    private_log "$STATE/rollback.log"
    [[ -z $(compose ps -q --status running) ]] || die 'Target still running; port conflict must be resolved manually.'
  fi
  if systemctl is-active --quiet wg-quick@wg0; then
    wg showconf wg0 > "$STATE/rollback-live.conf"
    python3 "$HELPER" compare "$BACKUP/live.conf" "$STATE/rollback-live.conf" || die 'Active host differs from backup; manual recovery required.'
    if [[ $(cat "$BACKUP/enabled.txt") == enabled ]]; then systemctl enable wg-quick@wg0 >> "$STATE/rollback.log" 2>&1; else systemctl disable wg-quick@wg0 >> "$STATE/rollback.log" 2>&1; fi
    rm -f -- "$STATE/rollback-armed"
    stage rolled-back
    printf 'Host was already active with original peers; target stopped and backup retained.\n'
    return 0
  fi
  if ss -H -lun | grep -Eq '(^|[[:space:]])[^[:space:]]*:51442([[:space:]]|$)'; then die 'UDP 51442 still occupied; refusing simultaneous host startup.'; fi
  install -m 600 "$BACKUP/wg0.conf" /etc/wireguard/wg0.conf
  if ! timeout 60 systemctl start wg-quick@wg0 >> "$STATE/rollback.log" 2>&1; then private_log "$STATE/rollback.log"; die 'Automatic recovery failed; backup retained. Repair host unit/interface manually.'; fi
  if [[ $(cat "$BACKUP/enabled.txt") == enabled ]]; then systemctl enable wg-quick@wg0 >> "$STATE/rollback.log" 2>&1; else systemctl disable wg-quick@wg0 >> "$STATE/rollback.log" 2>&1; fi
  wg showconf wg0 > "$STATE/rollback-live.conf"
  python3 "$HELPER" compare "$BACKUP/live.conf" "$STATE/rollback-live.conf" || die 'Restored host peer state differs from backup; manual recovery required.'
  systemctl is-active --quiet wg-quick@wg0 || die 'Restored host service is not active.'
  rm -f -- "$STATE/rollback-armed"
  stage rolled-back
  printf 'Host wg0 restored. Backups and target data retained; no firewall flush performed.\n'
}
cutover_exit(){
  local rc=$1
  trap - EXIT INT TERM HUP
  if [[ $rc -ne 0 ]]; then printf 'Cutover failed; attempting rollback.\n' >&2; rollback_core; fi
  exit "$rc"
}
case "$COMMAND" in
 inspect)
  OUT="$STATE/inspections/$(date -u +%Y%m%dT%H%M%SZ)-$$"
  collect "$OUT"
  cat "$OUT/sanitized-summary.json"
  printf 'Private inventory: %s\nOnly sanitized-summary.json may be shared; raw files contain keys.\n' "$OUT"
  ;;
 prepare)
  [[ ! -f "$STATE/rollback-armed" ]] || die 'Pending cutover: run verify or rollback, not prepare.'
  RUN="$(date -u +%Y%m%dT%H%M%SZ)-$$"; BACKUP="$STATE/backups/$RUN"
  collect "$BACKUP"
  # Do not overwrite a config changed only at runtime; rollback must restore all peers.
  python3 "$HELPER" compare "$BACKUP/wg0.conf" "$BACKUP/live.conf" || die 'Config/runtime drift. Backup retained; reconcile persistent host configuration before migration.'
  (cd "$BACKUP" && sha256sum wg0.conf live.conf live.dump interface.json firewall4.rules firewall6.rules > SHA256SUMS)
  printf '%s\n' "$RUN" > "$STATE/run-id"
  stage backup-only
  if [[ -z "$PLAN" ]]; then
    printf 'Protected backup prepared: %s\nNo production plan supplied: cutover remains blocked.\n' "$BACKUP"
    exit 0
  fi
  python3 "$HELPER" secure "$PLAN"
  install -m 600 "$PLAN" "$STATE/plan.json"
  render_plan
  stage prepared
  printf 'Backup and target plan validated. Host VPN was not stopped.\n'
  ;;
 cutover)
  [[ $CONFIRM -eq 1 ]] || die 'Explicit --confirm-cutover is required; no host service changed.'
  run_id
  [[ $(cat "$STATE/stage") == prepared ]] || die 'Cutover blocked: prepare a reviewed production plan with legacy-peer/PSK restore evidence first.'
  (cd "$BACKUP" && sha256sum -c SHA256SUMS > /dev/null) || die 'Backup integrity failure.'
  plan_drift_check
  systemctl is-active --quiet wg-quick@wg0 || die 'Source service not active.'
  cmp -s /etc/wireguard/wg0.conf "$BACKUP/wg0.conf" || die 'Host config changed after backup; rerun preparation.'
  wg showconf wg0 > "$STATE/pre-cutover.conf"
  python3 "$HELPER" compare "$BACKUP/live.conf" "$STATE/pre-cutover.conf"
  [[ -z $(compose ps -aq) ]] || die 'Target project already has containers; no blind replacement.'
  command -v systemd-run >/dev/null || die 'Persistent rollback scheduler missing.'
  TIMER="darc-vpn-migration-rollback-$RUN"
  printf '%s\n' "$TIMER" > "$STATE/timer-unit"
  touch "$STATE/rollback-armed"
  stage cutover-starting
  # Run in an independent root service even if the operator loses SSH.
  trap 'cutover_exit "$?"' EXIT
  systemd-run --unit="$TIMER" --on-active="$(get rollback_delay_seconds)s" --timer-property=AccuracySec=1s /bin/bash "$SELF" rollback --state "$STATE" > "$STATE/timer.log" 2>&1
  trap 'exit 130' INT TERM HUP
  if ! timeout 60 systemctl stop wg-quick@wg0 > "$STATE/cutover.log" 2>&1; then private_log "$STATE/cutover.log"; die 'Host stop failed.'; fi
  systemctl disable wg-quick@wg0 >> "$STATE/cutover.log" 2>&1
  compose_args
  if ! timeout 180 "${COMPOSE_ARGS[@]}" up -d --no-build --pull never --wait --wait-timeout 150 >> "$STATE/cutover.log" 2>&1; then private_log "$STATE/cutover.log"; die 'Target startup/readiness failed.'; fi
  compose exec -T wireguard wg showconf wg0 > "$STATE/target-live.conf" 2>> "$STATE/cutover.log"
  python3 "$HELPER" compare "$BACKUP/live.conf" "$STATE/target-live.conf"
  sleep 10
  compose exec -T wireguard wg showconf wg0 > "$STATE/target-live-second.conf" 2>> "$STATE/cutover.log"
  python3 "$HELPER" compare "$BACKUP/live.conf" "$STATE/target-live-second.conf"
  stage awaiting-client-verification
  trap - EXIT INT TERM HUP
  printf 'Target running with preserved identity/peers/PSKs. Rollback timer remains armed.\nRun verify with external client attestation for run %s; otherwise automatic rollback occurs.\n' "$RUN"
  ;;
 verify)
  run_id
  [[ $(cat "$STATE/stage") == awaiting-client-verification ]] || die 'No pending target verification.'
  plan_drift_check
  compose exec -T api /app/api health > "$STATE/verify-health.log" 2>&1 || die 'API not ready. Timer remains armed.'
  compose exec -T wireguard wg showconf wg0 > "$STATE/verify-live.conf" 2>> "$STATE/verify-health.log"
  python3 "$HELPER" compare "$BACKUP/live.conf" "$STATE/verify-live.conf"
  python3 "$HELPER" traffic "$(get traffic_proof)" "$RUN"
  systemctl is-active --quiet wg-quick@wg0 && die 'Host and target services both active; inspect conflict.'
  TIMER=$(cat "$STATE/timer-unit")
  systemctl stop "$TIMER.timer"
  # A timer-fired rollback process may be waiting for our lock: mark completion first.
  stage verified
  rm -f -- "$STATE/rollback-armed"
  printf 'Verified using kernel comparison, API readiness and operator external-client attestation. Backup retained.\n'
  ;;
 rollback) rollback_core ;;
esac
