#!/usr/bin/env bash
set -Eeuo pipefail
umask 077
if [[ ${1:-} == --help ]]; then
    echo 'update.sh [--image sha256:...]: build/update only API in the existing verified VPN stack.'
    exit 0
fi
[[ $EUID -eq 0 ]] || { echo 'Execute como root.' >&2; exit 1; }
REPO=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)
STATE=/var/lib/darc-vpn-migration
PROJECT=darc-vpn-prod-main
BASE=$STATE/runtime-model.json
OVERLAY=$REPO/infra/wireguard-production/compose.update-api.yaml
for cmd in docker python3 flock tar install; do command -v "$cmd" >/dev/null; done
[[ $(cat "$STATE/stage") == verified ]] || { echo 'A migração deve estar verificada antes de atualizar.' >&2; exit 1; }
exec 9>"$STATE/operation.lock"
flock -n 9 || { echo 'Outra operação da VPN está em andamento.' >&2; exit 1; }
run=$(mktemp -d "$STATE/api-update.XXXXXXXX")
compose=(docker compose --project-directory "$STATE" -p "$PROJECT" -f "$BASE")
api=$("${compose[@]}" ps -q api)
wg=$("${compose[@]}" ps -q wireguard)
[[ -n $api && -n $wg ]] || { echo 'Stack ativa não encontrada.' >&2; exit 1; }
old_image=$(docker inspect --format '{{.Image}}' "$api")
wg_start=$(docker inspect --format '{{.State.StartedAt}}' "$wg")
cp -- "$BASE" "$run/compose-before.json"
if [[ $# == 0 ]]; then
    tag="darc-vpn-api:update-$(date -u +%Y%m%dT%H%M%SZ)-$$"
    docker build -f "$REPO/infra/wireguard-production/Dockerfile.api" -t "$tag" "$REPO"
    image=$(docker image inspect --format '{{.Id}}' "$tag")
elif [[ $# == 2 && $1 == --image ]]; then
    image=$2
    [[ $image =~ ^sha256:[a-f0-9]{64}$ ]] || { echo 'Use um ID sha256 de imagem já construída.' >&2; exit 2; }
    docker image inspect "$image" >/dev/null
else
    echo 'Use update.sh ou update.sh --image sha256:...' >&2; exit 2
fi
# Prepare candidate/rollback models without changing runtime secrets or VPN settings.
data=$(python3 "$REPO/scripts/api-update/model.py" "$BASE" "$image" "$run/compose-next.json")
python3 "$REPO/scripts/api-update/model.py" "$BASE" "$old_image" "$run/compose-rollback.json" >/dev/null
export DARC_API_IMAGE=$image
next=("${compose[@]}" -f "$OVERLAY")
"${next[@]}" config --quiet >"$run/validation.log" 2>&1
rollback_needed=0
finish() {
    local result=$?
    trap - EXIT
    if [[ $result != 0 && $rollback_needed == 1 ]]; then
        if docker compose --project-directory "$STATE" -p "$PROJECT" -f "$run/compose-rollback.json" \
            up -d --no-deps --no-build --pull never --wait --wait-timeout 120 api >"$run/rollback.log" 2>&1; then
            install -m 600 "$run/compose-rollback.json" "$STATE/runtime-model.rollback.tmp"
            mv -f "$STATE/runtime-model.rollback.tmp" "$BASE"
            echo 'API voltou à imagem anterior; dados não foram restaurados automaticamente.' >&2
        else
            echo "Rollback da imagem não ficou saudável; revisão root necessária. Logs privados: $run" >&2
        fi
    fi
    [[ $result == 0 ]] || echo "Atualização não concluída. Evidência privada: $run" >&2
    exit "$result"
}
trap finish EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
rollback_needed=1
"${compose[@]}" stop api >"$run/stop.log" 2>&1
# API stopped: coherent database/key/receipts backup; WireGuard continues forwarding.
tar -C "$data" -czpf "$run/data-before.tar.gz" .
"${next[@]}" up -d --no-deps --no-build --pull never --wait --wait-timeout 120 api >"$run/update.log" 2>&1
"${next[@]}" exec -T api /app/api health >"$run/health.log" 2>&1
[[ $("${next[@]}" ps -q wireguard) == "$wg" ]]
[[ $(docker inspect --format '{{.State.StartedAt}}' "$wg") == "$wg_start" ]]
install -m 600 "$run/compose-next.json" "$STATE/runtime-model.next.tmp"
mv -f "$STATE/runtime-model.next.tmp" "$BASE"
rollback_needed=0
echo "API_UPDATE_OK; WireGuard não recriado/reiniciado. Backup privado: $run"
