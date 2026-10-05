---
schema_version: 1
managed_by: spec-drive
feature: vpn-access
task: production-target
revision: 1
status: authorized_02b_implementation_subset
---

# Spec integrada: preservar peers antigos e preparar alvo de produção

## Contexto, objetivo e fontes

Entregar código e configuração que permitam migrar o WireGuard existente sem trocar identidade, endereços, porta, peers ou PSKs; operador root fará importação e corte. [Discovery](discovery.md), [stack validada](../wireguard-stack/validation.md), [inventário root fornecido](../migration-scripts/owner-inventory.md) e [scripts entregues](../migration-scripts/validation.md).

Não basta copiar wg0.conf: API atual inicia uma fonte de verdade vazia e não restaura PSKs. A assinatura operacional é bdf982e05ee6c2d14a53a60bc5302abe07bf1ccea1f53c1d99f01a71e6640967; deve ser revalidada antes do corte, não tratada como estado eterno.

## Escopo e exclusões

Inclui backend Go necessário à importação/persistência legada, modelo de produção Compose/bootstrap, comandos offline e provas reproduzíveis de restauração. Exclui app/dashboard, vincular antigos peers a contas, publicar API/DNS/HTTPS, limpar firewall global e executar migração real pelo assistente.

## Comportamento proposto

### Preservação e fonte de verdade

- Criar registro legado separado das contas/perfis: chave pública, AllowedIPs completos, keepalive, PSK cifrada com AES-GCM e metadados de importação. Não criar senha/PrivateKey de cliente fictícia nem perfil recuperável pelo app.
- Antigos peers continuam usando suas configurações e chaves. Ficam externos ao app nesta entrega; não se inventa titularidade. Gestão/vinculação posterior exige decisão própria.
- Importador offline lê manifesto protegido produzido pelo operador root a partir do backup. Não envia segredos ao chat nem a codex. Identidade privada do servidor fica no armazenamento WG; manifesto de peers entregue ao backend não contém essa identidade privada.
- Importação transacional e idempotente: validar todas as entradas antes de commit; mesma importação não duplica; conflito de identidade/AllowedIPs/PSK não é sobrescrito silenciosamente. Adquirir o mesmo lock de escritor da API e executar com ela parada.
- Reserva de novos IPs exclui todos os prefixos legados, inclusive peers somente IPv4/IPv6 ou rotas maiores que /32 e /128. Colisão/incompatibilidade entre rotas exige revisão antes da importação; sem renumerar antigos automaticamente.
- Reconciliador conhece legados e gerenciados, recompõe PSK explicitamente e confirma estado ativo correspondente. Falha de decifragem/permissão/interface deixa readiness falsa, sem recriar chave/PSK ou cair para fake.
- Bloquear conta nova só remove seus próprios peers; não apaga legados externos. Peers desconhecidos fora das duas fontes continuam sendo erro, sem exclusão automática.
- Chave mestra persistente obrigatória: se existe banco cifrado e falta arquivo de chave, startup falha. Só gerar segredo numa inicialização comprovadamente vazia.

### Stack final e rede

- Dois serviços API/WG no namespace compartilhado, NET_ADMIN apenas, sem socket Docker/privileged/host network. Imagens já construídas fixadas por ID para o corte.
- WG usa identidade fornecida pelo operador, endereços originais e UDP 51442. Não gerar nova identidade implicitamente no modo produção. Bootstrap pode começar vazio, desde que API restaure toda a fonte durável antes de readiness.
- NAT44/NAT66 e encaminhamento no namespace limitados aos pools VPN; não copiar PostUp/PostDown amplos nem remover duplicatas do host por suposição. Bridge definida após checagem de conflitos.
- API TCP publicada em loopback do host, acessada por SSH. Credenciais, termos reais e retenção ficam configuráveis por arquivos root-only. Não usar termos/contas de fixture ou alegar API pública pronta.
- Banco, chave mestra, identity WG e comprovantes persistem separados da imagem. Documentar backup consistente com API parada e recuperação por volumes, sem reboot da VPS nesta entrega.

### Prova e operação pelo dono

- Implementação e teste do assistente usam três peers sintéticos com PSKs, sem ler os arquivos root reais.
- Gerar runbook/comandos para root exportar/importar o manifesto, materializar identidade, montar deployment e comparar baseline. Não pedir PrivateKeys dos clientes.
- Restauração fria com dados reais é executada pelo dono em namespace isolado, sem publicar porta alternativa do servidor real nem encaminhar clientes antigos a outra instância simultânea. Conferir estado kernel/PSKs, cold restore e reconciliação; dados sensíveis permanecem root-only.
- restore-proof.json só após testes reais e revisão de rede/termos, ligado à assinatura atual e ao modelo final; sem fabricar flags.
- Depois root prepara o plano e usa o script de corte/rollback existente. Após corte verifica um cliente externo antigo; assistente não executa essas operações pela aprovação deste documento.

