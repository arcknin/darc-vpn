# Validação: stack API/WireGuard

Escopo da spec rev. 1 concluído em 2026-10-04: implementação local e prova real em laboratório descartável na VPS. A VPN de produção não foi migrada. Não houve alteração de app/dashboard, DNS, proxy ou dados reais.

## Evidência e revisão

- Backend da máquina local e código compilado no laboratório têm fingerprint de implementação idêntico: `ce50bc9a013421fbf223e0956753544e8b603cdc2396db1df7938cad79c8d141` (30 arquivos Go sem `_test.go`, migrations SQL, go.mod e go.sum; caminho relativo + conteúdo, separados por NUL, ordenados).
- VPS confirmada por SSH: srv1519215/codex, Docker 29.7.2 e Compose 5.4.0, WG kernel presente. Projeto exclusivo `darc-vpn-stack-20261004-01`.
- Bases Docker fixadas por digest; duas imagens Linux com CGO/SQLite construídas. Compose renderizado com `config --quiet` e ambos os serviços ficaram healthy.
- Fonte: [backend](../../../../back-vpn/README.md), [Compose/runbook](../../../../infra/wireguard-stack/README.md), [scripts](../../../../scripts/wireguard-stack/lab.sh).

## Matriz de aceite

| Critério | Prova | Resultado e limite |
| --- | --- | --- |
| CA01 — dois serviços, controle direto | Compose, Docker inspect seletivo, API/kernel reais | Namespace compartilhado; NET_ADMIN, cap_drop ALL, privileged=false; sem Docker socket/host network. Cliente efêmero apenas durante teste. |
| CA02 — rede/interface dual-stack | Bootstrap, preflight não sobreposição, tráfego e rotas do cliente | WG/NAT44/NAT66 e API publicada host-loopback funcionaram. UDP lab 51443, HTTP lab 18081; publicação HTTPS fora do escopo. |
| CA03 — provisionamento confirmado | TestStackHTTPProvisioningRevocationAndRestart e API real | Principal/extra instalados, estado ready observado, chaves e IPs distintos, AllowedIPs de servidor apenas /32 e /128. |
| CA04 — autorização/segredos | Testes HTTP legados/novos e API real | Amigo recusado em criação admin, perfil de outra conta inacessível, segredo bloqueado antes do aceite; logs reais sem credenciais/tokens/chaves privadas de fixture. |
| CA05 — falhas sem sucesso falso | TestKernelProvisionAndRemovalRequireObservedState, TestManagerRecoversWithoutReinstallingRevokedAndRejectsUnknown, HTTP readiness/delivery | Mock que aceita sem aplicar não confirma sucesso; erro/interface indisponível não vira ready nem fake. Consulta kernel sem NET_ADMIN falhou no runtime. Falhas detalhadas são prova simulada local. |
| CA06 — corte real/durável | API block e cliente WG real | Dois peers da conta removidos, operação completed, sessão recusada e ping deixou de passar; peer revogado continuou rejeitado após recriação. |
| CA07 — corrida/exaustão/transação | TestManagedAllocationIsAtomicUniqueAndNeverReusesRevoked, TestStackConcurrentProvisionReconcileAndBlock | Concorrência sob race detector, pool esgotado sem conta parcial, bloqueio serializado com criação/reconciliação e todas as intenções revogadas. |
| CA08 — recuperação/escritor único | TestWriterExclusivity, HTTP segundo escritor, API/WG restart e recriação reais | Peers ativos recompostos; identidade do servidor de teste preservada (cliente com chave pública anterior continuou conectando); revogados não reapareceram. Não testado reboot do host. |
| CA09 — cliente interoperável | Cliente descartável derivado da API | serverPublicKey/endpointPort corretos; handshake, ICMP IPv4/IPv6 e HTTPS v4/v6 HTTP 200 com verificação TLS normal; rotas de egress pelo túnel. Cliente na mesma VPS, não prova externa nova pelo Mac. |
| CA10 — isolamento/cleanup | Preflight e comparação após cleanup | Zero containers/redes/volumes/imagens nomeadas da tarefa, portas livres e staging apagado. Produção ativa, ifindex=3 e endereços iguais; chave/config protegidas de produção não inspecionadas diretamente. Nenhuma operação de corte aplicada. |
| CA11 — contrato/limites | TestStackContractDefinesServerIdentityReadinessAndProvisioning, OpenAPI e runbook | Campos aditivos e erros documentados; QR atual continua pendente de adaptação. Laboratório não apresentado como migração ou app pronto. |

