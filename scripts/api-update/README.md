# Atualizar somente o backend na stack existente

O Compose de operação é `/var/lib/darc-vpn-migration/runtime-model.json`, com o projeto `darc-vpn-prod-main`. O clone contém o modelo de preparação e o overlay `infra/wireguard-production/compose.update-api.yaml`; não iniciar outra stack com os mesmos dados/portas.

Na VPS como root:

```bash
cd /opt/darc-vpn
git pull --ff-only
bash scripts/api-update/update.sh
```

O script constrói a API com o Dockerfile existente, fixa o ID sha256 da imagem, mantém configuração/credenciais/volumes do modelo ativo e usa `--no-deps` para recriar **somente API**. Depois da build, pausa a API para salvar banco, chave mestra e comprovantes juntos em backup privado. WireGuard continua encaminhando tráfego; autenticação/criação/bloqueio ficam indisponíveis durante essa breve troca.

Antes da execução, a migração deve estar `verified`. O script usa o mesmo lock de operações. Após subir, aguarda saúde da API, executa `/app/api health`, compara ID e horário de início do WireGuard e atualiza atomicamente a imagem no modelo de operação. Resultado esperado: `API_UPDATE_OK`.

Falha após a pausa tenta voltar à imagem anterior, **sem restaurar automaticamente o banco**, para não apagar escritas ou comprovações. Backup e logs ficam em `/var/lib/darc-vpn-migration/api-update.*`, somente root; não compartilhar esses arquivos. Mudanças futuras incompatíveis de schema precisam de estratégia de migração própria: voltar a imagem pode não bastar; não chamar isso de restauração de dados.

Para escolher uma imagem anterior já construída, após avaliar compatibilidade do banco:

```bash
bash scripts/api-update/update.sh --image sha256:ID_COMPLETO_DA_IMAGEM
```

O script não atualiza o WireGuard, não refaz o corte e não altera termos/modo de teste. Uma atualização do container WireGuard exige janela e prova de cliente próprias por compartilhar namespace com a API. Dashboard mantém seu Compose independente.

## Evidência local

Nove testes Python: preservação de modelo, rejeições de configurações inválidas e ciclo completo com Docker falso em diretório temporário (sucesso, falha/retorno de imagem e migração não verificada). Bash syntax/shellcheck e composição do overlay validados. Isso não é execução contra containers reais. Docker local está parado; build da nova imagem e atualização na VPS não foram executados nesta tarefa.
