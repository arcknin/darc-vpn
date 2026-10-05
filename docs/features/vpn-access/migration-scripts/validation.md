# Validação da entrega dos scripts

Entregue em 2026-10-04 em `/home/codex/darc-vpn-migration-20261004`, usuário codex da VPS srv1519215, via alias SSH darc-vpn.

Pacote `bundle.tar.gz`: SHA-256 `01f8ef519aa28f5baac5d2373a7005902ac72a6d6b792fcdc1466544b58ee213`. Esse é o valor a comparar depois de copiar o arquivo para root, antes de extrair/executar. Fonte local: [README](../../../../scripts/wireguard-migration/README.md) e [manifest](../../../../scripts/wireguard-migration/SHA256SUMS).

Validação local: 11 testes stdlib do helper (testes começaram falhando por ausência do helper), syntax Bash em todos os seis arquivos .sh e ShellCheck sem achados. Casos incluem segredo não exposto, perda/mudança de identidade/peer/PSK/AllowedIPs/porta, duplicatas, evidência ausente/incompatível, uso de fixture de laboratório, imagem não fixada, host networking, socket, privileged, capability ausente e publicação TCP externa.

Validação remota executada apenas como codex: hostname/usuário corretos, Bash 5.2.37; igualdade de hash do pacote e checksums de todos os arquivos; sintaxe/help e teste negativo de inspect sem root. Produção wg-quick@wg0 ativa e endereços observáveis preservados. Nenhum snapshot com segredo de produção foi lido/copied pelo assistente; nenhum diretório de estado root ou timer foi criado por esses testes.

Não executados: inspect/prepare/cutover/verify/rollback como root, teste de parada do serviço atual, restauração de firewall, importação de dados reais ou runtime privilegiado desta automação. Não declarar o procedimento de corte pronto com a stack atual: faltam importação legada/PSKs e prova de cold-restore/reconciliação com a implantação final.

O próximo comando operacional pertence ao dono root, depois de copiar/conferir o pacote em /root: `bash /root/darc-vpn-migration/01-inspect.sh`. Compartilhar somente sanitized-summary.json. Backup, modelo resolvido, dump e logs brutos permanecem root-only.
