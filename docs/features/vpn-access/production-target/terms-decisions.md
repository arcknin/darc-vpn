# Decisões para fechar os termos

Dono informou: identificação Delta A'rcanix; contato dev.lucasreis.santos@gmail.com; uso para amigos com possibilidade de cobrança; retenção ainda indefinida ("sla"). Não inferir serviço exclusivamente gratuito, pessoa jurídica ou consentimento para um prazo arbitrário.

Propostas aguardando decisão: seis meses para metadados app/API; comprovantes enquanto conta ativa + seis meses após encerramento definitivo, com ressalvas legais. São propostas de política operacional, não conclusão sobre obrigação de guarda específica do operador/VPN. Responsável precisa confirmar PF/empresa e nome efetivo se identificação for marca.

Fontes primárias consultadas: [Marco Civil](https://planalto.gov.br/ccivil_03/_ato2011-2014/2014/lei/l12965.htm), art. 5/13/15/16; [LGPD](https://planalto.gov.br/ccivil_03/_ato2015-2018/2018/lei/l13709compilado.htm), arts. 4/15/16; [ANPD — perguntas frequentes](https://www.gov.br/anpd/pt-br/acesso-a-informacao/perguntas-frequentes), item 5.5. Art. 15 Marco Civil prevê seis meses para provedores de aplicação PJ que exerçam atividade organizada/profissional/econômica; não aplicar automaticamente essa classificação à VPN ou a uma PF. Guarda de conexão tem enquadramento próprio; não equiparar logs próprios do app com histórico de sites de terceiros.

Limites técnicos verificados: audit.JSONSink escreve metadados na saída de logs, sem descarte por data; Compose de produção não configura retenção temporal; receipt store grava JSON e SQLite sem expiração. Porta de eventos VPN existe, mas não foi encontrada alimentação automática por observação WireGuard no caminho atual. Prazo em texto não implementa expurgo; não prometer coleta histórica/eliminação que ainda não foram implementadas/conferidas.

Antes de publicar: decisões de identidade/retenção, adequação legal conforme forma de operação, mecanismo de retenção/descarte dos dados e backups, evidência dos logs do app/API/sistema/firewall/DNS e definição dos eventos VPN realmente registrados. Proposta continua rascunho; não marcar production_terms_reviewed=true só para destravar 02b. Implementação de expurgo/monitoramento é alteração própria e deve ser delimitada antes de executar.

## Decisões posteriores do dono

Delta A'rcanix é o projeto de uma empresa que ainda será aberta; não apresentá-la como pessoa jurídica existente. Responsável pessoa natural precisa fornecer nome completo, solicitado sem CPF. Nome não será inferido do endereço de e-mail.

Dono aprovou a proposta: metadados app/API por seis meses; comprovantes de aceite durante a conta ativa + seis meses após encerramento definitivo, ressalvas legais. Aprovação é política desejada, não evidência de que expurgo/logs/fechamento de conta estejam implementados ou validação do enquadramento legal da VPN. Rascunho atualizado; prazo de registros VPN continua exigindo confirmação própria.

Responsável informado diretamente pelo dono: Lucas Reis. Contato dev.lucasreis.santos@gmail.com; projeto Delta A'rcanix. Identificação pessoal preenchida sem inferir CPF/CNPJ ou empresa constituída.

Texto proposto sem placeholders editoriais: terms-v1.0-proposed.md. Isso não é evidência de aprovação operacional/jurídica nem publicação. Continua necessário decidir/implementar procedimento de retenção e verificar logs/eventos VPN; responsabilidade/contato e política de seis meses foram confirmados pelo dono. A forma de operar/cobrar como PF e o enquadramento de eventuais registros de conexão não foram tratados como uma empresa já constituída.
