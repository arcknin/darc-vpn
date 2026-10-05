# Evidência local — 2026-10-04

- Flutter analyze: sem problemas. Flutter test: 20 testes passando (modelos/config, contrato HTTP, sessão, widgets e controlador de túnel com fake).
- TDD: teste de polling desconectado durante consentimento falhou antes da correção e passou depois. A tela preserva operação pendente sem declarar VPN ativa.
- Backend: testes Go com race e vet passaram nesta execução; isso é evidência local, não implantação na VPS.
- Ponte Kotlin, serviço foreground e dependência oficial WireGuard compilaram. O último build incremental terminou em 6,9 segundos.
- APK: `front-vpn/build/app/outputs/flutter-apk/app-debug.apk`, aproximadamente 92 MiB; SHA256 `3683328dec39157652b57455f794a240ede95e2d48702643db912d7525ad07a6`.
- Build: `flutter build apk --debug --target-platform android-arm64 --dart-define=API_BASE_URL=http://127.0.0.1:18082 --dart-define=ALLOW_INSECURE_API=true --dart-define=PRIVATE_TEST=true`.
- Pacote inspecionado com aapt/unzip: minSdk24, targetSdk36, debuggable, cleartext debug habilitado, serviço próprio não exportado com tipo systemExempted, sem serviço base duplicado. Engine Flutter somente arm64-v8a; a biblioteca WireGuard inclui também outras ABIs, o que não torna este APK compatível com Flutter em aparelhos 32 bits/x86.

## Limite da prova

ADB não encontrou dispositivo conectado nas consultas desta execução. Não houve instalação, consentimento Android real, login na API real, handshake ou tráfego pelo túnel deste aplicativo. Compilação e testes fake não fecham essa prova.

O acesso SSH/adb reverse e a conta nova de teste continuam pendentes conforme [roteiro USB](android-usb-test.md). Não houve alteração de root, contas/dados reais, imagem API na VPS ou publicação.
