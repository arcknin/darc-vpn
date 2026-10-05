# Validação privada Android USB

## Teste HTTPS independente do Mac

O dono pediu em 2026-10-05 que o APK principal de teste use o endpoint público:

```bash
bash scripts/app-integration/build-https-test-apk.sh
```

Saída: `front-vpn/build/app/outputs/flutter-apk/darc-vpn-private-test-https.apk`. Build profile/AOT, versão1.0.0-test.2/code2, ARM64/API24+, API `https://vpn.darcanix.com`, HTTP desabilitado, assinatura de desenvolvimento. PRIVATE_TEST=true preserva teste sem aceite. Não é release assinada para distribuição definitiva.

Esse APK não precisa de adb reverse nem SSH no Mac. USB pode servir para instalação/logs. API HTTPS/Nginx e endpoint UDP51442 ainda precisam funcionar na VPS; login real e VPN devem ser validados no aparelho. A consulta HTTPS automática foi rejeitada pela aprovação automática (`Sem chamadas externas`), sem tentativa de contorno.

## Transporte USB alternativo / histórico

A implementação local compilou; [evidência](validation.md). APK debug usa API_BASE_URL=http://127.0.0.1:18082, ALLOW_INSECURE_API=true, PRIVATE_TEST=true. Não publicar esse build. Release mantém cleartext false e fluxo normal de aceite.

1. Conectar Android, ativar depuração USB e autorizar o Mac. `adb devices -l` deve mostrar um único device autorizado; nenhum device foi encontrado no primeiro inventário.
2. Abrir túnel em janela separada: `bash scripts/app-integration/open-api-tunnel.sh`. Isso acessa somente API privada; não altera a VPS.
3. Confirmar `adb -s <serial> shell getprop ro.product.cpu.abi`: o APK atual requer arm64-v8a (Android API24+). Para dispositivo identificado: `adb -s <serial> reverse tcp:18082 tcp:18082`. Verificar API via túnel antes de instalar APK debug no dispositivo informado.
4. Login utiliza conta de amigo nova, criada pelo administrador para teste. Os três peers legados não são contas do app e não têm PrivateKey de cliente recuperável; não converter/copiar chaves existentes para forçar esse teste. Credenciais não devem ser coladas no chat.
5. Entrar no app privado sem aceite, conferir principal/extra e seleção; selecionar não conecta. Pedir Ligar VPN, autorizar consentimento Android e observar estado confirmado. “VPN ativa no aparelho” não equivale a handshake/internet.
6. Testar tráfego/HTTPS v4/v6 no Android, mudança de seleção sem migração automática, desligar, logout e revogação externa pela tela de VPN. Não registrar config/PrivateKey/PSK/token em logs/evidência.

A biblioteca oficial é `com.wireguard.android:tunnel:1.0.20260102`; ponte nativa usa GoBackend com serviço foreground próprio e VpnService.prepare. Metadata de estado é sanitizada. API informa acceptanceRequired na revisão local; compatibilidade debug com v3 detecta a versão reservada private-technical-test quando campo novo não existe.

Não modificar root/config/contas reais da VPS automaticamente. Conta de teste e eventual atualização da imagem API pertencem ao operador. Atualização local de contrato não é prova de implantação da nova revisão na VPS.
