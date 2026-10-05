# Documentação Darc VPN

Entrada: [vpn-access](features/vpn-access/feature.md).

[Domínios definidos](features/vpn-access/domains.md).

| Tarefa | Papel | Fonte para retomar |
| --- | --- | --- |
| [mvp](features/vpn-access/mvp/) | Produto e implementação local existentes; contrato rev. 2 e plano rev. 3 preservados | [Checkpoint legado](features/vpn-access/mvp/mvp-checkpoint.yaml) |
| [wireguard-stack](features/vpn-access/wireguard-stack/) | Integração API/WireGuard validada em laboratório; migração posterior confirmada pelo operador | [Conclusão operacional](features/vpn-access/production-target/owner-migration-completion.md) e [validação local](features/vpn-access/wireguard-stack/validation.md) |
| [app-integration](features/vpn-access/app-integration/) | Continuação Flutter/backend e túnel nativo; teste Android USB pendente | [Spec](features/vpn-access/app-integration/spec-01-app.md) e [checkpoint](features/vpn-access/app-integration/app-integration-checkpoint.yaml) |

## Organização com spec-drive

Cada tarefa nova reúne `discovery.md`, uma spec integrada quando o escopo estiver fechado e `<task>-checkpoint.yaml` no mesmo diretório. Planejamento fica dentro da spec; não há necessidade de contrato, plano ou pastas de estágio separados.

O MVP mantém os arquivos legados e suas aprovações. Eles não autorizam automaticamente integração real ou migração. A nova tarefa referencia o MVP e registra decisões posteriores sem reescrever seu contrato aprovado.

Os READMEs dos projetos descrevem execução técnica; os documentos aqui registram decisões, evidências, limites de autorização e retomada. Estado histórico da VPS deve ser identificado como histórico.
