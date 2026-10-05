# Termos de Uso — Darc VPN

> **Rascunho para revisão antes da publicação.** Responsável legal e vigência pendentes; política de retenção aprovada pelo dono, implementação e enquadramento dos registros VPN ainda pendentes. Não publicar ou marcar revisão concluída só para passar no script.

**Versão proposta:** 1.0  
**Vigência:** [DEFINIR]  
**Nome do projeto:** Delta A'rcanix  
**Responsável atual pelo serviço e pelos dados:** Lucas Reis (pessoa responsável enquanto a empresa não estiver formalizada)  
**Contato para dúvidas e pedidos sobre dados pessoais:** dev.lucasreis.santos@gmail.com

## 1. Uso do serviço

A Darc VPN é oferecida somente a pessoas autorizadas pelo administrador. O serviço pode ser gratuito ou oferecido mediante cobrança. Quando houver cobrança, valor, período, condições de pagamento e cancelamento devem ser apresentados e aceitos antes da contratação; este texto não fixa preços nem implementa pagamentos. Você deve usar sua conta e seus perfis de acesso de forma lícita e não deve ceder suas credenciais a terceiros sem autorização. O uso da VPN não torna atividades ilegais permitidas ou anônimas.

## 2. Registros e privacidade

Os perfis e as chaves privadas guardados pelo backend serão criptografados em repouso. O túnel WireGuard criptografa o tráfego entre seu dispositivo e o servidor VPN; o servidor precisa descriptografar os pacotes para encaminhá-los. Essa criptografia não significa que o servidor seja tecnicamente incapaz de observar destinos ou outros dados de tráfego.

O serviço registrará o identificador da conta e do perfil, o endereço IP de origem e a data e hora em UTC a cada autenticação e chamada autenticada do app/API, sem gravar o conteúdo da chamada. Também registrará esses dados nos eventos de conexão VPN que o servidor conseguir observar. Esses registros são metadados de acesso. **Não manteremos histórico dos sites, destinos ou conteúdo acessados pela VPN.** Antes da publicação, essa regra deve ser implementada e conferida também nos logs do sistema, firewall e DNS; o prazo de retenção dos metadados precisa ser informado aqui: **seis meses para metadados de acesso ao app/API, ressalvadas hipóteses legais de conservação. A classificação e política de eventuais registros de conexão VPN precisam ser confirmadas separadamente antes da publicação**.

Os registros serão acessados apenas por pessoas autorizadas e protegidos contra acesso indevido. Informações sobre seus dados e pedidos relativos a eles podem ser enviados ao contato indicado acima.

## 3. Investigações e pedidos legais

Em investigação de possível crime ou outra situação prevista em lei, os registros existentes, inclusive endereços IP de origem e horários, poderão ser preservados e fornecidos às autoridades competentes **quando houver ordem judicial ou outra hipótese legal aplicável**. A autoridade pode solicitar esses registros, mas um pedido informal não significa entrega automática. Como não manteremos histórico dos destinos nem do conteúdo acessado, não poderemos fornecer esse histórico; os metadados disponíveis tampouco garantem a identificação de quem praticou uma atividade específica na internet.

## 4. Descumprimento

O administrador poderá bloquear uma conta por uso indevido ou violação destes Termos, observadas as regras de bloqueio e revogação de perfis do serviço.

## 5. Aceite

Antes de receber seus perfis de VPN ou exibir seus QR codes, você deve ler estes Termos e selecionar **“Li e aceito”**. O backend registrará o identificador da conta, a versão e o hash do texto aceito, a data e hora do aceite e um identificador do evento. O comprovante será mantido em JSON em um diretório persistente e configurável do servidor, separado dos arquivos de build. O comprovante será mantido enquanto a conta estiver ativa e por seis meses após seu encerramento definitivo, ressalvadas as hipóteses legais de conservação.

## 6. Atualizações

Alterações destes Termos serão identificadas por nova versão e apresentadas ao usuário. A regra para exigir um novo aceite quando a versão mudar ainda precisa ser definida antes da publicação.
