---
schema_version: 1
managed_by: spec-drive
feature: vpn-access
task: migration-scripts
revision: 1
status: approved_scope_direct_request
---

# Spec integrada: entregar scripts para o operador root

## Contexto e fontes

[Discovery](discovery.md); continuidade relacionada à stack validada em laboratório, não autorização para migrar produção. O pedido direto do dono autoriza escrever os scripts e enviá-los para codex; root executará.

## Objetivos e não objetivos

Entregar inspect, prepare, cutover, verify e rollback em Bash, com estado protegido e recusa segura quando faltar plano. Preservar fonte local e entregar cópia remota verificável. Não executar operações privilegiadas, importar peers/dados reais, implementar backend legado, publicar API, trocar DNS ou migrar VPN nesta tarefa.

## Desenho e execução

- Cinco wrappers e driver migration.sh; inventory.py stdlib apenas para parsing/validação, sem execução de hooks ou avaliação shell.
- Exigir execução a partir de cópia root-owned protegida e checksums; recusar scripts de checkout codex-writable.
- Inspect lê estado/config/firewall; dados secretos ficam root-only e somente resumo sanitizado é compartilhável.
- Prepare faz backup sem interrupção, recusa drift configuração/runtime; sem plano termina backup-only. Plano tem Compose/env root-owned, imagens fixadas já existentes e prova root de importação/restauração fria/reconciliação dos peers/PSKs e revisão de rede/termos reais.
- Cutover exige confirmação e prepared, modelo congelado e backup íntegro. Agenda rollback externo à sessão SSH antes de parar host; inicia só alvo validado e confere identidade/peer state. Falha tenta retorno sem flush global de firewall.
- Verify exige readiness, comparação de configuração WG e declaração root de teste externo com run_id correspondente; só então desarma timer. Não apresentar declaração de operador como verificação independente.
- Rollback retém dados/backups; reconhece limite de hooks/parciais e necessidade possível de recuperação manual.
- Entrega: scripts/wireguard-migration → diretório exclusivo /home/codex/darc-vpn-migration-20261004; gerar manifest e hash do pacote; verificar igualdade remota sem executar comandos root.

## Arquivos, autoridade e aceitação

Principal edita somente scripts/wireguard-migration e docs desta tarefa/índices. Pedido direto cobre implementação do pacote, testes e upload. Root actions não autorizadas ao assistente. Critérios: syntax/ShellCheck verdes; testes de redaction, drift, permissões/modelo inseguro e prova ausente; help/negativa sem root; manifest e hash iguais após upload; documentação explicita cutover ainda bloqueado pelo desenho/importação de produção.

## Validação, TDD, automação e riscos

Testes do helper foram escritos antes de sua implementação e falharam por ausência; executar validação proporcional de scripts. Não testar por parada real do host. Manifest de entrega não substitui autenticidade: root deve comparar hash recebido por canal da conversa antes de extrair a cópia protegida. Não declarar segurança infalível em rollback/hook/reboot. Scripts repetem preparação em backups novos e serializam operações; nenhuma remoção de dados real por cleanup automático.

## Aprovações e retomada

Escopo de escrita/envio autorizado diretamente pelo dono em 2026-10-04. Operações root, implantação/importação e corte não executados nem autorizados ao assistente. [Checkpoint](migration-scripts-checkpoint.yaml) registra a entrega; próximo passo é o operador root rodar inventário protegido.
