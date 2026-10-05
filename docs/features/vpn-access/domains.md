# Domínios definidos pelo dono

- Backend HTTPS e endpoint WireGuard: `vpn.darcanix.com` (uso compartilhado definido em 2026-10-05): TCP443 via Nginx para API loopback18082, UDP51442 direto ao WireGuard.
- Site administrativo: `dashboard-vpn.darcanix.com` (confirmado pelo dono em 2026-10-05).

A confirmação do domínio do painel registra a escolha. DNS, certificado HTTPS,
proxy e implantação do dashboard ainda precisam ser configurados e verificados.
Dashboard: upstream planejado `127.0.0.1:3002`, porta livre no inventário SSH.
Script root preparado: `scripts/configure-nginx-domains.sh`. DNS de ambos deve
apontar diretamente a `187.124.240.109` antes da emissão dos certificados.
Status: configuração preparada; execução root, certificados e processo do
dashboard ainda não verificados.
