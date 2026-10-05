# Spec integrada: atualização API sem recriar WireGuard

Escopo autorizado pelo pedido direto: overlay Compose de API, script root de build/update, backup e retorno de imagem, testes e documentação. Preservar banco, chave mestra, peers, namespace, modo privado e credenciais. Não executar atualização da VPS automaticamente. TDD yes, continuidade da escolha anterior.

Ordem: testes de preservação do modelo/red; helper/overlay/runner; testes do fluxo com CLI Docker falsa; shellcheck/syntax/Compose config; publicar para clone do dono.

Contrato: usar mesmo projeto/modelo/lock da migração verified; construir imagem antes da pausa, salvar dados coerentes com API parada, `up --no-deps --wait api`, saúde interna e comparação do container/início WireGuard. Persistir só image da API no modelo ativo. Falha tenta imagem anterior; nunca restaurar banco automaticamente. Config privada/segredos não aparecem no stdout.

Limites: breve indisponibilidade da API, VPN continua pelo mesmo container. Rollback da imagem não garante compatibilidade de migrações futuras; mudanças incompatíveis de dados exigem plano próprio. Atualização do WireGuard fica fora. [Operação](../../../../scripts/api-update/README.md).
