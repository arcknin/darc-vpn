# Laboratório da stack API/WireGuard

Dois serviços permanentes: API Go e WireGuard kernel. A API compartilha o namespace do WireGuard e usa wgctrl, com NET_ADMIN, sem Docker socket ou docker exec para administrar peers. Essa permissão também administra a rede compartilhada e pode consultar material do dispositivo; não equivale a isolamento de chave do servidor.

## Preparação

Use diretório remoto exclusivo sob `/home/codex/darc-vpn-labs/<LAB_ID>`, alias `ssh darc-vpn`, usuário codex. Faça primeiro `python3 scripts/wireguard-stack/preflight.py` na VPS. O script verifica alvo, kernel, portas 51443/18081, produção ativa e não sobreposição dos pools fixados neste laboratório; não executa alterações no host.

Crie `.lab-owner` com o ID `darc-vpn-stack-<data>-<sufixo>` e `.lab.env` modo 0600 contendo LAB_ID, LAB_WG_PORT=51443, LAB_API_PORT=18081, LAB_ADMIN_PASSWORD, LAB_AUDIT_TOKEN e LAB_ENDPOINT (IP efetivo da VPS). Gere segredos de teste aleatórios; nunca use contas/chaves de produção. Pools e endereços devem permanecer iguais entre API/bootstrap/Compose/aceitação; conflito exige nova configuração coerente, não ignorar o preflight.

Depois execute, na raiz enviada:

```sh
./scripts/wireguard-stack/lab.sh build
./scripts/wireguard-stack/lab.sh up
./scripts/wireguard-stack/lab.sh check
./scripts/wireguard-stack/lab.sh down
```

`check` cria contas de fixture e um cliente efêmero; execute com dados novos. Repetição contra o mesmo banco não é silenciosamente bem-sucedida. O script confirma tráfego, HTTPS com verificação TLS, revogação, reinícios e recriação. Remove seu cliente em finally, inclusive se falhar; o operador deve sempre executar down da stack na saída.

API TCP publicada apenas em loopback do host: acesso externo de teste por `ssh -L 18081:127.0.0.1:18081 darc-vpn`. Não alterar Nginx/DNS. UDP 51443 é só do laboratório; wg0 do host em 51442 permanece fora da stack.

## Persistência e recuperação

Volume identity mantém a chave do servidor; volume data mantém SQLite, chave mestra e comprovantes. O bootstrap cria interface vazia; só a API escreve peers, derivados de contas ativas e intenções persistidas. Não há arquivo de peers restaurado pelo wg-quick nem SaveConfig.

API e WG reiniciados/recriados devem recuperar os peers ativos sem revogados reaparecerem. Recriar WG pode trocar namespace: recrie/revincule também API. `lab.sh recreate` faz isso preservando os volumes. Readiness verifica interface e reconciliação; modo kernel nunca cai para fake. Segunda API escritora é recusada por flock no volume.

`down` apaga volumes e imagens exclusivas da tarefa. Antes de remover o diretório de upload, preserve somente acceptance-results.json e evidência sanitizada; elimine .lab.env e arquivo fonte com fixtures, sem manter configurações privadas. Nunca use docker prune ou remova recursos não manifestados.

## Compatibilidade

peerPublicKey continua sendo chave do cliente; serverPublicKey e endpointPort são os campos corretos para a configuração [Peer] do cliente. O QR Flutter atual precisa de atualização posterior; esta stack não corrige o app. Provisionamento administrativo expõe pending/ready/failed/revoked; recuperação privada exige ready, propriedade e aceite. IPs reservados não são reutilizados nesta revisão.

Esta entrega não migra a VPN existente, não testa reboot da VPS, não publica API HTTPS e não valida o aplicativo. Os testes de cliente dentro da VPS comprovam a integração e os caminhos de tráfego dessa stack; o handshake Mac histórico é evidência externa separada.
