# Domínios definidos pelo dono

- Backend HTTPS e endpoint WireGuard: `vpn.darcanix.com` (uso compartilhado definido em 2026-10-05): TCP443 via Nginx para API loopback18082, UDP51442 direto ao WireGuard.
- Site administrativo: `dashboard-vpn.darcanix.com` (confirmado pelo dono em 2026-10-05).

A confirmação do domínio do painel registra a escolha. DNS, certificado HTTPS,
proxy e implantação do dashboard ainda precisam ser configurados e verificados.
Dashboard: upstream planejado `127.0.0.1:3002`, porta livre no inventário SSH.
Script root preparado: `scripts/configure-nginx-domains.sh`. `vpn` precisa de
DNS direto para `187.124.240.109` por atender WireGuard; `dashboard-vpn` pode
usar proxy Cloudflare, com origem nessa VPS. Validação ACME confirma acesso HTTP.
Status: dashboard Healthy informado pelo dono e login loopback3002 confirmado
por SSH. Configuração Nginx ainda não aplicada: script anterior barrava o proxy
do dashboard; exigência removida. Certificados/acesso HTTPS e login real pendentes.
