# Stacks separadas na VPS Linux

- API + WireGuard: `../wireguard-production/compose.yaml`, modelo de preparação da migração. A stack ativa conserva o modelo resolvido, identidade, banco e chave mestra criados na migração.
- Dashboard: `compose.yaml`, projeto `darc-vpn-dashboard`, independente. Não montar dados/chaves da VPN nem socket Docker no painel.

O dashboard usa rede host na VPS Linux para acessar a API já publicada somente em `127.0.0.1:18082`. O servidor Next escuta somente `127.0.0.1:3002`; Nginx termina HTTPS e sobrescreve X-Forwarded-Proto. Não há publicação Docker extra. `DASHBOARD_ALLOW_LOOPBACK_API` autoriza HTTP apenas para a API em loopback; o navegador continua exigindo HTTPS, cookies Secure e HttpOnly. `DASHBOARD_LOCAL_MODE` fica false.

## Subir apenas o dashboard

Na VPS, dentro do clone atualizado em `/opt/darc-vpn`:

```bash
docker compose -f infra/dashboard/compose.yaml config --quiet
docker compose -f infra/dashboard/compose.yaml up -d --build --wait
docker compose -f infra/dashboard/compose.yaml ps
curl --fail --silent --show-error -o /dev/null http://127.0.0.1:3002/login
```

O Dockerfile usa lockfile (`npm ci`), build Next standalone e runtime sem root. Não inclui credenciais administrativas nem dependências/builds locais no contexto. Healthcheck confirma só a tela de login; integração API e autenticação devem ser verificadas separadamente no navegador, com a credencial existente do dono.

Para parar somente o painel:

```bash
docker compose -f infra/dashboard/compose.yaml down
```

Não executar o Compose de preparação API/WireGuard como uma segunda stack contra as mesmas portas/chaves. A VPN já migrada usa o projeto `darc-vpn-prod-main` e o modelo ativo `/var/lib/darc-vpn-migration/runtime-model.json`; o clone novo não substitui esses dados. Para inspecioná-la como root, sem recriar:

```bash
docker compose --project-directory /var/lib/darc-vpn-migration \
  -p darc-vpn-prod-main -f /var/lib/darc-vpn-migration/runtime-model.json ps
```

O arquivo ativo contém configurações privadas: não compartilhar `config`, modelo JSON ou env da VPN. Mudanças de imagem/stack da VPN têm validação e rollback próprios; subir o painel não exige repetir o corte.

Referências: [Next standalone](https://nextjs.org/docs/app/api-reference/config/next-config-js/output), [rede host Linux](https://docs.docker.com/engine/network/drivers/host/).