## Passos de execução e arquivos

1. Testes de importação/erro/PSK/reserva antes do código; migração aditiva SQLite e importador offline com lock/cifragem.
2. Estender reconciliador/adaptador para restaurar legados/PSKs e testes de coexistência com bloqueio de contas novas.
3. Criar infra/wireguard-production e scripts de export/import/validação; adaptar entrypoint sem quebrar laboratório.
4. Rodar suíte Go race/vet/build, validação Compose e testes sintéticos de restauração Linux.
5. Entregar pacote, hashes e comandos root; marcar real-data proof/plan/cutover como pendentes do dono, com exata retomada.

Escopo local: back-vpn/{cmd,internal,migrations,api,test}, infra/wireguard-production, alterações mínimas compatíveis em infra/wireguard-stack e scripts/wireguard-migration quando necessárias, docs desta tarefa. Principal responsável; sem delegação automática ou reviewer obrigatório.

## Aceitação e validação

- [ ] Três legados sintéticos com PSK importados atomicamente; repetição idempotente e conflitos recusados sem commit parcial.
- [ ] Banco/manifests/logs administrativos não expõem PSKs/client secrets; corrupção/chave errada falha fechada.
- [ ] Novos perfis não recebem IPs/prefixos reservados; legados só de uma família preservados sem impor endereço inventado.
- [ ] Reconciliador restaura PSKs/AllowedIPs/keepalive após recriação e rejeita desconhecidos; bloqueio de conta nova não interfere em antigos.
- [ ] Chave/porta/endereço de servidor sintético permanecem iguais; tráfego autenticado com PSK correto funciona e PSK errado é recusado.
- [ ] Fonte protegida/configuração de produção não contém fixtures; Compose guarda isolamento e exposição loopback.
- [ ] Pacote root inclui prova reproduzível de cold restore sem tocar wg0 do host, além de instruções de importação e rollback.
- [ ] Todos os testes relevantes passam; dados/produção reais continuam sem execução pelo assistente.

Comandos: go test -race ./..., go vet ./..., gofmt -l, shellcheck, validação Python/shell, docker compose config --quiet, laboratório Linux descartável com chaves novas. Se testes Linux na VPS forem desejados, limitar autoridade explicitamente antes da execução; aprovação local não autoriza uso de dados reais.

## TDD, automação e autoridade

TDD proposto: yes, continuidade da preferência confirmada no trabalho de servidor; confirmar na aprovação desta nova tarefa. Testes obrigatórios independentes da escolha. Scripts para importação/prova se justificam por segredo, consistência transacional e repetibilidade; não avaliar configuração como shell, não fazer prune/flush, não instalar pacotes no host automaticamente.

Este draft não autoriza implementação nova. Aprovação autorizará código/testes locais e entrega de pacote. Operador root importa dados reais e executa verificação/corte; sem autoridade implícita de agente para root, banco real, publicação ou produção.

## Decisões e riscos

Decisão principal para aprovação: manter peers antigos como legados externos, sem contas/QR recuperável no app. Se dono exigir associação imediata a usuários, redefinir dados/fluxo antes de implementar. Definir os termos finais antes da preparação operacional, sem bloquear testes sintéticos do código. Mudanças que exijam renumerar peers, obter PrivateKeys dos clientes, alterar firewall global ou terceiro controlador exigem retorno ao dono.

## Aprovação e retomada

Spec rev. 1: pending. Implementação: pending. Real-data/produção: operador root em etapa separada. [Checkpoint](production-target-checkpoint.yaml). Próximo passo: dono avaliar/aprovar esta proposta; depois executar os cinco passos, enquanto a VPN atual segue no host.

## Autorização posterior e entrega 02b

Dono pediu explicitamente criar 02b para executar a preparação faltante. Esse pedido autoriza suporte técnico necessário ao script (registro legado/PSK cifrado, importador/reconciliador, templates e pacote) e testes locais. Não autoriza root pelo assistente nem execução real de importação/corte. A implementação desta parte está entregue; provas Linux/real-data serão produzidas pelo operador rodando 02b. Termos/revisão de rede e aceitação após corte permanecem pendentes, sem marcar os critérios runtime como concluídos.
