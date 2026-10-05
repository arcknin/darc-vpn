---
schema_version: 1
managed_by: spec-drive
feature: vpn-access
task: wireguard-stack
revision: 1
status: approved
---

# Spec integrada: stack API/WireGuard com controle direto

## Contexto e fontes

Objetivo: entregar uma stack Compose com dois serviços permanentes, API Go e WireGuard, em que o backend administra peers diretamente via wgctrl no namespace compartilhado. Validar o ciclo real em laboratório descartável na VPS; não migrar o wg0 de produção nesta entrega.

- [Discovery](discovery.md) contém fontes, decisões e acesso SSH. [MVP aprovado](../mvp/mvp-contract.md) e [plano legado](../mvp/mvp-plan.md) continuam válidos no seu escopo local.
- Raiz: `/Users/arcknin/Desktop/darc/softwares/darc-vpn`; fontes relevantes em `back-vpn/`. Sem Git no inventário; preservar arquivos existentes e não inicializar repositório nesta tarefa.
- `cmd/api/main.go`: controlador fake, gerador IPv4 por hash, somente bind HTTP loopback. `internal/domain/ports.go`: remoção/consulta, sem provisionamento. `internal/storage/store.go`: persistência SQLite e bloqueio transacional. `internal/profiles/factory.go`: peer_public_key é a chave pública do cliente.
- `front-vpn/lib/src/qr_config.dart` usa esse peer_public_key em `[Peer] PublicKey` e porta fixa 51820: o QR atual não demonstra interoperabilidade com o servidor. A correção do app fica em tarefa posterior; esta entrega adiciona metadados corretos à API e usa cliente de teste independente.
- [wgctrl](https://github.com/WireGuard/wgctrl-go) configura dispositivos existentes; bootstrap da interface e endereços fica no serviço WireGuard. [Compose](https://docs.docker.com/reference/compose-file/services/#network_mode) fornece namespace compartilhado; [protocolo kernel](https://docs.kernel.org/netlink/specs/wireguard.html) fornece configuração/consulta do dispositivo.
- Estado histórico da VPS: WG host em UDP 51442, três peers. Container de laboratório e handshake Mac comprovados anteriormente; stack nova, identidade/privilégios atuais, firewall e persistência ainda não validados.

## Objetivos

1. Inicializar wg0 em container e permitir à API consultar, instalar e remover peers sem Docker socket, docker exec ou terceiro controlador.
2. Vincular criação administrativa de conta/perfil e bloqueio ao estado WireGuard real, com persistência e recuperação após falha/reinício.
3. Impedir colisões de IP e separar chave do cliente, chave do servidor e porta do endpoint.
4. Demonstrar API, tráfego IPv4/IPv6, revogação e recuperação na VPS em ambiente isolado; entregar instruções reproduzíveis e evidência sanitizada.

## Não objetivos

- Migrar/parar o wg0 do host, importar seus peers/chaves ou usar UDP 51442.
- Alterar DNS, Nginx existente, publicar API/dashboard, configurar domínio/certificado definitivo ou distribuir APK.
- Implementar Flutter/QR/ponte Android ou dashboard; adicionar terceiro serviço permanente de controle.
- Manipular contas, SQLite, termos, chaves ou peers reais. Limpar regras existentes de firewall, conceder sudo, instalar pacotes no host ou carregar módulos automaticamente.
- Alta disponibilidade/múltiplas réplicas, rotação de chave mestra ou política final de observabilidade VPN/termos. Não declarar o produto pronto a partir do laboratório.

## Desenho e execução

### Stack e privilégios

- `wireguard` cria e mantém o namespace, a interface kernel wg0, endereço do servidor, rotas e encaminhamento IPv4/IPv6. Imagem própria com ferramentas de rede mínimas, base/versionamento fixados na implementação; não usar latest nem um gerenciador que gere peers em paralelo.
- `api` usa `network_mode: service:wireguard`, sem networks/ports próprios, e acessa wg0 com wgctrl. Ambos sem privileged, Docker socket, namespace de rede do host ou montagem de `/etc/wireguard` do host.
- Adicionar NET_ADMIN no namespace compartilhado; API não recebe NET_RAW nem SYS_ADMIN/SYS_MODULE. Capacidades adicionais do bootstrap só com necessidade demonstrada e registrada. Kernel WG deve existir; preflight falha com instrução concreta se não existir, sem alterar módulos do host.
- NET_ADMIN administra a rede compartilhada, não apenas peers. A API pode ter acesso à chave do dispositivo por consulta administrativa: não prometer isolamento dessa chave. Não serializar/logar respostas brutas wgctrl.
- Um processo API, um escritor. Lock exclusivo no volume SQLite impede duas instâncias de reconciliação; falha ao adquirir lock impede readiness.
- Expor UDP apenas no serviço wireguard. No laboratório usar porta livre alternativa, preferencialmente 51443 após verificar resíduos; nunca assumir que esteja livre. Publicar API TCP somente em loopback do host numa porta livre; acesso via túnel SSH.
- Para publicação Docker funcionar, API escuta `0.0.0.0:8080` em modo explícito `container-lab`. O Compose restringe o publish a `127.0.0.1`. Modo local mantém restrição loopback atual; não habilitar bind amplo implicitamente. Credenciais fictícias só via loopback/SSH; publicação HTTPS definitiva fica fora desta spec.
- Rede Compose dual-stack própria, sub-redes não sobrepostas verificadas no preflight. Bootstrap NAT44/NAT66 e forwarding dentro do namespace; Docker gerencia apenas regras normais dos novos recursos no host. Não reproduzir hooks do host nem editar/apagar suas regras manualmente.

### Bootstrap, dados e autoridade de peers

- Chave privada do servidor de laboratório gerada uma vez em diretório/volume protegido, fora da fonte. WireGuard lê somente essa identidade e parâmetros de interface; API não precisa receber arquivo da chave. A persistência desse arquivo preserva identidade após recriação.
- Bootstrap começa sem peers; não salva/recarrega peers via wg-quick SaveConfig. SQLite é a fonte de verdade dos peers gerenciados. API recompõe apenas contas ativas; revogações persistidas nunca reaparecem.
- SQLite, comprovantes de aceite e chave mestra ficam persistentes e separados da imagem; proprietário/permissões e chave mestra estável são validados. Termos fictícios completos e chaves de laboratório não saem em logs/relatório.
- Lab usa pools configuráveis, inicialmente `10.77.77.0/24` e `fd42:77:77::/64`, sujeitos ao preflight. Servidor usa .1/::1; clientes recebem /32 e /128 únicos, excluindo rede, broadcast IPv4 e endereço do servidor.
- Reservas de IPv4/IPv6 são transacionais e únicas no SQLite. Nesta revisão, não reutilizar endereços de perfis revogados; exaustão retorna erro explícito sem salvar conta/perfil parcial. Testar concorrência.
- Migração aditiva de banco local de teste: detectar duplicatas/incompatibilidades antigas e parar sem corrigir silenciosamente. Aplicar novas restrições apenas após validação. Nunca executar contra banco real nesta entrega.
- Server AllowedIPs de cada peer são apenas os IPs atribuídos /32 e /128; rotas `0.0.0.0/0` e `::/0` pertencem à configuração do cliente. O servidor não assume rota default pelo peer.

### Integração do backend, contrato e falhas

- Adicionar adaptador wgctrl ao serviço existente e modo explícito `fake`/`kernel`. Fake só permitido em teste/dev configurado; modo kernel nunca recai para fake em caso de falha.
- Criação de conta/principal e perfil extra grava perfil e intenção durável `pending` na mesma transação; tenta provisionar após commit. Sucesso gerenciado exige consultar e confirmar chave pública e AllowedIPs ativos.
- Intenção por perfil: `pending`, `ready`, `failed`, `revoked`; erro sanitizado, revisão e data. Falha externa não perde perfil/reserva nem gera sucesso de provisionamento. Recuperação privada de perfil não-ready falha com código estável `profile_not_ready`; nenhuma chave é entregue antes do aceite existente.
- Reconciliador único inicia com snapshot SQLite e sincroniza antes de readiness. Depois tenta pendências em intervalos configuráveis com backoff limitado (proposta: início 5 s, máximo 60 s). Mudanças de conta/perfil e aplicação WG compartilham serialização: provisionamento pendente não pode reinstalar um peer depois do bloqueio.
- Bloqueio mantém o commit de conta/sessões antes da remoção. Persiste estado desejado revogado, retira peers e consulta ausência real. `completed` exige ausência ativa de todos e exclusão das intenções de recriação; erro/parcial fica failed/pending e é retentado. Reinício não depende de um arquivo de peers antigo para confirmar persistência.
- Se API cair, peers já instalados podem permanecer até recuperação; não prometer revogação durante indisponibilidade total. WG recriado inicia vazio e aguarda reconciliação. Readiness falha enquanto interface inacessível ou sincronização inicial incompleta; liveness apenas indica processo vivo. Criar endpoint de readiness separado de `/healthz`.
- No namespace dedicado, detectar peers inesperados e falhar readiness com erro sanitizado, sem apagá-los automaticamente. Não atuar em interfaces fora do nome configurado validado; não tocar wg0 do host.
- Contrato API: conservar `peerPublicKey` como identidade pública do cliente; adicionar `serverPublicKey` e `endpointPort` ao DTO privado e metadados de provisionamento ao DTO administrativo sem segredo. `address` comporta endereços IPv4/IPv6 em formato WireGuard; campos antigos não mudam de significado.
- Endpoint hostname permanece `vpn.darcanix.com`; laboratório gera cliente de teste substituindo apenas destino de transporte pelo endereço efetivo da VPS e porta alternativa, sem DNS. ServerPublicKey vem da interface; a PrivateKey entregue é a do cliente.
- OpenAPI, erros, testes contratuais e fixtures devem refletir extensões. Frontends antigos não ficam automaticamente corrigidos; registrar incompatibilidade operacional do QR atual e o ponto de retomada do cliente.

### Reinício e rollback do laboratório

- Reiniciar API: adquirir lock, confirmar interface e reconciliar antes de readiness. Reiniciar WG: detectar perda de interface, suspender readiness e reconectar wgctrl quando disponível.
- Recriar WG pode trocar o namespace: procedimento documentado deve recriar/revincular API dependente, preservando volumes. Não prometer que restart policy resolve sozinho essa dependência.
- Repetir criação/reconciliação não duplica peers nem troca chaves/IPs. Recriação conjunta mantém identidade e peers ativos, nunca os revogados.
- Rollback nesta entrega é parar/remover somente projeto de laboratório e restaurar seus dados de teste quando necessário. Não produzir script de corte real sem inventário atualizado e autorização específica.

### Passos ordenados da entrega única

1. Congelar configuração, extensões OpenAPI, modelos de intenção/alocação e cenários TDD; manter testes fake legados.
2. Implementar reservas, intenção durável, provisionamento/revogação/reconciliação e adaptador wgctrl com testes de falha/concorrência.
3. Construir imagens/Compose/bootstrap/healthchecks e scripts de laboratório; validar configuração renderizada sem expor segredos.
4. Executar checks Go e integrações locais possíveis; registrar quando kernel WireGuard requer Linux, sem simular prova kernel no Mac.
5. Fazer preflight remoto sanitizado via SSH; executar laboratório isolado conforme autoridade abaixo, testar API e cliente real descartável, reinícios/recriação/falhas e revogação.
6. Limpar recursos identificados, verificar produção preservada e registrar matriz de evidências. Deixar migração e atualização do app explicitamente pendentes.

### Arquivos e propriedade

| Raiz | Escopo | Ação | Responsável |
| --- | --- | --- | --- |
| back-vpn | cmd/api, internal/domain/accounts/profiles/storage/revocation/wireguard/http, migrations | Adaptador real, persistência, contrato e configuração | Principal |
| back-vpn | api/openapi.yaml, test, cmd/api/*test.go | TDD e integração contratual/concorrente | Principal |
| workspace | infra/wireguard-stack/ | Dockerfiles, Compose, bootstrap, exemplos sem segredo e runbook | Principal |
| workspace | scripts/wireguard-stack/ | Preflight, laboratório, evidência e cleanup delimitados | Principal |
| docs | diretório desta tarefa e índices | Decisões, prova e checkpoint | Principal |

## Dependências e autoridade

- Requer Docker/Compose e WG kernel na VPS, portas/sub-redes livres, acesso `ssh -o BatchMode=yes darc-vpn` como codex. Chave SSH existente permanece local.
- Histórico mostra acesso Docker; confirmar identidade, privilégios e recursos no preflight. Não contornar falta de sudo/acesso root usando container para ler arquivos protegidos do host.
- Aprovar esta spec autoriza implementação e testes locais no escopo acima. Operações remotas abaixo são proposta explícita para a validação posterior solicitada pelo dono, a confirmar na aprovação da spec.
- Laboratório remoto: upload exclusivo para diretório de tarefa novo, build/pull de imagens fixadas, criação de projeto/rede/volumes identificados, porta alternativa, chaves/dados fictícios, testes e remoção exclusiva desses recursos. Container cliente de teste é temporário, não terceiro serviço permanente.
- Nenhuma autoridade de produção: sem migração, reboot VPS, pacote no host, firewall manual, DNS/proxy, peers/chaves reais, banco real ou limpeza de recursos preexistentes. Se preflight exigir isso, registrar limitação e voltar ao dono.
- Tarefa do cliente depende das extensões finalizadas desta entrega; não condiciona fechamento da evidência servidor/WG. A migração depende desta prova mais inventário root sanitizado e spec/autorização operacional própria.

## Critérios de aceite

- [x] CA01: Compose contém dois serviços permanentes; API compartilha namespace e administra WG com wgctrl sem Docker socket/exec/privileged/host network.
- [x] CA02: bootstrap cria interface/endereço/NAT dual-stack, sem capturar rota default pelo peer; API acessível apenas pelo caminho de lab definido.
- [x] CA03: criação administrativa gera chaves distintas e IPs únicos, instala peer com AllowedIPs corretos e só marca ready após consulta real.
- [x] CA04: usuário comum não provisiona perfil arbitrário; segredo só ao proprietário após aceite; saída administrativa/logs não contêm chaves privadas, PSKs ou tokens.
- [x] CA05: falta de interface/permissão e falhas de aplicação/consulta não viram fake ou ready/completed; retry converge sem duplicação.
- [x] CA06: bloquear conta corta sessões imediatamente e elimina todos seus peers ativos/desejados; tráfego de teste deixa de passar após confirmação e peers não voltam depois de reiniciar/recriar.
- [x] CA07: corrida criação/retry/bloqueio, pool cheio e falha de banco não reinstalam revogados nem deixam reservas/perfis inconsistentes.
- [x] CA08: reinício API, WG e recriação conjunta preservam identidade/reservas/contas ativas e recompõem peers; segunda instância escritora é recusada.
- [x] CA09: cliente WG descartável derivado da API usa chave pública do servidor e porta corretas; handshake, tráfego de túnel v4/v6 e saída HTTPS v4/v6 verificados com TLS normal.
- [x] CA10: laboratório usa novos recursos; identidade/endereço/estado do wg0 de produção e porta 51442 preservados. Cleanup deixa zero recursos da tarefa, porta alternativa livre e comprovante sanitizado.
- [x] CA11: OpenAPI/tests documentam novos campos; runbook registra limite do cliente existente e nenhum resultado de laboratório é chamado de migração ou produto pronto.

## Validação

- Fonte/configuração: `gofmt -l`, `go vet ./...`, `docker compose config --quiet`, inspeção de capabilities/mounts/port bindings e lint shell conforme ferramentas disponíveis.
- Testes Go: `go test -race ./...`; novas migrações só em SQLite descartável. Cobrir transações, exaustão, exclusividade, concorrência, falhas antes/depois do efeito externo e recuperações.
- Build: binário/imagem Linux com suporte CGO/SQLite, imagens fixadas e compilação completa; não instalar dependências globais no Mac ou host sem necessidade/autoridade.
- Runtime remoto: preflight antes/depois; teste black-box HTTP via SSH e cliente WG efêmero independente, confirmando peers com leitura kernel. Teste v4/v6 de tunnel e egress é adicional específico desta stack integrada, não repetição da descoberta do Mac.
- Reinícios: stop/start API e WG, recriação conjunta com volumes preservados, falha WG, retomada e tráfego negado de peer revogado. Reboot real da VPS fora do escopo; documentar que teste de containers não prova boot do host.
- Evidência: CA → teste/comando → resultado → ambiente/revisão local. Sanitizar sem esconder erros. Não exportar configs privadas, respostas completas wgctrl, ambientes, tokens ou senhas.
- Limite: fake prova regras; laboratório prova esta integração em recursos descartáveis. Host real, peers antigos, proxy/HTTPS publicado, app e migração permanecem sem validação nesta entrega.

## Testes e TDD

- Decisão do dono nesta tarefa: **yes**, manter TDD.
- Ordem: red mínimo observável → implementação → green → refactor. Para infraestrutura, cenários verificáveis do harness devem detectar configuração/efeitos ausentes; não provocar falha destrutiva para registrar red.
- Testes de segurança/contrato e concorrência acompanham mudanças; runtime real verifica interface/revogações que mocks não provam. Não criar gate/revisor automático.

## Automação e scripts

- Scripts justificados por preparação dependente, repetibilidade, prevenção de cleanup incorreto e captura de evidência. Escopo único por identificador de projeto; recusar diretório existente sem manifesto de posse.
- Inputs: alias SSH, ID tarefa, portas/pools verificados e caminhos de dados da própria tarefa. Nunca receber comandos shell arbitrários nem imprimir segredos. Credenciais/chaves geradas em arquivos restritos.
- Preflight só lê; lab cria exclusivamente recursos manifestados; cleanup checa labels/nomes/caminhos e remove somente esses recursos. Nunca `docker prune`, remoção ampla, cleanup de lab antigo ou flush firewall.
- Exit 0 exige checks realizados; dependência ausente/skip retorna resultado não aprovado. Falha captura diagnóstico sanitizado e faz cleanup seguro; se cleanup falhar, registrar resíduos e comando exato de retomada, sem declarar encerramento.

## Riscos e decisões

- NET_ADMIN e consulta WG têm alcance maior que um endpoint de add/remove: compromisso do processo pode afetar o namespace. Aceitação desse alcance faz parte da aprovação do desenho, sem Docker/root do host como solução de atalho.
- Dependência do namespace em recriação e NAT66 no ambiente específico precisam de teste. Se funcionar apenas via host network, socket Docker ou terceiro controlador, isso muda o desenho e exige retorno ao dono.
- Provisionamento persistente/IPv6 são mudanças backend indispensáveis ao efeito real. Não ampliar para telas/apps, observabilidade completa ou migração automática.
- Imagem/pools/portas são parâmetros técnicos, escolhidos com preflight; mudança do comportamento dual-stack, fonte de verdade, autorização, contrato ou escopo de ambiente exige decisão material.
- Termos publicados, endpoint real e API HTTPS não estão fechados; laboratorio usa fixtures. Não converter fixtures em configuração de publicação.

## Aprovações

- Caminho de arquitetura: escolhido pelo dono em 2026-10-04.
- TDD: yes, confirmado nesta conversa.
- Spec rev. 1: **approved** pelo dono nesta conversa em 2026-10-04; executar o escopo descrito.
- Implementação local: autorizada pela aprovação.
- Laboratório remoto conforme seção de autoridade: autorizado pela aprovação explícita de execução desta spec; recursos descartáveis apenas.
- Produção/migração/dados reais: fora do escopo e sem autorização nesta entrega.

## Execução e retomada

- [Checkpoint](wireguard-stack-checkpoint.yaml): fonte compacta de andamento.
- Stage: completed; implementação e laboratório aprovados concluídos. [Matriz de evidência](validation.md) registra provas e limitações.
- Retomada: escopo encerrado, laboratório removido. Migração real e frontend precisam de continuidade própria; ver checkpoint e validation.md.
