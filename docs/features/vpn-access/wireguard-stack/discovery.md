# Discovery: stack API e WireGuard com controle direto

## Request and investigation boundary

- Em 2026-10-04, o dono escolheu uma stack Compose com API Go e WireGuard em dois containers, compartilhamento do namespace de rede via `network_mode: service:wireguard` e controle direto pelo backend com `wgctrl`. Pediu organizar a documentação conforme spec-drive e tratar isso como nova discovery.
- Foco atual: servidor e infraestrutura necessários à integração. Alterações do backend indispensáveis serão delimitadas na spec; app Flutter e dashboard não fazem parte desta discovery.
- Autoridade inicial: pesquisa e documentação local. O pedido posterior de validação na VPS está registrado na seção de acesso abaixo; implementação, operações concretas de instalação/laboratório e migração serão delimitadas na spec e nas autorizações correspondentes.
- Workspace: `/Users/arcknin/Desktop/darc/softwares/darc-vpn`; projetos sem Git no inventário desta sessão. Não inicializar Git nem substituir fontes existentes para adaptar o workflow.

## Verified current behavior

- `back-vpn/cmd/api/main.go`: usa `wireguard.NewFake(FakeConfirmAll)` na composição padrão; geração de perfis não instala peers reais.
- `back-vpn/internal/domain/ports.go`: WireGuardController define apenas RemovePeers/ConfirmPeersRemoved. Controle de provisionamento ainda precisa ser delimitado.
- `back-vpn/go.mod`: `golang.zx2c4.com/wireguard/wgctrl` já é dependência. Isso não comprova existência do adaptador live.
- Não foram encontrados Dockerfile/Compose no inventário local. Não há stack definitiva entregue nas fontes consultadas.
- Fonte oficial wgctrl: controla dispositivos WireGuard existentes, inclusive via generic netlink no Linux; criação da interface e atribuição de endereços não são funções dessa biblioteca.
- Fonte oficial Compose: `network_mode: service:wireguard` permite compartilhar o namespace de rede do serviço. Uma bridge comum não fornece por si só acesso à interface WireGuard do outro container.
- MVP histórico: wg0 no host, UDP 51442, três peers, endereços 10.66.66.1/24 e fd42:42:42::1/64. Laboratório container dual-stack e handshake externo Mac comprovados em setembro; migração definitiva e estado atual não verificados nesta sessão.
- Regras NAT/firewall duplicadas e sua propriedade/persistência, configuração completa sanitizada dos peers e limpeza final do laboratório Mac permanecem sem verificação atual.

## Sources and delegated findings

