#!/usr/bin/env bash
set -Eeuo pipefail
REPO=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)
cd "$REPO/front-vpn"
# Profile is AOT/optimized; private-test semantics are allowed outside release.
# Main/profile manifests reject cleartext. Signing remains the development key.
flutter build apk --profile --target-platform android-arm64 \
  --build-name=1.0.0-test.2 --build-number=2 \
  --dart-define=API_BASE_URL=https://vpn.darcanix.com \
  --dart-define=ALLOW_INSECURE_API=false \
  --dart-define=PRIVATE_TEST=true
cp build/app/outputs/flutter-apk/app-profile.apk \
  build/app/outputs/flutter-apk/darc-vpn-private-test-https.apk
echo "APK: $REPO/front-vpn/build/app/outputs/flutter-apk/darc-vpn-private-test-https.apk"
