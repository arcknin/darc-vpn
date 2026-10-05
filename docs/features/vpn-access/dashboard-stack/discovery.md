# Discovery — stacks separadas

Pedido do dono: API/WireGuard juntos e dashboard separado; clone da VPS em `/opt/darc-vpn`. Compose API/WireGuard já existe em `infra/wireguard-production/compose.yaml`; migração ativa usa modelo runtime com dados/chaves externos ao clone. Painel Next não tinha Dockerfile, e HTTP interno em produção só era permitido via modo local que também relaxa HTTPS do navegador.

API observada em loopback18082; Nginx preparado para painel loopback3002. Rede host Linux permite ao painel acessar API sem mover serviços/chaves ou publicar outra porta. Docker local está parado; Docker Compose CLI standalone disponível para validar YAML. Não implantar/recriar VPN para testar o painel.
