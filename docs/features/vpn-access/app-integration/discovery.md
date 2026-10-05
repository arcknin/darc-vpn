# Discovery: continuidade app Android e backend

Dono pediu continuidade front/back após confirmar corte/04 da migração privada e escolheu validação em Android USB. Fonte operacional é o relato do dono: 03 com peers/PSKs preservados, testes Mac IPv4/IPv6/HTTPS e 04 Verified; não atribuir esse relato a inspeção root independente.

Fontes locais lidas: front-vpn/lib/{main,app}.dart, src/{models,session_controller,qr_config}.dart, Android manifests/Gradle/MainActivity, testes B06; backend TermsDocument/service e composição main. Projetos sem Git e sem AGENTS.md local encontrado; preservar arquivos e guias da conversa.

Lacunas verificadas: QR usa chave pública do cliente em lugar do servidor e porta fixa51820; DTO privado do backend já fornece serverPublicKey/endpointPort. App exige aceite mesmo no modo privado sem termos. Android tem FlutterActivity sem ponte/VpnService; B07/B08 permanecem abertos. API na VPS é loopback18082; nenhum HTTPS/FQDN novo aprovado.

Caminho: corrigir contrato/serialização e modo privado, integrar biblioteca oficial WireGuard Android por ponte nativa e validar com testes, APK debug e depois Android USB via SSH+adb reverse. Nenhum acesso root/dado real/publicação é inferido. Runtime pede aparelho disponível e conta de teste criada pelo dono; não pedir segredos no chat.

Fontes primárias consultadas: WireGuard embedding, GoBackend/Tunnel no repositório oficial, Maven Central com.wireguard.android:tunnel 1.0.20260102 (minSdk24). Lifecycle da biblioteca confirma callbacks DOWN ao destruir serviço; UP indica interface local ativa, não handshake/saída de internet.
