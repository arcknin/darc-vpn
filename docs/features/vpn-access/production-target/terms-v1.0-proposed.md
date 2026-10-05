# Termos de Uso e Informações sobre Dados — Darc VPN

**Versão:** 1.0  
**Vigência:** a partir da disponibilização desta versão ao usuário e de seu aceite.  
**Responsável pelo serviço e pelos dados:** Lucas Reis.  
**Nome do projeto:** Delta A'rcanix.  
**Contato:** dev.lucasreis.santos@gmail.com.

## 1. Acesso e condições

O Darc VPN é destinado às pessoas autorizadas pelo administrador. Cada usuário deve utilizar suas credenciais e perfis de forma lícita, sem ceder acesso a terceiros sem autorização. A VPN não torna atividades ilícitas permitidas nem assegura anonimato absoluto.

O acesso pode ser gratuito ou sujeito a cobrança. Quando houver cobrança, valor, período de acesso, condições de pagamento, renovação e cancelamento deverão ser informados e aceitos antes da contratação. Estes termos não fixam preços nem autorizam cobranças que não tenham sido previamente combinadas.

## 2. Perfis e proteção

O aplicativo fornece os perfis disponíveis para a conta autenticada. Perfis distintos usam identidades WireGuard distintas. Exportar um perfil por QR reutiliza a identidade daquele perfil; sua utilização simultânea em aparelhos diferentes pode causar instabilidade ou perda de conexão. Credenciais, configurações privadas e QR devem ser tratados como secretos.

Os perfis privados e as PSKs armazenados pelo backend são protegidos por criptografia em repouso. WireGuard protege o tráfego entre dispositivo e servidor VPN; no servidor, os pacotes precisam ser processados para encaminhamento. Isso não torna o servidor incapaz de observar tráfego nem substitui a proteção HTTPS utilizada pelos serviços acessados.

## 3. Metadados, finalidades e retenção

O serviço trata os dados necessários para autenticar usuários, administrar contas/perfis, controlar acesso, proteger a operação e comprovar o aceite destes termos. Metadados de acesso ao app/API incluem identificadores da conta/perfil, IP de origem, tipo de evento e horário UTC; não incluem conteúdo de senhas, tokens ou material privado de perfil.

A política aprovada para metadados de acesso ao app/API é sua conservação por seis meses a partir do evento, seguida de eliminação, ressalvadas hipóteses legais de conservação ou preservação específicas. Esses registros próprios do app/API não correspondem a um histórico dos sites visitados através da VPN.

A política do serviço é não manter histórico dos sites, destinos ou conteúdo da navegação pela VPN. Eventos de conexão VPN que sejam registrados devem ter coleta, finalidade e retenção compatíveis com a operação e com as obrigações legais aplicáveis; não se confundem com registros de uso do app/API.

Comprovantes de aceite serão conservados enquanto a conta estiver ativa e por seis meses após seu encerramento definitivo, ressalvadas hipóteses legais de conservação. Bloqueio de acesso e encerramento definitivo são situações distintas. Contas, perfis/chaves e dados relacionados são conservados conforme sua necessidade operacional, pedidos aplicáveis e obrigações legais; não há autorização para guarda indiscriminada de dados sem finalidade.

Os registros devem ter acesso restrito a pessoas autorizadas. Solicitações sobre acesso, correção, eliminação ou outras questões relativas a dados pessoais podem ser enviadas ao contato indicado acima e serão analisadas segundo os direitos e as hipóteses de conservação aplicáveis.

## 4. Pedidos legais

Dados existentes poderão ser preservados ou fornecidos quando houver obrigação ou autorização legal aplicável. A disponibilização dos registros sujeitos a ordem judicial observará essa exigência; um pedido informal não gera entrega automática. O serviço não poderá fornecer histórico de navegação que não mantém, e metadados disponíveis não asseguram, por si só, a identificação de quem praticou determinada atividade.

## 5. Bloqueio e revogação

O administrador poderá bloquear o acesso por uso indevido ou violação destes termos. O bloqueio de conta impede o acesso ao app/API e inicia a remoção dos peers associados; a conclusão depende da confirmação da remoção no servidor. Perfis exportados dependem do controle de acesso no WireGuard, e não de uma consulta contínua ao aplicativo.

## 6. Aceite e atualizações

Antes do primeiro fornecimento dos perfis pelo aplicativo, o usuário deve ler a versão apresentada e confirmar “Li e aceito”. O serviço registra conta, versão e hash do texto, horário UTC e identificador do aceite em comprovante persistente.

Alterações serão identificadas por versão e comunicadas quando aplicável. Mudanças na cobrança ou na política de tratamento de dados não são autorizadas retroativamente por este aceite. O serviço deverá informar as condições aplicáveis antes de disponibilizar novas versões e funcionalidades.
