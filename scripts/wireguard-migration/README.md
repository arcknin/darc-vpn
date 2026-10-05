# Scripts operacionais WireGuard — execução pelo dono como root

Este pacote prepara um procedimento, não uma migração já aprovada/testada em produção. O envio ao usuário codex não executa operações privilegiadas. A stack de laboratório **não é um artefato de produção**: começa sem peers; o backend ainda não tem importação dos peers antigos/PSKs nem gestão persistente de peers legados. Corte com ela interromperia os acessos.

## Etapas

| Script | Ação |
| --- | --- |
| 01-inspect.sh | Lê configuração/estado, hooks, rotas, firewall e redes; salva tudo em diretório root-only e imprime apenas resumo sanitizado. |
| 02-prepare.sh | Cria backup íntegro sem parar wg0; opcionalmente valida plano de produção. Sem plano, termina em backup-only, impedindo corte. |
| 03-cutover.sh | Exige plano/evidência correspondentes, confirmação explícita e ausência de drift. Agenda rollback fora da sessão SSH antes de parar o host e iniciar o alvo. |
| 04-verify.sh | Confere identidade/peers/PSKs no kernel, API pronta e declaração root de teste com cliente externo. Só então cancela rollback. |
| 05-rollback.sh | Para alvo, restaura configuração e inicialização originais e valida os peers do host. Mantém backups e dados do alvo. |

As cinco entradas usam migration.sh (Bash). inventory.py usa Python 3 padrão para comparar JSON/configurações sem avaliar texto como shell. Dependências no Linux: Bash 4+, WireGuard tools, iproute2, iptables/ip6tables, Docker Compose, systemd/systemd-run, flock, timeout, sha256sum e Python 3. O script não instala nada.

## Primeira execução segura

O diretório entregue pertence a codex. **Não execute dali como root.** Copie o arquivo de pacote para `/root`, confira o SHA-256 comunicado pelo assistente e só depois extraia. Isso evita executar código que um usuário não root possa alterar durante o corte. Não use checksum fornecido apenas pelo mesmo arquivo não verificado como prova de autenticidade.

Exemplo (substituir o caminho do pacote entregue e seu hash real):

```bash
install -d -m 700 /root/darc-vpn-migration
install -m 600 /home/codex/darc-vpn-migration-20261004/bundle.tar.gz /root/darc-vpn-migration/bundle.tar.gz
# Verifique sha256sum /root/darc-vpn-migration/bundle.tar.gz contra o hash informado no chat.
# Apenas se corresponder:
tar -xzf /root/darc-vpn-migration/bundle.tar.gz -C /root/darc-vpn-migration
chown -R root:root /root/darc-vpn-migration
chmod 700 /root/darc-vpn-migration
bash /root/darc-vpn-migration/01-inspect.sh
```

Compartilhe somente `sanitized-summary.json` e o texto resumido. **Não cole wg0.conf, live.conf, live.dump, logs brutos, environment ou backups no chat:** wg showconf/dump incluem material privado. Os arquivos ficam em `/var/lib/darc-vpn-migration`, protegido por root, arquivos 0600.

Depois do inventário, pode fazer backup sem plano e sem interrupção:

```bash
bash /root/darc-vpn-migration/02-prepare.sh
```

Se o arquivo persistente e o estado ativo divergirem, o backup permanece salvo, mas prepare recusa declarar prontidão. Resolver a diferença exige decisão root; o script não reescreve silenciosamente peers/configuração.

## O que falta antes do corte

1. Analisar os dados root de hooks/firewall/routing: não copiar as regras duplicadas, não limpar regras de outros serviços.
2. Preparar um Compose de produção root-owned, com dois serviços, imagens já construídas fixadas por ID sha256, sem builds no corte, sem socket/host network/privileged, API loopback e identidade/porta/endereços originais.
3. Implementar e testar a importação/persistência dos peers legados e PSKs, com reservas que não conflitem com novos perfis. O backend atual não entrega esse mecanismo. Não inventar a PrivateKey de cliente que o servidor não possui.
4. Fechar termos reais, segredos persistentes, exposição da API e permissões. Fixtures do laboratório são rejeitadas pelo validador.
5. Provar restauração fria e reconciliação da implantação final; produzir restore-proof.json root-owned, associada ao baseline atual e ao hash do `docker compose config --format json` resolvido. O script verifica correspondência, mas **a prova é uma declaração do operador sobre testes reais**, não um teste independente criado pelo validador.

Copie production-plan.example.json para um arquivo **fora do pacote congelado**, ajuste os caminhos após essas decisões e faça `02-prepare.sh --plan /root/production-plan.json`. Não edite os scripts/manifests do pacote congelado.

