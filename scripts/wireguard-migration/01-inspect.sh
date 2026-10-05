#!/bin/bash
set -Eeuo pipefail
HERE=$(cd -P -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
exec /bin/bash "$HERE/migration.sh" inspect "$@"
