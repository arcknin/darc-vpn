# Dashboard Admin VPN

Dashboard Next.js do MVP VPN Access. O dashboard chama apenas as operações administrativas do contrato em `back-vpn/api/openapi.yaml`.

## Execução local

Defina a URL da API explicitamente. O modo local permite uma API HTTP somente em loopback:

```sh
VPN_API_BASE_URL=http://127.0.0.1:8080 \
DASHBOARD_LOCAL_MODE=true \
npm run dev
```

Em produção, `VPN_API_BASE_URL` precisa ser HTTPS. O dashboard não assume FQDN para si nem para a API.

## Sessão e segurança

- O bearer emitido pelo backend fica somente no cookie `vpn_admin_session`, com `HttpOnly`, `Secure`, `SameSite=Lax`, caminho raiz e expiração limitada.
- O navegador chama rotas internas do Next; o token não é exposto ao JavaScript do cliente e não é salvo em `localStorage` ou `sessionStorage`.
- O cliente administrativo modela somente DTOs de conta, perfil administrativo e revogação. Material privado de perfil não faz parte dos tipos, rotas proxy ou componentes do dashboard.
- Os estados `pending`, `failed` e `completed` são exibidos conforme a resposta do backend. Erros de rede, respostas parciais e respostas inválidas nunca são convertidos em `completed`.

## Verificação

```sh
npm run lint
npm run typecheck
npm test
npm run build
```

O cenário Playwright está em `tests/playwright/admin.spec.mjs` e requer que o pacote e um navegador estejam disponíveis no ambiente. Ele usa apenas uma URL local e credenciais de teste fornecidas pelo ambiente.