- [Discovery do MVP](../mvp/discovery.md), [checkpoint](../mvp/mvp-checkpoint.yaml), [contrato rev. 2](../mvp/mvp-contract.md), [plano rev. 3](../mvp/mvp-plan.md), [histórico](../mvp/mvp-history.md), [minuta dos termos](../mvp/terms-of-use-draft.md).
- Fontes Go acima diretamente consultadas; inventário `rg --files -g '*compose*' -g '*Dockerfile*'` sem resultados.
- [WireGuard/wgctrl-go](https://github.com/WireGuard/wgctrl-go), README oficial consultado em 2026-10-04.
- [Docker Compose services](https://docs.docker.com/reference/compose-file/services/#network_mode), documentação oficial consultada em 2026-10-04.
- Conversas `segunda parte pós inicio` (01a0ecd5-31d7-7dc2-a61e-db7a40d3cd9c) e `inicio` (01a0eb04-70d5-7170-80a5-ccbd7e471e7b) lidas nesta sessão. Agente contexto_wireguard reconciliou o histórico; principal verificou as decisões finais. Não houve nova delegação para esta organização.

## Uncertainties and decisions to discuss

### Decisão atual do dono

- Uma stack, dois containers: API e WireGuard; controlador como módulo do backend, sem terceiro serviço de controle, sem socket Docker e sem docker exec como mecanismo de integração.
- Caminho escolhido para investigar: namespace de rede compartilhado e wgctrl. A escolha não equivale a validação prática nem aprovação de uma spec executável.
- A proposta anterior de controlador privado separado fica como histórico, substituída por esta decisão para o desenho novo.

### Verificações técnicas pendentes

- Confirmar implementação kernel WireGuard e permissões mínimas para wgctrl no namespace compartilhado. NET_ADMIN não é uma permissão limitada apenas a peers WireGuard; o alcance sobre a rede compartilhada deve estar explícito. Não prometer inacessibilidade da chave privada do dispositivo sem verificar o acesso efetivo oferecido pela biblioteca/kernel.
- Definir imagem e bootstrap responsáveis por criar wg0, endereços, rotas, encaminhamento e NAT; wgctrl configura o dispositivo existente, não substitui essas etapas.
- Definir como a API será acessada por proxy/HTTPS no namespace compartilhado; publicação de portas e redes deve ser compatível com network_mode, sem presumir FQDN da API.
- Definir fonte de verdade persistente, único escritor e reconciliação após reinício/falha. Recriação do container WireGuard e da API deve ser testada para confirmar vínculo ao namespace e recuperação dos peers, sem peers revogados voltarem.
- Definir alocação sem colisão, pools e IPv6: gerador atual usa IPv4 derivado de hash sem unicidade e anuncia ::/0 sem IPv6 do cliente. A migração também deve preservar peers existentes sem conflitar com novos perfis.
- Delimitar mudanças necessárias de provisionamento/remoção/consulta no backend e prova de remoção ativa/persistente; fake não satisfaz essa prova.
- Completar inventário root sanitizado, backup protegido, posse das regras firewall e procedimento de corte/rollback antes de elaborar migração executável.
- TDD nesta tarefa: yes, confirmado explicitamente pelo dono durante a elaboração da spec, sem transferir aprovação do MVP.

## Continuation

- Checkpoint: `wireguard-stack-checkpoint.yaml`; [spec integrada rev. 1](spec-01-stack.md) criada como draft.
- Próximo passo: discutir/aprovar spec; bootstrap, permissões, persistência, exposição e reinícios têm desenho proposto e prova runtime pendente, sem serem tratados como fatos já validados.
- Teste descartável para comprovar o desenho será delimitado na spec conforme pedido de validação posterior abaixo. O handshake Mac anterior não precisa ser repetido para redescobrir a viabilidade de container WireGuard.

## Acesso à VPS e validação posterior

- Decisão posterior do dono em 2026-10-04: registrar o acesso para o assistente testar na VPS após preparar o fluxo. Isso autoriza planejar a validação remota; operações do laboratório serão explicitadas na spec antes da execução. Não autoriza corte de produção.
- Alias existente confirmado por leitura seletiva de `~/.ssh/config`: `darc-vpn`, usuário `codex`, chave local `~/.ssh/darc_vpn_codex`, `IdentitiesOnly yes`. O destino é resolvido pelo alias; não copiar chave privada para documentos ou containers.
- Conexão: `ssh -o BatchMode=yes darc-vpn`. Preflight de identidade, quando a etapa remota começar: `ssh -o BatchMode=yes darc-vpn 'hostname; id -un'`. A configuração local foi verificada; conectividade, identidade remota e privilégios atuais ainda não foram retestados.
- Histórico de acesso: codex conseguia usar Docker, mas não ler a configuração protegida do WireGuard do host; acesso root necessário será tratado por comandos sanitizados para o dono ou autorização explícita, sem assumir sudo disponível.
- Validação prevista: recursos descartáveis identificados da nova stack, porta alternativa livre, chaves/peers de teste e armazenamento isolado; comprovar wgctrl, acesso à API, persistência, revogação e recriação dos containers. Definir limpeza e evidência na spec.
- Preservar `wg-quick@wg0`, UDP 51442, peers/configuração reais e regras existentes durante o laboratório. Migração definitiva, alterações de firewall do host e dados reais precisam de autoridade própria.

## Achados na elaboração da spec

- `internal/profiles/factory.go` grava a chave pública do cliente em peer_public_key. `front-vpn/lib/src/qr_config.dart` coloca esse campo em `[Peer] PublicKey` e fixa 51820; serverPublicKey/endpointPort ainda não existem no contrato atual. A nova spec propõe campos aditivos e cliente descartável correto; app fora do escopo.
- `cmd/api/main.go` exige bind loopback. Publicação Docker requer modo container explícito para bind adequado, com publish host-loopback no lab; não relaxar o padrão local silenciosamente.

## Encerramento da execução aprovada

A discovery acima registra o ponto de partida e as decisões. A spec rev. 1 foi aprovada e executada nesta conversa. Implementação, testes locais e laboratório real passaram; [validation.md](validation.md) e o checkpoint substituem as limitações iniciais de ausência de código/runtime/SSH. O laboratório foi removido, produção não foi migrada e o app continua fora deste escopo.
