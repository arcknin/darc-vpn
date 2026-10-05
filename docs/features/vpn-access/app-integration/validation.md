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

## Retomada USB — 2026-10-05

Aparelho Android API35/arm64-v8a conectado e processo Darc VPN presente. Logs filtrados do app não mostraram exceção. Ao relatar indisponibilidade, não havia adb reverse nem listener SSH no Mac; configurados reverse18082 e túnel SSH loopback18082. Listener e reverse confirmados; checagem interna da API VPS passou. Login real e tráfego do túnel do aplicativo ainda aguardam nova tentativa do dono; isso não é prova de autenticação/handshake.

## APK de teste HTTPS — 2026-10-05

Dono informou que funcionou após restaurar transporte USB/SSH e pediu teste independente do Mac. Build profile/AOT concluído em 131,9s: `darc-vpn-private-test-https.apk`, versão1.0.0-test.2/code2, aproximadamente39,5MB, engine ARM64/API24+. PRIVATE_TEST=true, ALLOW_INSECURE_API=false, API pública HTTPS. URL `https://vpn.darcanix.com` encontrada no binário compilado. Manifesto final usesCleartextTraffic=false; profile permanece debuggable para diagnóstico, não é distribuição definitiva.

Assinatura verificada com apksigner: CN=Android Debug; certificado SHA256 `5c57f3ec118ea2f04d75f4afc06f2857d15d9ac80529c9f55602993fa3e372b0`, idêntico ao debug anterior. SHA256 APK `f48ea4a23675c54622ccc611a2f1db367b10124255b91d6a7a1fb15c91681fdf`.

Não houve instalação automática, atualização da VPS ou login público por este APK. Aprovação automática rejeitou a consulta HTTPS de validação com motivo `Sem chamadas externas`; sem contorno. Login público/handshake/tráfego dependem de teste no aparelho.