Contrato de restore-proof.json: schema_version=1; configuration_signature do resumo; target_config_sha256 do modelo resolvido; passed, legacy_peers_preserved, psks_preserved, reconcile_passed, cold_restore_passed, routing_firewall_reviewed e production_terms_reviewed devem ser true após testes/revisão reais. Não fabricar flags para destravar o corte.

## Corte/validação/retorno — somente quando a preparação declarar prepared

```bash
bash /root/darc-vpn-migration/03-cutover.sh --confirm-cutover
# Testar um perfil existente de cliente externo: conexão, túnel v4/v6 e HTTPS v4/v6.
# Registrar declaração root em traffic_proof com run_id impresso no corte e:
# existing_client_connected, ipv4_tunnel, ipv6_tunnel, https_ipv4, https_ipv6=true.
bash /root/darc-vpn-migration/04-verify.sh
# Se falhar ou decidir voltar, antes da verificação:
bash /root/darc-vpn-migration/05-rollback.sh
```

Rollback programado padrão: 600 segundos, configurável de 300 a 1800 no plano. Unidade systemd transitória funciona fora da sessão SSH; perda de SSH não é contada como aceitação. Timer não sobrevive a reboot; reboot durante a janela não faz parte do procedimento. Não executar reboots ou administração concorrente de wg0 durante a migração.

O modelo Compose validado é congelado no estado root para executar e fazer rollback mesmo se o arquivo original for alterado. Não se restaura/limpa o firewall global automaticamente; wg-quick executa os hooks originais. Se hooks falharem e deixarem interface/regra parcial, recuperação manual pode ser necessária; logs e backup ficam protegidos para o operador. Não há promessa de interrupção zero ou rollback infalível.

Após verify, o script não faz rollback automático de uma implantação que já foi aceita; retorno posterior exige plano atualizado. Não apaga volumes/banco de produção nem backups. A limpeza definitiva só vem depois da aceitação e decisão própria.

## Validação deste pacote

Testes locais de parser/guardas, Bash syntax, ShellCheck e negativa sem root. Nenhum inspect/prepare/cutover/verify/rollback privilegiado foi executado na VPS pelo assistente. Validação real começará pelo inspect que o dono rodará como root.

Fontes primárias consultadas: [WireGuard wg(8)](https://git.zx2c4.com/wireguard-tools/about/src/man/wg.8) e [Compose config](https://docs.docker.com/reference/cli/docker/compose/config/).

## Versão 2: etapa 02b implementada

`02b-build-target.sh` usa o backup criado pelo 02 para construir o alvo, importar/cifrar peers/PSKs e testar cold restore duas vezes, sem parar host ou publicar outro servidor. O pacote v2 inclui fontes verificáveis em payload/; Docker build não recebe keys/SQLite do alvo como contexto.

Depois de copiar/verificar/congelar **o pacote v2** em `/root/darc-vpn-migration-v2`, execute:

```bash
bash /root/darc-vpn-migration-v2/02b-build-target.sh
```

A etapa usa o mesmo estado `/var/lib/darc-vpn-migration` e o backup existente. Sem settings finaliza apenas a preparação legada e mostra o caminho da configuração faltante; não inventa termos nem marca revisão concluída. Para finalizar o plano:

```bash
# Copiar /opt/darc-vpn-production/production-settings.example.json para arquivo root-owned fora do bundle;
# preencher termos reais e marcar revisões somente após fazê-las.
bash /root/darc-vpn-migration-v2/02b-build-target.sh --settings /root/production-settings.json
bash /root/darc-vpn-migration-v2/02-prepare.sh --plan /opt/darc-vpn-production/production-plan.json
```

A implementação atual do backend **agora tem** registro legado/PSK persistente; os avisos anteriores sobre sua ausência descrevem a versão 1. A prova runtime com dados reais ainda precisa do 02b executado pelo dono. O 03 não foi executado.

## Versão 3: teste privado sem barreira de termos

Ordem do dono: termos/retencão finais ficam para antes de disponibilizar o serviço a usuários, sem travar teste técnico atual. Usar `02b-build-target.sh --private-test` na cópia root verificada v3. Isso gera a configuração final técnica, testa API isolada e mantém as provas de identidade/peers/PSKs; não exige terms_file nem as flags manuais de revisão final. O backend permite acesso autenticado aos próprios perfis sem aceite neste modo explícito, conserva autorização/readiness e mantém exigências normais nos demais modos.

A API continua publicada somente em loopback; não é liberação comercial/para público. Não marcar termos como revisados por conveniência: o proof usa private_test_only=true e terms_deferred=true ligado ao modo/container correto. Backups, proteção de chaves, colissões de rede/porta, cold restore, confirmação explícita do 03 e rollback continuam. O 02b não faz corte. Depois dele, preparar plano com o 02 e só então operador pode executar 03/04.
