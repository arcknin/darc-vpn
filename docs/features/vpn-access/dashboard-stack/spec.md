# Spec — duas stacks Docker

Autorização: pedido direto do dono em 2026-10-05 para preparar os Composes. Continuidade do TDD previamente escolhido. Escopo: dashboard Dockerfile/Compose/README, saída Next standalone e flag de HTTP interno em loopback; preservar Compose API/WireGuard e seus dados. Sem reiniciar ou implantar serviços reais nesta tarefa.

Entrega: stack API/WireGuard existente e independente; novo projeto `darc-vpn-dashboard`. Painel não root, sem capabilities, filesystem read-only, escuta127.0.0.1:3002 e API127.0.0.1:18082 via rede host Linux. Healthcheck da tela de login, logs limitados, build com npm ci. Flag de loopback não libera HTTP de navegador nem API remota HTTP.

Ordem: teste negativo/red para acesso interno mantendo HTTPS; implementação/green; build standalone e validação dos dois Composes; Docker build quando daemon disponível. Prova de navegador/conta real fica separada. Não duplicar ou executar a stack de preparação da VPN contra identidade/portas em uso.

Aceitação local: testes/tipos/lint/build Next e Compose config. Limitações Docker/HTTP reais devem constar no checkpoint; nenhuma prova fake equivale a implantação. [Roteiro](../../../../infra/dashboard/README.md).
