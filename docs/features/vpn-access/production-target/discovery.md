# Discovery: alvo de produção e peers legados

Continuação solicitada após inspeção/backup pelo dono root. Fonte operacional: [owner-inventory.md](../migration-scripts/owner-inventory.md); três peers/PSKs, endereço 10.66.66.1/24 e fd42:42:42::1/64, porta 51442, fwmark zero, hooks/NAT duplicados. Inventário fornecido pelo dono, não repetido pelo assistente.

Fontes verificadas: back-vpn/internal/storage/managed.go só reserva IPs de perfis novos; internal/wireguard/manager.go rejeita peers fora desses registros; kernel.go não restaura PSK na configuração; internal/crypto/aesgcm.go fornece AES-GCM reutilizável; infra/wireguard-stack/api-entrypoint.sh gera chave mestra se arquivo ausente, comportamento que precisa falhar se já houver banco cifrado.

Consequência: stack de laboratório comprovada não migra peers existentes. São necessários registro durável legado separado de contas/perfis novos, PSKs cifradas, reservas dos AllowedIPs antigos e restauração pelo reconciliador. PrivateKeys dos clientes não são inferíveis do servidor e não serão inventadas.

Escopo discutido: preparar spec e depois, se aprovada, implementação/testes sintéticos e pacote final. Operador root importa dados reais e executa corte. Assistente não recebeu autoridade para operações root, banco real ou cutover. Preservar worktree sem Git e todas as fontes existentes.

Decisões propostas: peers antigos permanecem externos ao app, sem associação automática a contas; API inicialmente loopback/SSH para administração; publicação HTTPS/frontend ficam separadas. Requer aprovação do dono.
