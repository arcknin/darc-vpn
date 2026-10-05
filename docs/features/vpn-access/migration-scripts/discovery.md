# Discovery: pacote Bash de migração para execução pelo dono

Pedido atual: criar scripts e entregá-los no usuário codex da VPS; dono executará como root. Não executar operações de migração pelo assistente.

Fontes diretas: ../wireguard-stack/validation.md, ../../../../infra/wireguard-stack/compose.yaml e wireguard-entrypoint.sh, ../../../../back-vpn/internal/wireguard/manager.go, migrations/001_b02_identity.sql. Workspace sem Git; nenhuma instrução AGENTS.md em arquivo encontrada no inventário, instruções da conversa aplicadas.

Stack validada é de laboratório e inicia sem peers. Manager aceita apenas peers conhecidos no SQLite; EnsurePeer não recompõe PSKs legadas após bootstrap vazio. Não existe importação de peers antigos/client PrivateKeys desconhecidas. Isso bloqueia o corte real, não a entrega de inspeção/backup e um runner com pré-condições verificáveis.

Desenho do pacote: cinco entradas Bash, driver compartilhado e helper Python stdlib para processamento seguro. Inventário bruto e backups root-only; resumo sanitizado. Corte exige implantação final e evidência root de reconciliação/restauração fria com a mesma identidade/peers/PSKs. Root-owned plano e Compose congelado; rollback por systemd timer para sobreviver à queda da sessão SSH; não restaura/limpa firewall global.

Autorização: criar/testar localmente e enviar código ao usuário codex. Operações root apenas pelo dono. TDD/validação: testes negativos escritos antes do helper, continuando a preferência de testes do trabalho de servidor; não transferir aprovação de produção do laboratório.

Próximo passo do dono: congelar/verificar o pacote em /root, executar 01-inspect.sh e compartilhar apenas sanitized-summary.json. Fechar desenho de produção/importação antes de cutover.

## Atualização do operador

Dono executou inspect e prepare backup-only e forneceu resumo sanitizado; ver [owner-inventory.md](owner-inventory.md). O próximo passo original de inventário/backup está cumprido. Falta implantação final/persistência legada; nenhum corte foi executado.
