# Inventário fornecido pelo operador root

Evidência enviada pelo dono nesta conversa; não é uma leitura root independente do assistente.

- Inspect: `/var/lib/darc-vpn-migration/inspections/20261004T214729Z-2430330`.
- Backup: `/var/lib/darc-vpn-migration/backups/20261004T215007Z-2430675`; prepare terminou backup-only, sem plano de produção/corte.
- Assinatura: `bdf982e05ee6c2d14a53a60bc5302abe07bf1ccea1f53c1d99f01a71e6640967`.
- wg0: 10.66.66.1/24 e fd42:42:42::1/64, UDP 51442, fwmark zero, três peers, três PSKs.

## Hooks classificados

Seis PostUp e seis PostDown simétricos: IPv4 INPUT UDP 51442 ACCEPT; IPv4 FORWARD eth0→wg0 ACCEPT e saída de wg0 ACCEPT; IPv4 MASQUERADE por saída eth0; IPv6 FORWARD de wg0 ACCEPT; IPv6 MASQUERADE por saída eth0. Não há PreUp/PreDown. O NAT atual não tem restrição de subnet nas regras dos hooks.

## Regras selecionadas do backup

IPv4: duas liberações INPUT UDP 51442, duas cópias de cada regra FORWARD envolvendo wg0, três MASQUERADE amplos em eth0, além de NAT específicos das bridges Docker 172.17/16 e 172.18/16. IPv6: duas liberações FORWARD wg0 e dois MASQUERADE amplos em eth0. Há cadeias Docker e UFW nas duas famílias.

A origem das duplicatas permanece desconhecida. PostDown remove uma ocorrência correspondente, não prova propriedade de regra nem elimina todas as cópias. Não limpar duplicatas/global firewall como parte automática do corte. A seleção omitindo outras regras não prova comportamento completo de UFW/Docker/IPv6 INPUT; não inferir que a policy ACCEPT isolada torna todo tráfego permitido.

## Consequência para produção

1. Preservar identidade, três peers/PSKs, endereços e UDP 51442; a assinatura fornecida ancora a futura prova de restauração.
2. Traduzir encaminhamento/NAT para o namespace do container, com NAT limitado aos pools VPN, como no laboratório; não transportar os hooks amplos do host literalmente. Eth0 dentro do container não é a interface física do host.
3. Manter regras preexistentes não atribuídas a esta migração; Docker administra somente as regras de seus novos recursos. Validar políticas efetivas com cliente, não só existência de regra.
4. Implementar persistência/importação de peers legados e PSKs na fonte de verdade do backend antes de usar a stack com API. Bootstrap atual inicia vazio; reconciliador desconhece os antigos e não recompõe PSK. Servidor não possui necessariamente PrivateKeys dos clientes: não inventar chaves/contas ao importar.
5. Preparar Compose/env de produção e executar teste de restauração fria/reconciliação antes do plano de corte. Provas devem ser reais, não flags editadas para vencer a guarda.

O inventário recebido resolve a classificação dos hooks e confirma as duplicatas; não resolve a implementação de peers legados nem fecha toda a política de firewall. Não é necessário pedir outro dump amplo ao operador neste marco. Próximo trabalho: especificar a implantação final/persistência legada e delimitar sua implementação; root executará o corte somente depois de validação.