## Testes locais e TDD

- Baseline dos lotes legados passou antes da implementação.
- Red registrado para funções ausentes de alocação/controlador/manager/configuração e campos/Compose ausentes no contrato; implementação tornou os cenários verdes.
- Final: **38 testes Go de primeiro nível** na suíte (inclui subcenários adicionais), `go test -race ./...` e `go vet ./...` aprovados; `gofmt -l cmd internal test migrations` vazio.
- Cache de build dedicado em `/private/tmp/darc-vpn-stack-execution/go-cache`. Testes HTTP loopback executados com permissão de bind após sandbox negar o bind padrão. Nenhum skip foi contado como green.
- Sintaxe shell verificada com `sh -n`; Python compilado/validado, e execução real dos scripts passou. Máquina local não tinha plugin Compose; configuração/build/runtime foram validados no Linux remoto.

## Prova runtime e correções do harness

Resultado completo em [acceptance-results.json](evidence/acceptance-results.json). O harness falhou e foi corrigido antes da execução final aprovada:

1. Docker normalizou NET_ADMIN para CAP_NET_ADMIN; inspeção seletiva confirmou o privilégio real e teste passou a normalizar o nome.
2. Arquivo de perfil copiado ficou ilegível para o cliente sem DAC_OVERRIDE. A primeira sequência shell ignorava falha intermediária; passou a usar set -eu. Perfil passou a ser criado pelo próprio processo via stdin, com umask 077, sem ampliar capabilities ou imprimir o segredo.
3. Polling de readiness passou a tolerar reset de conexão apenas durante o prazo de restart. Tráfego passou a aguardar renegociação por prazo limitado de 75 s por família, sem recriar o cliente para mascarar a recuperação.

Execução final passou do início ao fim: criação/auth/termos → handshake/egress → bloqueio → API restart → WG restart com relink API → recriação conjunta → peer revogado negado → kernel sem capability negado → logs sem segredos. Esses ajustes são do teste/operacionalização; o backend não ganhou fallback ou privilégios externos.

## Cleanup e produção

[cleanup-results.json](evidence/cleanup-results.json) registra verificação independente após `lab.sh down`:

- 0 containers de projeto/cliente, 0 redes, 0 volumes e 0 imagens nomeadas da tarefa; manifests anteriores identificados da própria tarefa também removidos quando presentes.
- UDP 51443 e TCP 18081 livres. Diretório remoto exclusivo removido após conferir ownership; fixtures/arquivos de upload eliminados. Credenciais temporárias locais eliminadas.
- wg-quick@wg0 continuou active, mesma interface ifindex 3 e endereços 10.66.66.1/24 e fd42:42:42::1/64. Nenhuma inspeção da chave protegida, alteração de peer real ou teste de tráfego de clientes reais.
- Bases e cache de build compartilhados foram preservados; não se executou prune global.

## Continuidade

Para produção: nova decisão/spec operacional com inventário root sanitizado, identidade/peers existentes, pools compatíveis, termos finais, API HTTPS/hosts, persistência, backup e rollback. O laboratório foi removido e não deve ser tratado como serviço implantado.

Para app: consumir serverPublicKey e endpointPort no QR/configuração nativa; continuar lotes MVP e suas validações separadamente. Esta entrega deixa o servidor controlável comprovado, não o aplicativo finalizado.
