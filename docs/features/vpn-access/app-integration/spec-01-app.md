---
schema_version: 1
managed_by: spec-drive
feature: vpn-access
task: app-integration
revision: 2
status: authorized_local_continuation
---

# Spec integrada: aplicativo Android consumindo backend WireGuard

## Contexto e escopo

Emenda autorizada em 2026-10-05: dono pediu teste semelhante à produção, independente de USB/SSH. APK de teste profile/AOT usa `https://vpn.darcanix.com`, cleartext desabilitado, assinatura de desenvolvimento e PRIVATE_TEST=true. Mantém teste privado sem termos; build release/publicação definitiva continua fora. USB fica disponível para diagnóstico, não como transporte obrigatório da API.

[Discovery](discovery.md). Continuar MVP aprovado (B06/B07/B08), com emendas técnicas decorrentes da API real e ordem expressa de teste privado sem termos. Pedido atual autoriza código e validação local; escolha Android USB direciona prova runtime. Sem root, mutação de contas reais, publicação, release assinada ou DNS/proxy neste trabalho.

## Objetivo e comportamento

- DTO privado valida serverPublicKey e endpointPort; config [Peer] usa chave do servidor e porta da API, conservando chave/identidade do cliente e PSK.
- Backend informa acceptanceRequired explicitamente no documento de termos. App usa sinal false em build privada debug/profile para pular aceite sem gravar consentimento falso; compatibilidade com API v3 reservada private-technical-test até atualização. Release/fluxo normal mantém gate obrigatório.
- Login carrega perfis e principal/última seleção. Selecionar não conecta; alterar seleção com túnel ativo não migra. Conexão só por botão e pelo perfil selecionado.
- Native bridge prepare/start/stop/state via MethodChannel/EventChannel, biblioteca oficial WireGuard tunnel versão fixada, API24+. Negação da permissão, falha ou revogação retorna estado real desconectado/erro. Nunca imprimir config/PrivateKey/PSK nos erros.
- UI distingue interface VPN ativa de handshake/internet verificados. Callback/consulta local decide estado; retorno de pedido não basta. Logout desliga túnel controlado pelo app antes de limpar sessão. Não persistir config secreta nem fazer auto-conexão.
- Acesso principal de teste: HTTPS público em vpn.darcanix.com. Alternativa técnica: SSH local18082 → VPS loopback18082 com Android USB adb reverse. HTTP somente em debug explícita; profile/release continuam cleartext false. Sem alterar endpoint UDP da VPN.

## Ordem e arquivos

1. TDD nos modelos/QR/sessão e metadado backend; manter testes B06 normais.
2. TDD na máquina de estados e fake bridge; integrar ponte nativa e UI.
3. Analyze/tests/contrato, build APK debug; testes nativos quando ambiente disponível.
4. Preparar comando/runbook de teste USB e prova integrada; conta privada/credenciais fornecidas no aparelho pelo dono, não coletadas no chat. Instalação/conexão somente no dispositivo identificado com usuário informado.

Escopo: back-vpn/domain/terms/main/OpenAPI/tests e front-vpn/lib/android/tests/README; docs desta tarefa. Evitar mexer na migração em produção. Principal mantém execução; sem reviewer automático.

## Aceitação e provas

- [ ] QR/config do túnel usa server key/porta corretos e contém o mesmo segredo de perfil.
- [ ] Modo debug privado passa sem aceite; termosAccepted não é falsamente marcado e normal/release exige aceite.
- [ ] Estado de túnel deriva de callback/consulta; start pendente/falha/negação não vira conectado.
- [ ] Seleção não conecta e logout desconecta; corridas não instalam config após logout.
- [ ] Runtime Android prepara consentimento e inicia/para serviço, com diagnóstico sanitizado. UP local não prova handshake.
- [ ] Flutter analyze/tests/build debug e contrato backend passam; evidência live fica separada de fake/build.

TDD: yes, continuidade da escolha confirmada. Go race/vet e Flutter testes de lógica/widget; Gradle compile e testes nativos proporcionais. API24 e Android atual quando devices/emulador disponíveis; limitações explicitadas sem registrar skip como green. Não instalar SDK global ou apagar caches alheios para resolver espaço.

## Autoridade e retomada

Execução local em continuidade do pedido do dono; uso USB selecionado. Root/dados de produção fora; mudança da API real necessita implantação pelo operador separado. Privado continua habilitado; termos/retencão não bloqueiam esta entrega. [Checkpoint](app-integration-checkpoint.yaml) indica progresso/dispositivo/credenciais pendentes.
