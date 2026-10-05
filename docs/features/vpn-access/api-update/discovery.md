# Discovery: atualização do backend

A stack ativa usa o projeto darc-vpn-prod-main e o modelo root-only `/var/lib/darc-vpn-migration/runtime-model.json`. A API compartilha namespace com WireGuard; dados/chave mestra ficam no bind /data externo ao clone. Modelo de preparação no Git não é substituto automático do runtime. Migração registra stage verified e lock operation.lock. Não havia fluxo de build/troca de imagem para updates comuns.

Pedido do dono em 2026-10-05: criar suporte Compose para atualizar backend. Dashboard já tem Compose independente. Docker local parado: validação local pode testar modelo/controle com Docker falso e validar composição; implantação real não autorizada nesta tarefa de preparação.
