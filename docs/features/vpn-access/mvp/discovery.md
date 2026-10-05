# Discovery: retomada do VPN Access MVP

## Request and investigation boundary

- Pedido em 2026-10-04: usar spec-drive para entender o rumo atual antes de continuar.
- Escopo: fontes locais de back-vpn, dashboard-admin-vpn e front-vpn; contrato, plano e checkpoint legados. Sem execução de lotes, builds, testes, SSH, instalação ou mudança operacional nesta investigação.
- Nenhum AGENTS.md encontrado na árvore dos projetos ou nos ancestrais consultados; instruções fornecidas pelo dono na conversa aplicadas.
- Workspace e três projetos sem metadados Git; não há branch/diff verificável. O plano existente pede explicitamente não inicializar Git.

## Verified current behavior

- Contrato aprovado revisão 2 e plano congelado revisão 3 preservados. SHA-256 verificados: contrato `477a54dadf1fd064c5b4e2fe8c355b8d51ac157ca33537e7f8393cfb6972b683`; plano `0561be3fbda9c670f0c24a8bbf0995f76701d9fc9df71d47c873a8a1308a1ee7`. TDD histórico: yes; autoridade histórica: código/testes locais.
- B01-B04: fontes e testes existem. `back-vpn/cmd/api/main.go` compõe SQLite, identidade, perfis cifrados, termos, auditoria e revogação. README do backend ainda descreve B01 e não representa essa composição atual.
- Controlador padrão em `cmd/api/main.go` é `wireguard.NewFake(FakeConfirmAll)`. A porta `internal/domain/ports.go` só define RemovePeers/ConfirmPeersRemoved. `internal/accounts/service.go` cria registros e perfis sem cadastrar peers no servidor. Conclusão de bloqueio com esse fake não comprova revogação externa.
- `addressForProfile` deriva um IPv4 de um byte de hash, em 10.8.0.2-254/32; `migrations/001_b02_identity.sql` não exige unicidade de address. A composição anuncia ::/0 sem atribuir endereço IPv6 de cliente. Necessário definir alocação e comportamento IPv6 antes da integração real.
- B05: dashboard e testes unitários/contratuais, cenário Playwright e runner presentes. O runner `tests/run-playwright.mjs` retorna exit 0 quando Playwright está ausente: isso é skip, não prova de E2E aprovado.
- B06: app, login, aceite, seleção e QR com testes presentes. minSdk=24. Nenhum APK/AAB encontrado na árvore.
- B07: `MainActivity.kt` contém apenas FlutterActivity; sem ponte MethodChannel/VpnService ou dependência WireGuard encontrada nas fontes consultadas. B08: harness de aceitação não encontrado no inventário de testes.
- `df -h .`: aproximadamente 18 GiB disponíveis; falta histórica de espaço não é um bloqueio atual demonstrado. go/flutter/node/npm disponíveis no PATH; compilação e testes não foram executados.

## Sources and delegated findings

- Leitura direta dos três READMEs, composição Go, serviços de conta/perfil/revogação, porta WireGuard, fake, migração, fontes Flutter/Android, manifests de dependências, runner E2E e documentos do MVP. Sem delegação.
- `skillr resume` do bundle retornou o checkpoint mvp pausado em 2026-09-29. Ranking/data não foram usados como autoridade.
- Checkpoint histórico relata laboratório Docker IPv4/IPv6 e handshake externo Mac. Isso não valida o app, perfis gerados pela API, revogação real, persistência ou migração de produção. Estado atual da VPS e limpeza final do laboratório não foram consultados nesta sessão.

## Uncertainties and decisions to discuss

- Suítes/builds precisam ser revalidados; evidência anterior não foi convertida em resultado atual.
- Recomenda-se fechar B06 (analyze/test/APK) e B05 (E2E efetivamente executado), depois B07 e B08, preservando o escopo local legado.
- A integração real e eventual migração para container são tarefa relacionada com spec própria vinculada ao MVP: alocação única, IPv6, criação/remoção/confirmacão de peers, reconciliação após falhas/reinício, fronteira API/controlador, persistência, backup, firewall e rollback. Não é requisito inventado para fechar o MVP local.
- FQDN do dashboard/API, termos finais, retenção e autoridade operacional continuam sem fechamento verificável.

## Continuation

- Checkpoint: `mvp-checkpoint.yaml`; contrato/plano legados permanecem normativos. Nenhuma nova spec executável ou aprovação foi criada nesta retomada.
- Próximo passo recomendado: retomar a validação local B06 e B05; depois implementar B07 e fechar B08. Tratar integração live/container em decisão separada antes de qualquer ação externa.

## Server infrastructure conversation reconciliation

> Continuação: a proposta de controlador separado abaixo é histórica. O dono escolheu posteriormente controle direto via wgctrl com namespace compartilhado; o desenho atual está em [wireguard-stack/discovery.md](../wireguard-stack/discovery.md).

- Pedido subsequente do dono: focar apenas infraestrutura WireGuard necessária ao backend, com agente lendo `segunda parte pós inicio` (01a0ecd5-31d7-7dc2-a61e-db7a40d3cd9c) e `inicio` (01a0eb04-70d5-7170-80a5-ccbd7e471e7b). Agente folha `contexto_wireguard`, gpt-6-luna/xhigh, somente leitura, retornou a pesquisa. Principal chamou read_thread para ambas e verificou diretamente as últimas decisões.
- Turnos decisivos lidos pelo principal na segunda conversa: 01a0edce-c95c-78c2-b01f-bd41e4552e7d (arquitetura proposta backend → controlador privado → WireGuard container); 01a0edcb-04c5-7d11-bef6-49fd78199fc2 (pedido de análise/scripts, inventário root faltante antes do corte); 01a0edc9-0097-79e3-8bd3-62bf03a38df1 (preservar identidade, três peers e UDP 51442); 01a0edc5-c8eb-7883-b85a-a77680c526bf (handshake externo Mac confirmado). Não há necessidade de repetir esse handshake como descoberta.
- Pesquisa do agente reconciliada com checkpoint local: regras NAT/firewall duplicadas e propriedade/persistência ainda desconhecidas; container definitivo, volumes, reinício automático e controlador privado ainda não entregues. O backend não deve depender de acesso direto ao socket Docker nem à chave privada do servidor.
- Ordem recomendada para este foco: inventário root sanitizado da configuração/regras/peers e verificação de resíduos do laboratório; desenho persistente e fronteira do controlador; backup protegido e scripts de preparação/corte/rollback; validação do desenho definitivo; migração explicitamente autorizada. Preservar identidade do servidor, peers e porta; não copiar ou remover regras duplicadas por suposição.
- Leitura de histórico não comprova estado atual da VPS nem limpeza do laboratório Mac após seu prazo. Nesta pesquisa não houve SSH, scripts operacionais, alterações de backend ou ação de produção.
