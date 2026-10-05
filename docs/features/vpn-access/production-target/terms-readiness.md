# Prontidão dos termos 1.0

Identificação aprovada pelo dono: responsável Lucas Reis; projeto Delta A'rcanix, empresa ainda não formalizada; contato dev.lucasreis.santos@gmail.com. Uso para amigos com possível cobrança; nenhuma tabela de preços ou infraestrutura de pagamentos autorizada por este documento.

Política aprovada: metadados app/API por seis meses; comprovantes enquanto conta ativa + seis meses após encerramento definitivo, ressalvas legais. Texto sem placeholders preparado em terms-v1.0-proposed.md para leitura do dono, não publicado nem enviado ao servidor.

Não confundir prontidão editorial com execução operacional. O código atual escreve audit metadata em stdout/Docker logs e comprovantes em JSON/SQLite, sem expiração. Falta definir/implementar procedimento de guarda e descarte, incluindo backups e significado operacional do encerramento definitivo (bloqueio não equivale a encerramento). Eventos VPN automáticos e política de registros da infraestrutura também não têm prova completa neste marco.

Próxima entrega necessária: mecanismo/procedimento de retenção delimitado, com teste de corte temporal para metadados, recibos de contas ativas preservados, recibos de encerradas eliminados só após prazo/ressalvas e evidência de não apagar perfis/peers/contas por efeito colateral. Execução em dados reais continua com operador root, nunca automaticamente pelo assistente.

Após revisão do texto e da operação: materializar arquivo /root/terms-final.txt; terms_file nesse caminho e terms_version=1.0. Marcar production_terms_reviewed apenas quando texto/realidade corresponderem; network review separada. Repetir 02b --settings, 02-prepare --plan e somente depois corte pelo dono.

## Ordem posterior do dono

Adiar termos/retenção finais até liberação a pessoas; não travar teste privado. Documentos ficam pendentes sem condicionar preparo técnico. Novo modo container-private-test não exige arquivo/aceite; proof registra adiamento, não revisão falsa. API host-loopback, autenticação, ownership, kernel/readiness e backup/rollback preservados.
