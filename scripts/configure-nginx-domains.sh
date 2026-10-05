#!/usr/bin/env bash
set -Eeuo pipefail
umask 077
[[ $EUID -eq 0 ]] || { echo 'Execute como root.' >&2; exit 1; }
for cmd in nginx certbot python3 systemctl; do command -v "$cmd" >/dev/null; done
nginx -t
# Both names must point directly to this VPS before ACME HTTP validation.
python3 - <<'PY'
import socket
for host in ('vpn.darcanix.com', 'dashboard-vpn.darcanix.com'):
    try:
        ips = set(socket.gethostbyname_ex(host)[2])
    except OSError:
        raise SystemExit(f'DNS ainda indisponível: {host}')
    if ips != {'187.124.240.109'}:
        raise SystemExit(f'{host}: configure A=187.124.240.109, Somente DNS, antes de executar.')
PY
marker='# Managed by darc-vpn configure-nginx-domains.sh'
for domain in vpn.darcanix.com dashboard-vpn.darcanix.com; do
    file="/etc/nginx/sites-available/$domain"
    link="/etc/nginx/sites-enabled/$domain"
    if [[ -e $file || -L $file ]]; then
        if [[ ! -f $file || -L $file ]] || ! grep -Fxq "$marker" "$file"; then
            echo "Configuração existente não gerenciada: $file. Nada sobrescrito." >&2; exit 1;
        fi
    fi
    if [[ -e $link || -L $link ]]; then
        [[ -L $link && $(readlink "$link") == "$file" ]] || {
            echo "Entrada existente incompatível: $link" >&2; exit 1;
        }
    fi
done
backup=$(mktemp -d /root/darc-nginx-backup.XXXXXXXX)
cp -a /etc/nginx "$backup/nginx"
echo "Backup Nginx: $backup/nginx"
for pair in vpn.darcanix.com:18082 dashboard-vpn.darcanix.com:3002; do
    domain=${pair%:*}; port=${pair##*:}
    file="/etc/nginx/sites-available/$domain"
    if [[ ! -e $file ]]; then
        # Nginx variables stay literal; only known host/port placeholders change.
        sed "s/__DOMAIN__/$domain/g; s/__PORT__/$port/g" > "$file" <<'NGINX'
# Managed by darc-vpn configure-nginx-domains.sh
server {
    listen 80;
    listen [::]:80;
    server_name __DOMAIN__;
    location / {
        proxy_pass http://127.0.0.1:__PORT__;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
NGINX
        chmod 644 "$file"
    fi
    [[ -L /etc/nginx/sites-enabled/$domain ]] || ln -s "$file" "/etc/nginx/sites-enabled/$domain"
done
if ! nginx -t; then
    echo "Nginx não recarregado. Corrija os novos arquivos ou restaure o backup: $backup/nginx" >&2
    exit 1
fi
systemctl reload nginx
certbot --nginx --non-interactive --agree-tos --redirect \
    --email dev.lucasreis.santos@gmail.com \
    --cert-name darc-vpn-sites \
    -d vpn.darcanix.com -d dashboard-vpn.darcanix.com
nginx -t
systemctl reload nginx
echo 'NGINX_HTTPS_OK: API -> 127.0.0.1:18082; dashboard -> 127.0.0.1:3002.'
echo 'O processo do dashboard ainda deve ser implantado na porta 3002; sem ele, o painel retorna 502.'
