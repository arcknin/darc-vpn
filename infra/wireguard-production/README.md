# Alvo com preservação de peers antigos

Para atualizar o backend já migrado, use o [fluxo de atualização da API](../../scripts/api-update/README.md), com overlay Compose e backup. Não repetir a preparação/corte para uma atualização comum de código.

Esta é a stack **API + WireGuard**. O dashboard tem Compose próprio em
[`../dashboard/compose.yaml`](../dashboard/compose.yaml); consulte o
[roteiro das duas stacks](../dashboard/README.md). A VPN já migrada usa o
modelo runtime resolvido; não iniciar outra instância com este template.

O 02b constrói as imagens com API e importador offline, cria a identidade a partir do backup root, importa registros legados separados de contas e cifra PSKs no SQLite. Não cria PrivateKeys de clientes nem associa antigos peers a usuários.

O reconciliador restaura PSK/keepalive/AllowedIPs e reserva prefixes legados contra colisão de novos perfis. Pode preservar peers só IPv4 ou só IPv6; defaults /0 e sobreposição entre peers são recusados para revisão. Falta de chave mestra com banco existente falha fechada.

Bootstrap de produção exige identidade existente. Teste de cold restore usa --network none, sem publish, duas recriações e duas reconciliações por ciclo; nenhuma instância alternativa é exposta com identidade de produção. O 02b não para wg0 nem faz o corte.

Compose final fornece endereços 10.66.66.1/24, fd42:42:42::1/64, UDP 51442, API loopback TCP 18082, bridge 172.30.88/24 + fd42:30:88::/64 (checagem de conflitos antes de plano). NAT escopado aos pools dentro do namespace. Termos reais e credenciais são arquivos root-only, não fixtures no runtime.

Sem --settings, 02b prepara/importa/verifica e salva Compose/image IDs/prova de cold restore, mas não libera 03. production-settings.example.json em /opt/darc-vpn-production lista os campos faltantes: texto/versão dos termos, usuário administrativo e revisão de rede/termos. Tokens/passwords são gerados e retidos no diretório root-only, nunca publicados no console.

Com --settings root-owned válido, 02b ainda testa API/termos e reconciliação num namespace isolado antes de gerar restore-proof.json e production-plan.json. Depois executar 02-prepare.sh --plan /opt/darc-vpn-production/production-plan.json; só então 03 pode ser invocado pelo dono. A confirmação de cliente existente após corte permanece no 04.

Esta preparação faz escrita apenas no alvo /opt/darc-vpn-production e estado de migração. Backups originais e host WG não são reescritos. Imagens de build e dados alvo são mantidos para corte/diagnóstico; os containers temporários são removidos por trap. O comando operador root não foi executado pelo assistente.
