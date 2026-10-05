# VPN Access backend

API Go do MVP; identidade, sessões, perfis cifrados, termos, auditoria e bloqueio estão compostos em `cmd/api`. Contrato em `api/openapi.yaml`.

## Modos

Defina `VPN_WIREGUARD_MODE=fake` explicitamente para desenvolvimento local. O fake verifica protocolos e não altera servidor WireGuard. `VPN_MODE=local` mantém bind HTTP em loopback.

A stack em `../infra/wireguard-stack/` usa `VPN_MODE=container-lab`, `VPN_WIREGUARD_MODE=kernel` e namespace compartilhado com WireGuard. `wgctrl` configura somente a interface existente; o bootstrap é do container WG. Não há fallback kernel → fake.

Modo kernel requer interface wg0, pools IPv4/IPv6 e porta do endpoint, além das configurações existentes de banco, chave mestra, administrador, termos/comprovantes e auditoria. `VPN_RETRY_INITIAL`/`VPN_RETRY_MAX` controlam backoff (padrão 5s/60s). Um flock no diretório do banco garante único escritor; dados e chave mestra devem sobreviver a reinícios.

A API persiste intenções de provisionamento e reservas únicas, recompõe peers ativos e não recria revogados. `GET /readyz` verifica interface/reconciliação; `/healthz` é liveness. O bind 0.0.0.0 só é aceito em modo container-lab explicitamente configurado; publicação Docker deve restringir HTTP ao loopback do host, com acesso via SSH.

`peerPublicKey` é a identidade do cliente. O DTO privado em modo kernel inclui `serverPublicKey` e `endpointPort`, necessários à configuração [Peer]. Metadados administrativos incluem estado/erro de provisionamento sem segredos. Recuperação privada exige propriedade, aceite e confirmação de peer ready.

## Validação

```sh
go test -race ./...
go vet ./...
gofmt -l cmd internal test migrations
```

Testes kernel usam clientes simulados para erros/concorrência; prova WireGuard real pertence ao laboratório Linux descrito na [spec](../docs/features/vpn-access/wireguard-stack/spec-01-stack.md). Nenhum resultado local prova migração de produção nem funcionamento do app.
