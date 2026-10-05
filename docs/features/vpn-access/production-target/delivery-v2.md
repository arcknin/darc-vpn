# Entrega 02b v2

Pacote SHA-256: `13d53351331b8e77cb0f000e136d89037076ec99b1ff03b33448e67d1b37e1d9`.

Destino: `/home/codex/darc-vpn-migration-20261004-v2/bundle.tar.gz`. Copiar/verificar antes de extrair para `/root/darc-vpn-migration-v2`; não substituir a cópia v1 nem o backup. Depois executar `bash /root/darc-vpn-migration-v2/02b-build-target.sh`.

Validação local: suíte Go race/vet/formatação, 13 testes stdlib e ShellCheck/sintaxe. Backend legado/PSK cifrado e importador estão implementados. Não foram executados build Linux/cold restore com dados reais nem comandos root na VPS pelo assistente; o 02b executado pelo dono produz essas provas. Sem settings reais, prepara legados/templates mas não libera 03.

Entrega remota confirmada: pacote/hash e todos os arquivos do manifest conferidos, help/sintaxe 02b validado e chamada sem root recusada; wg-quick@wg0 continua active. Rejeição inicial de upload resolvida por auditoria fonte-only sem key/db/config reais e verificação de srv1519215/codex/diretório modo700, seguida de envio autorizado.
