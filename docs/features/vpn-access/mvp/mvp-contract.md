---
schema_version: 1
flow: orchestration-flow
feature: vpn-access
task: mvp
revision: 2
status: approved
owner_approved_at: 2026-09-29T03:10:24Z # source-registration timestamp for owner's approval in this chat
tdd: "yes"
implementation_authorized_at: 2026-09-29T03:10:24Z # source-registration timestamp; local code only
repositories:
  - front-vpn/ (Flutter app)
  - back-vpn/ (Go backend)
  - dashboard-admin-vpn/ (Next.js dashboard)
outcome: >-
  Oferecer a um grupo de amigos acesso simples ao serviço WireGuard existente:
  o administrador cria contas e perfis de acesso; cada amigo usa o app Android
  para gerenciar perfis, conectar o último perfil selecionado e exportar qualquer
  perfil por QR para importação opcional no app oficial WireGuard no iPhone/iPad.
in_scope:
  - Os três projetos indicados na raiz, com as tecnologias escolhidas pelo dono.
  - App Flutter somente para Android 7.0/API 24 ou superior, com ligar/desligar do túnel no próprio app por API nativa.
  - Login do amigo no app com as credenciais da conta; o backend fornece automaticamente os dados WireGuard necessários e o app Android configura o túnel sem exigir que o amigo copie arquivo, escaneie QR ou manipule parâmetros.
  - Na primeira utilização, o perfil principal é o selecionado por padrão. Selecionar perfil não conecta automaticamente; ao iniciar a conexão, o app conecta o último perfil selecionado e mantém essa escolha entre sessões do app.
  - Somente o administrador cria contas de amigos.
  - Cada conta começa com um perfil principal; somente o administrador pode adicionar slots/perfis extras.
  - O amigo logado vê o perfil principal e todos os slots extras configurados para sua conta; pode exibir o QR de qualquer perfil, inclusive o principal, para importá-lo no app oficial WireGuard no iPhone/iPad.
  - Os slots são perfis de acesso, não aparelhos físicos. Slots diferentes têm peers e pares de chaves próprios; exibir/importar o QR de um perfil copia o mesmo peer e a mesma chave, inclusive o perfil principal mesmo quando a conta tem só um slot.
  - Ao exibir QR ou perfil, o app Android mostra aviso claro de que usar simultaneamente o mesmo perfil/chave em dois aparelhos pode causar instabilidade e lentidão ou perda de conexão; o perfil compartilhado destina-se a uso alternado, sem promessa de simultaneidade estável.
  - Dashboard Next.js protegido por senha, para o administrador criar contas e gerenciar usuários, slots de perfil e acesso.
  - Backend armazena perfis e chaves privadas criptografados para permitir recuperação após reinstalação do app; somente a conta autenticada proprietária pode recuperar seus segredos, entregues pelo serviço, e o fluxo administrativo não precisa exibir chaves privadas.
  - Tráfego entre app/backend e dashboard/API usa transporte autenticado e criptografado.
  - Antes de entregar perfis ou QR, o app apresenta os termos de uso e exige aceite eletrônico Li e aceito. O backend persiste comprovante JSON em diretório de dados configurável e persistente, fora do build, com conta, versão e hash dos termos, data/hora UTC e identificador do evento.
  - Registrar conta/perfil, IP de origem e horário UTC a cada autenticação e chamada autenticada app/API e nos eventos de conexão VPN observáveis, sem armazenar histórico de sites, destinos ou conteúdo acessados. O texto ao usuário distingue criptografia em repouso dos perfis/chaves e criptografia do túnel até o servidor, que descriptografa pacotes para encaminhá-los.
  - Bloquear uma conta nega imediatamente o acesso app/API e, na mesma operação lógica, tenta remover todos os peers da configuração WireGuard ativa e do estado persistente. O bloqueio só é concluído no dashboard quando o servidor confirma a remoção de todos; até lá mostra pendente/falha. Após remoção confirmada, túneis ativos e QR já importados no WireGuard do iPhone/iPad deixam de ser aceitos pelo servidor.
  - Usar vpn.darcanix.com como hostname desejado do endpoint VPN.
  - O nome indicado para o dashboard administrativo na VPS do dono é dashboard-vpn; manter pendentes o FQDN completo do painel e a decisão de hospedar a API nessa VPS.
out_of_scope:
  - App Flutter para iOS, macOS ou qualquer outra plataforma além de Android 7.0/API 24 ou superior.
  - Prometer suporte a versões específicas de iOS/macOS ou a plataformas além da importação opcional de QR no app oficial WireGuard em iPhone/iPad.
  - Exigir QR, arquivo ou cópia de parâmetros como fluxo de configuração do app Android; a exibição opcional de QR para importar qualquer perfil no app oficial WireGuard em iPhone/iPad está dentro do escopo.
  - Garantir uso simultâneo estável do mesmo perfil/peer/chave em dois aparelhos físicos.
  - Integração live com o servidor WireGuard antes da sessão SSH que o dono abrirá e das decisões técnicas subsequentes.
  - Publicar em produção, alterar DNS ou operar credenciais reais sem autorização específica.
  - Exigir novo aceite para futuras revisões dos termos; o MVP aprova apenas o aceite da primeira versão publicada. Um perfil já importado por QR no app WireGuard não passa pelo app/API para revalidar aceite.
observable_behavior:
  - O administrador autentica-se por senha no dashboard e é o único perfil que cria contas.
  - Cada amigo entra no app Android com suas credenciais e recebe automaticamente do backend os dados de conexão necessários.
  - Na primeira utilização, o perfil principal está selecionado por padrão. Selecionar outro perfil não conecta; ao iniciar a conexão, o app usa o último selecionado e preserva a escolha entre sessões do app.
  - O amigo liga e desliga a VPN dentro do app Android por VpnService; a UI mostra o estado real confirmado do túnel, não apenas o resultado do pedido de ligar/desligar.
  - Se a permissão de VPN do Android for concedida, o app pode iniciar o túnel; se negada ou posteriormente revogada, o túnel não aparece como conectado e o app informa que a conexão não está ativa.
  - Uma conta começa com o perfil principal; somente o administrador pode adicionar perfis/slots extras.
  - O amigo logado vê o perfil principal e os slots extras configurados para sua conta, e pode exibir o QR de qualquer um deles, inclusive o principal.
  - O QR pode ser importado pelo app oficial WireGuard no iPhone/iPad; não há app Flutter iOS.
  - Slots diferentes correspondem a peers e pares de chaves diferentes; importar o QR de um slot em outro aparelho reutiliza aquele mesmo peer/chave e é destinado a uso alternado.
  - Ao mostrar um QR ou perfil, o app Android avisa que uso simultâneo do mesmo perfil/chave em dois aparelhos pode causar instabilidade e lentidão ou perda de conexão.
  - O backend armazena perfis e chaves privadas criptografados para recuperação após reinstalação do app.
  - Apenas a conta autenticada proprietária pode recuperar seus próprios perfis/segredos; o serviço pode descriptografá-los para entrega e o dashboard administrativo não exibe chaves privadas.
  - App/backend e dashboard/API usam transporte autenticado e criptografado.
  - Sessões de conta bloqueada não acessam app/API. Bloquear conta nega esse acesso imediatamente e inicia na mesma operação a remoção de todos os seus peers da configuração ativa e do estado persistente WireGuard.
  - O dashboard sinaliza bloqueio concluído somente depois que o servidor confirma a remoção de todos os peers; sem confirmação, informa pendente ou falha. Após sucesso, túneis ativos e QR importados no app WireGuard de iPhone/iPad deixam de ser aceitos.
  - O endpoint VPN usa vpn.darcanix.com; dashboard-vpn é o nome indicado para o painel na VPS, mas seu FQDN completo ainda precisa ser confirmado.
  - Sem aceite da versão inicial publicada dos termos, o app/API não entrega perfis nem QR. Após o aceite, o backend grava comprovante JSON persistente e verificável. O sistema mantém apenas metadados de acesso definidos neste contrato, sem histórico de destinos ou conteúdo da navegação.
interfaces_and_data:
  - "front-vpn/: app Flutter para Android; back-vpn/: backend Go; dashboard-admin-vpn/: projeto Next.js. dashboard-vpn: nome indicado para o painel na VPS, sem FQDN completo confirmado."
  - A senha do dashboard e as credenciais do amigo são credenciais de acesso; não são chaves WireGuard.
  - "Cada slot é um perfil WireGuard com peer e par de chaves próprios; o slot não é um aparelho físico. O QR exibe/transporta o perfil existente, reutilizando o mesmo peer e as mesmas chaves quando importado em outro aparelho."
  - O backend fornece automaticamente ao app os perfis disponíveis à conta após o login. O amigo pode exibir por QR qualquer perfil, inclusive o principal.
  - Um QR de configuração contém PrivateKey e pode conter PresharedKey; é uma credencial exportável e deve ser tratada como secreta.
  - O backend armazena os perfis privados, incluindo chaves, criptografados em repouso para recuperação após reinstalação. Somente a conta autenticada proprietária recupera seus segredos; o serviço pode descriptografá-los para entrega, e o fluxo administrativo não precisa exibir chaves privadas.
  - A recuperação deve falhar visivelmente se os dados criptografados não puderem ser recuperados. Mecanismo criptográfico e credenciais operacionais não foram definidos.
  - Perfis/chaves de outra conta, sessões bloqueadas e criação de slots pelo usuário devem ser recusados.
  - App/backend e dashboard/API comunicam-se por transporte autenticado e criptografado.
  - O comprovante de aceite contém conta, versão e hash dos termos, timestamp UTC e identificador do evento; arquivos JSON ficam em diretório persistente configurável, com acesso restrito e fora do build. A política de retenção e os dados do responsável precisam ser preenchidos antes da publicação dos termos.
  - Metadados de acesso incluem conta/perfil, IP de origem e horário UTC por autenticação e chamada autenticada app/API e por evento VPN observável; não incluem histórico de sites, destinos ou conteúdo. O servidor pode tecnicamente observar tráfego ao encaminhá-lo, embora o serviço não mantenha esse histórico.
constraints:
  - "O app tem suporte mínimo Android 7.0/API 24 e controla o túnel no próprio app por API nativa. A documentação oficial do Android identifica Android 7.0 como API 24, e a revisão referenciada do módulo Tunnel do WireGuard para Android também fixa minSdk 24: https://developer.android.com/about/versions/nougat/android-7.0-changes ; https://git.zx2c4.com/wireguard-android/commit/tunnel/build.gradle.kts?follow=1&id=2a017642822ccf95f673d3520f07d103f654f5f4."
  - A integração alvo é o serviço WireGuard existente no servidor do dono, mas o dono abrirá uma sessão SSH posteriormente; detalhes do servidor seguem pendentes e integração live não está autorizada.
  - "vpn.darcanix.com é exclusivo do endpoint VPN. dashboard-vpn é o nome indicado para o painel na VPS; o FQDN completo precisa ser confirmado, sem presumir dashboard-vpn.darcanix.com. Ainda não está decidido se a API também será hospedada na VPS."
  - "O app oficial WireGuard listado pela Apple para iPhone/iPad importa túneis por QR; a implementação oficial do scanner QR lê a configuração do túnel no código. Referências: https://apps.apple.com/us/app/wireguard/id1441195209 ; https://git.zx2c4.com/wireguard-apple/tree/WireGuard/WireGuard/UI/iOS/QRScanViewController.swift?h=0.0.20181104-1&id=bede8a17ad292120e414051a8c833543e73197a0. Isso não define versões de sistema suportadas pelo produto nem inclui app Flutter iOS/macOS."
  - "Um perfil WireGuard copiado via QR contém PrivateKey e pode conter PresharedKey, conforme o formato de configuração WireGuard: https://git.zx2c4.com/wireguard-tools/about/src/man/wg.8. O QR é credencial exportável."
  - "WireGuard identifica peers por chave pública e pode atualizar o endpoint de um peer ao receber tráfego autenticado desse peer. Portanto, o mesmo perfil/chave aberto simultaneamente em dois aparelhos pode disputar o endpoint, causando instabilidade e possível lentidão ou perda de conexão; essa consequência é uma inferência operacional da documentação, não mera perda de velocidade. Referências: https://www.wireguard.com/#conceptual-overview ; https://git.zx2c4.com/wireguard-tools/about/src/man/wg.8."
  - Bloquear conta nega imediatamente app/API e, na mesma operação lógica, tenta retirar todos os peers da configuração WireGuard ativa e do estado persistente. O dashboard confirma conclusão somente após confirmação do servidor para todos os peers; caso contrário mostra pendente/falha. Após confirmação, conexões ativas e cópias QR deixam de ser aceitas. Tática, timeout e retry ficam para code plan após levantamento SSH; não há SLA numérico neste contrato.
  - Todas as comunicações app/API e admin/API usam transporte autenticado e criptografado. Dados privados de perfil, inclusive chaves, são criptografados em repouso; somente a conta autenticada proprietária pode recuperar seus segredos, com descriptografia pelo serviço para entrega. o fluxo administrativo não precisa exibir chaves privadas.
  - Rejeitar sessões de contas bloqueadas, acesso a segredos/QR de outra conta e criação de slots pelo usuário; mostrar falha quando a recuperação de dados criptografados não for possível.
  - Manter o escopo simples sem inferir detalhes criptográficos, processo de rotação, limite de slots, autenticação adicional ou detecção de uso simultâneo.
  - A minuta docs/features/vpn-access/mvp/terms-of-use-draft.md orienta o texto inicial, mas seus campos de responsável, contato e prazo de retenção exigem preenchimento antes de exposição a usuários. Revisões futuras dos termos e reaceite não fazem parte desta aprovação.
acceptance_criteria:
  - "administrador com senha válida acessa o dashboard; somente ele pode criar contas de amigos."
  - "conta começa com um perfil principal; só o administrador adiciona slots extras, e o usuário só vê os próprios perfis. Usuário não consegue criar slots nem ler segredo/QR de outra conta."
  - "na primeira utilização o perfil principal está selecionado; selecionar outro perfil não conecta; ao iniciar a conexão o app usa o último selecionado, e a escolha continua após encerrar/reabrir o app."
  - "no Android 7/API 24+, com permissão VpnService concedida, iniciar conexão ativa o perfil selecionado; selecionar perfil não inicia conexão por si só."
  - "se a permissão VpnService for negada ou revogada, o app não declara conexão ativa e informa que o túnel não está conectado."
  - "a UI só mostra conectado quando o estado real do túnel Android/WireGuard confirma conexão ativa e reflete a desconexão observada."
  - "amigo pode exibir o QR de qualquer perfil disponível, inclusive o principal mesmo que seja o único slot, e importar esse QR no app oficial WireGuard em iPhone/iPad; isso não cria app Flutter iOS nem promete versão específica de sistema Apple."
  - "slots diferentes de perfil usam peers e pares de chaves distintos; ao copiar/importar um QR em outro aparelho, o perfil mantém o mesmo peer/chave e destina-se a uso alternado, sem promessa de simultaneidade estável."
  - "sempre que mostrar QR ou perfil, o app Android exibe aviso claro de que usar o mesmo perfil/chave ao mesmo tempo em dois aparelhos pode causar instabilidade e lentidão ou perda de conexão."
  - "backend armazena em repouso perfis privados e chaves criptografados e os recupera após reinstalação; se a recuperação falhar, o app informa claramente a falha. Só a conta autenticada proprietária recupera os segredos; serviço pode descriptografá-los para entrega; UI de admin não mostra chaves privadas."
  - "app/API e dashboard/API usam transporte autenticado e criptografado."
  - "bloquear conta nega imediatamente acesso app/API e, na mesma operação lógica, tenta remover todos os peers da configuração WireGuard ativa e do estado persistente."
  - "sucesso de bloqueio só aparece no dashboard após confirmação do servidor de que todos os peers foram removidos; sem confirmação, dashboard mostra pendente ou falha, nunca sucesso."
  - "após remoção de todos os peers confirmada, túneis já ativos e QR já importados no WireGuard do iPhone/iPad deixam de ser aceitos pelo servidor."
  - "vpn.darcanix.com é usado exclusivamente pelo endpoint VPN; o painel com nome indicado dashboard-vpn fica na VPS do dono, com FQDN completo a confirmar, e a hospedagem da API segue pendente."
  - "Sem aceite eletrônico dos termos da versão inicial publicada, usuário autenticado não recebe perfil privado nem QR; após Li e aceito, o backend grava comprovante JSON persistente com conta, versão/hash do texto, horário UTC e identificador do evento. Falha de gravação não concede acesso."
  - "Autenticações e chamadas autenticadas app/API registram conta/perfil, IP de origem e horário UTC; eventos VPN observáveis registram os mesmos metadados. O serviço não grava histórico de sites, destinos ou conteúdo acessados, inclusive em logs de infraestrutura sob seu controle."
validation_boundary:
  - Os critérios acima foram aprovados pelo dono nesta conversa; aprovação não é evidência de teste ou comportamento executado.
  - Planejamento, implementação e testes locais estão autorizados; integração live e acesso ao servidor WireGuard não estão autorizados, e o dono abrirá uma sessão SSH depois para levantar detalhes.
  - DNS, deploy, produção e uso de credenciais/dados reais requerem autorização específica.
  - Antes de publicar os termos ou usar dados reais, preencher responsável, contato e retenção e conferir o comportamento dos logs app/API, WireGuard, firewall e DNS.
authority_required:
  - Autorização específica para abrir/usar a sessão SSH, integrar ao servidor WireGuard, configurar DNS/TLS ou publicar em produção.
  - TDD `yes`, contrato e implementação local foram aprovados; isso não autoriza instalação, SSH, integração live ou produção.
failure_and_recovery:
  - Falha ao recuperar perfil criptografado deve ser apresentada claramente ao usuário; detalhes técnicos da recuperação ainda precisam de especificação.
  - Bloqueio nega imediatamente app/API; sem confirmação do servidor da remoção de todos os peers, dashboard mantém pendente ou informa falha, nunca sucesso.
  - Prazo numérico, timeout e retry não foram definidos. Regras para login inválido, conta/slot duplicado, backend/WireGuard indisponível, perda/rotação de chaves e recuperação de senha permanecem pendentes.
  - Falha ao gravar comprovante JSON do aceite não libera perfis nem QR; o usuário recebe erro e pode tentar novamente.
assumptions: []
owner_decisions:
  - "Três projetos na raiz: front-vpn/ em Flutter, back-vpn/ em Go e dashboard-admin-vpn/ em Next.js."
  - Já existe um serviço WireGuard no servidor do dono.
  - O backend lista e gerencia usuários; cada amigo recebe usuário e senha; o administrador acessa o dashboard por senha, cria contas e controla acesso.
  - O público-alvo inicial é um grupo de amigos e a solução deve ser simples.
  - O app é somente Android, suporta Android 7.0/API 24 ou superior e liga/desliga o túnel dentro do app por API nativa.
  - O amigo faz login no app Android; o backend entrega automaticamente os dados para configurá-lo sem manipulação manual de arquivo, QR ou parâmetros; opcionalmente, o amigo pode exibir por QR qualquer perfil, inclusive o principal, para importação no app oficial WireGuard em iPhone/iPad.
  - Na primeira utilização, o perfil principal é selecionado por padrão; selecionar não conecta, e iniciar conexão usa o último perfil selecionado, preservado entre sessões do app.
  - Somente o administrador cria contas. Cada conta começa com um perfil principal; somente o administrador pode adicionar perfis/slots extras, e o amigo logado vê todos os perfis disponíveis para sua conta.
  - Um slot é um perfil de acesso, não um aparelho físico. Slots diferentes têm peers e pares de chaves distintos; o QR de um slot importado em outro aparelho reutiliza o mesmo peer/chave. O uso desse mesmo perfil em aparelhos físicos é destinado a alternância, sem promessa de simultaneidade estável.
  - Ao exibir qualquer perfil ou QR, o app Android mostra aviso de que usar o mesmo perfil/chave simultaneamente em dois aparelhos pode causar instabilidade e lentidão ou perda de conexão.
  - Backend armazena perfis e chaves privadas criptografados para recuperação após reinstalação do app.
  - Antes de entregar perfis/QR, exigir Li e aceito e registrar comprovante JSON em diretório persistente e configurável fora do build. A minuta inicial dos termos está em docs/features/vpn-access/mvp/terms-of-use-draft.md.
  - Guardar metadados de conta/perfil, IP de origem e horário UTC por autenticação e chamada autenticada app/API e evento VPN observável, sem guardar histórico de sites, destinos ou conteúdo; fornecer registros existentes a autoridades somente nas hipóteses legais aplicáveis.
  - Bloquear uma conta nega imediatamente o acesso app/API e, na mesma operação lógica, tenta remover todos os seus peers da configuração WireGuard ativa e do estado persistente. O dashboard só informa conclusão após confirmação do servidor de remoção de todos os peers; sem confirmação, informa pendente/falha. Após remoção confirmada, túneis ativos e perfis importados via QR, inclusive em iPhone/iPad, deixam de ser aceitos. Tática, timeout e retry serão especificados após levantamento SSH, sem SLA numérico neste contrato.
  - "TDD escolhido explicitamente: yes."
  - "vpn.darcanix.com é o endereço exclusivo do endpoint VPN. O nome indicado para o dashboard na VPS é dashboard-vpn; o FQDN completo do painel e a hospedagem da API continuam pendentes."
  - O dono abrirá uma sessão SSH posteriormente; detalhes do servidor seguem pendentes e nenhuma integração live está autorizada.
  - App Flutter continua somente Android; não haverá app Flutter iOS. O QR de qualquer perfil, inclusive o principal, pode ser importado no app oficial WireGuard em iPhone/iPad; não foi aprovada versão de iOS nem suporte macOS.
  - Respostas anteriores de formulário foram invalidadas como cliques acidentais. Decisões diretas posteriores substituem o clique anterior sobre plataforma e estabelecem perfis/slots separados de aparelhos físicos. Permanecem pendentes limite e detalhes do fluxo administrativo dos slots, especificações de criptografia/recuperação, tática/timeout/retry da remoção no servidor e localização/OS do servidor.
  - Em 2026-09-29, o dono aprovou o contrato com inclusão do termo de uso e pediu início da orquestração; a autorização cobre código e testes locais, não ações externas ou de produção.
---

# Task Contract: VPN Access MVP

## Outcome

Oferecer a um grupo de amigos acesso simples ao serviço WireGuard existente. O administrador cria contas e gerencia perfis; amigos usam o app Flutter Android para conectar o último perfil selecionado e podem exportar qualquer perfil por QR para importação no app oficial WireGuard em iPhone/iPad. O bloqueio de uma conta impede imediatamente o acesso ao app/API e inicia a remoção dos peers daquela conta do WireGuard ativo e persistente.

## Scope

### In

- Criar na raiz `front-vpn/` (Flutter), `back-vpn/` (Go) e `dashboard-admin-vpn/` (Next.js).
- App Flutter somente Android, Android 7.0/API 24 ou superior, com controle do túnel dentro do app por API nativa.
- Amigo faz login com credenciais da conta; backend fornece automaticamente os dados necessários e o app configura sem copiar arquivo, escanear QR ou manipular parâmetros WireGuard.
- Na primeira utilização, o perfil principal é selecionado por padrão. Selecionar um perfil não conecta; ao iniciar conexão, o app usa o último selecionado, mantendo a escolha após sair e reabrir o app.
- Somente administrador cria contas. Cada conta começa com o perfil principal; somente administrador adiciona slots/perfis extras.
- Amigo logado vê o perfil principal e os slots extras de sua conta e pode exibir QR de qualquer perfil, inclusive o principal, para importar no app oficial WireGuard em iPhone/iPad. Não haverá app Flutter iOS.
- Cada slot é um perfil de acesso, não um aparelho físico. Slots diferentes têm peers e pares de chaves distintos. Um QR copiado reutiliza o mesmo peer e chave, inclusive do perfil principal; uso simultâneo do mesmo perfil em aparelhos pode causar instabilidade, lentidão ou perda de conexão, sem promessa de simultaneidade estável.
- Ao exibir QR ou perfil, app Android mostra aviso claro sobre os riscos de uso simultâneo do mesmo perfil/chave.
- Backend armazena perfis e chaves privadas criptografados para permitir recuperação após reinstalação. Só a conta autenticada proprietária recupera os próprios segredos; o serviço pode descriptografá-los para entrega. O fluxo administrativo não precisa exibir chaves privadas.
- Transporte app/API e dashboard/API autenticado e criptografado.
- Antes de entregar perfis ou QR, apresentar a versão inicial publicada dos [Termos de Uso](terms-of-use-draft.md) e exigir aceite eletrônico Li e aceito; guardar comprovante JSON persistente fora do build. Registrar IP de origem e horário UTC de autenticações, chamadas autenticadas app/API e eventos VPN observáveis, sem histórico de sites, destinos ou conteúdo.
- Bloquear conta nega imediatamente o acesso app/API e, na mesma operação lógica, tenta remover todos os peers da conta da configuração WireGuard ativa e do estado persistente. Dashboard só informa conclusão após confirmação do servidor de remoção de todos os peers; sem confirmação, mostra pendente/falha. Após confirmação, túneis ativos e QR importados deixam de ser aceitos pelo servidor.
- `vpn.darcanix.com` é exclusivo do endpoint VPN. `dashboard-vpn` é o nome indicado para o dashboard na VPS; FQDN completo e hospedagem da API ainda precisam ser confirmados.

### Out

- App Flutter para iOS, macOS ou plataforma além de Android 7.0/API 24 ou superior; também não se afirma suporte a qualquer versão específica de iOS/macOS para importação QR.
- Exigir QR, arquivo ou cópia de parâmetros como fluxo de configuração do app Android. A exportação QR para o app WireGuard no iPhone/iPad permanece opcional e inclui o perfil principal.
- Garantir uso simultâneo estável do mesmo perfil/peer/chave em aparelhos diferentes.
- Integração live com o servidor antes do levantamento em sessão SSH e das decisões técnicas subsequentes.
- Definir agora como gerar/armazenar localmente as chaves, limite de slots, autorização de aparelhos adicionais, política criptográfica específica, tática/timeout/retry de revogação, instalação de servidor, FQDN do painel, localização da API ou procedimento de deploy.
- Publicar em produção, alterar DNS ou operar credenciais reais sem autorização específica.
- Exigir novo aceite para futuras revisões dos termos. Um perfil já importado por QR no app WireGuard não passa pelo app/API para revalidar aceite.

## Observable behavior

- Administrador autentica-se no dashboard por senha e é o único que cria contas e adiciona slots extras.
- Amigo entra no app Android e recebe automaticamente seus perfis. Contas bloqueadas não acessam app/API. Usuário não cria slots, nem lê segredos ou QR de outra conta.
- Na primeira utilização, perfil principal selecionado; seleção não liga o túnel. Ao solicitar conexão, app usa o perfil selecionado mais recentemente e preserva a escolha entre sessões.
- Com permissão Android VpnService concedida, app pode iniciar o túnel. Se permissão é negada ou revogada, não informa conectado. UI reflete o estado real do túnel, inclusive desconexão observada.
- Amigo vê seus perfis e pode exibir QR de qualquer um, inclusive do principal; o QR pode ser importado no app oficial WireGuard em iPhone/iPad.
- QR importado reutiliza o mesmo peer/chave do perfil de origem. Slots distintos têm peers/pares de chaves distintos. Aviso claro aparece ao exibir qualquer QR/perfil e descreve instabilidade, lentidão ou perda de conexão possível quando o mesmo perfil/chave é usado simultaneamente.
- Perfis privados, inclusive chaves, são criptografados em repouso e recuperáveis após reinstalação. Somente a conta autenticada proprietária pode recuperar seus segredos; o serviço pode descriptografá-los para entrega. Se recuperação falha, o app informa a falha. O fluxo administrativo não precisa exibir chaves privadas.
- App/API e dashboard/API usam transporte autenticado e criptografado.
- Sem aceite dos termos da versão inicial publicada, perfis e QR não são entregues. Falha ao persistir o comprovante impede a liberação. A interface não promete anonimato absoluto nem incapacidade técnica do servidor de observar tráfego.
- Bloqueio nega imediatamente o acesso app/API e tenta, na mesma operação lógica, retirar todos os peers da configuração WireGuard ativa e persistente. Dashboard mostra conclusão somente após confirmação do servidor para todos os peers; até então mostra pendente/falha. Após sucesso, túnel ativo e perfil copiado por QR, inclusive usado no iPhone/iPad, deixam de ser aceitos.
- `vpn.darcanix.com` é apenas endpoint VPN. `dashboard-vpn` é o nome indicado para o painel na VPS; seu FQDN completo e hospedagem da API permanecem pendentes.

## Interfaces and data

- `front-vpn/`: Flutter Android; `back-vpn/`: backend Go; `dashboard-admin-vpn/`: dashboard Next.js. `dashboard-vpn` é o nome indicado do dashboard na VPS, sem FQDN completo confirmado.
- Senhas de conta/admin são credenciais de acesso, não chaves WireGuard.
- Cada conta inicia com perfil principal; slots extras são criados somente pelo admin. Slots distintos representam peers e pares de chaves distintos, não aparelhos físicos.
- O backend fornece os perfis à conta autenticada proprietária. QR contém `PrivateKey` e pode conter `PresharedKey`; é uma credencial exportável e segredo. QR copiado reutiliza o mesmo perfil/peer/chave.
- Dados privados de perfil, incluindo chaves, são criptografados em repouso. Somente a conta autenticada proprietária pode recuperar os próprios segredos; serviço pode descriptografar para entrega. O fluxo administrativo não precisa exibir chave privada. Falha de recuperação é visível.
- Perfis/segredos de outra conta, sessão bloqueada e criação de slot pelo usuário são recusados.
- App/API e dashboard/API comunicam-se por transporte autenticado e criptografado.
- O comprovante de aceite é JSON em diretório persistente e configurável, fora do build, contendo conta, versão/hash dos termos, horário UTC e identificador do evento. Metadados de acesso contêm conta/perfil, IP de origem e horário UTC; o serviço não guarda histórico de destinos ou conteúdo acessados.

## Constraints

- Plataforma mínima Android 7.0/API 24; controle nativo do túnel no app. Referências: [Android 7.0](https://developer.android.com/about/versions/nougat/android-7.0-changes) e [revisão do módulo Tunnel WireGuard Android com minSdk 24](https://git.zx2c4.com/wireguard-android/commit/tunnel/build.gradle.kts?follow=1&id=2a017642822ccf95f673d3520f07d103f654f5f4).
- Android VpnService pode exigir autorização do usuário; o app deve refletir permissão e estado efetivo do túnel.
- App/API e dashboard/API exigem transporte autenticado e criptografado. Dados privados de perfil são criptografados em repouso; somente a conta autenticada proprietária recupera seus segredos, com descriptografia pelo serviço para entrega. O fluxo administrativo não precisa exibir chaves privadas. Não se especificam cifra, KMS ou chave mestra neste contrato.
- QR WireGuard serializa `PrivateKey` e pode conter `PresharedKey`, portanto é credencial exportável ([wg(8)](https://git.zx2c4.com/wireguard-tools/about/src/man/wg.8)). O app oficial WireGuard para iPhone/iPad importa túnel por QR ([App Store](https://apps.apple.com/us/app/wireguard/id1441195209)); o scanner oficial lê a configuração no formato do túnel ([scanner iOS](https://git.zx2c4.com/wireguard-apple/tree/WireGuard/WireGuard/UI/iOS/QRScanViewController.swift?h=0.0.20181104-1&id=bede8a17ad292120e414051a8c833543e73197a0). Isso não define versão de sistema nem inclui Flutter iOS/macOS.
- Slots distintos usam keys/peers distintos; copiar QR reutiliza o mesmo peer/chave. Inferência operacional baseada no endpoint roaming do WireGuard: o mesmo perfil simultâneo em dois aparelhos pode alternar o endpoint do peer, causando instabilidade e lentidão ou perda de conexão, não apenas menor velocidade ([overview](https://www.wireguard.com/#conceptual-overview), [wg(8)](https://git.zx2c4.com/wireguard-tools/about/src/man/wg.8)).
- Regra de bloqueio decidida: negar app/API imediatamente e tentar remover todos os peers da conta do WireGuard ativo e persistente na mesma operação lógica; sucesso no dashboard apenas com confirmação do servidor para todos, caso contrário pendente/falha. Ao concluir, peers deixam de ser aceitos, inclusive em túneis ativos e QR copiados. Tática, timeout, retry e propagação técnica ficam para Code Plan após SSH; sem SLA numérico aqui.
- O serviço WireGuard existente é o alvo, mas o dono abrirá SSH depois; não há integração live autorizada agora. `vpn.darcanix.com` é endpoint VPN. `dashboard-vpn` é nome indicado do dashboard na VPS; FQDN completo e hospedagem da API não estão confirmados.
- Não inferir limite de slots, autenticação adicional, política local de armazenamento/geração de chave, processo de rotação ou detecção de uso simultâneo.
- A primeira versão dos termos requer aceite antes da entrega de perfis/QR. O texto do usuário ainda exige responsável, contato e prazo de retenção antes da publicação; nova versão e reaceite futuro estão fora do MVP aprovado. Logs de app/API, WireGuard, firewall e DNS devem ser confrontados com a afirmação de ausência de histórico de destinos/conteúdo antes de publicar.

## Acceptance criteria

Os cenários abaixo são ATDD **aprovada pelo dono em 2026-09-29**, ainda não executada.

- [ ] Administrador autenticado por senha acessa o dashboard e cria contas; usuário comum não cria conta.
- [ ] Conta inicia com perfil principal; admin pode criar perfis extras; usuário só vê seus perfis e não consegue criar slot nem obter segredo/QR de outra conta.
- [ ] No primeiro uso, principal é selecionado. Selecionar um perfil não inicia conexão. Ao solicitar conexão, app conecta o último perfil selecionado; escolha persiste após encerrar e reabrir app.
- [ ] Com permissão VpnService concedida no Android 7/API 24+, iniciar conexão inicia o túnel do perfil selecionado, e estado mostrado acompanha o estado efetivo do túnel/desconexão.
- [ ] Se permissão VpnService é negada ou revogada, app não declara conexão ativa e informa que não está conectado.
- [ ] Sessão bloqueada é recusada para acesso app/API.
- [ ] Amigo vê e pode exibir QR de qualquer perfil próprio, inclusive principal com apenas um slot; QR é importável no app oficial WireGuard em iPhone/iPad. Isso não cria app Flutter iOS nem promete versão específica de sistema.
- [ ] Slots diferentes têm pares de chaves e peers distintos; copiar/importar um QR mantém o mesmo peer/chave, destinado a uso alternado sem promessa de simultaneidade estável.
- [ ] Ao exibir QR ou perfil, app mostra aviso de que uso simultâneo da mesma chave/perfil em dois aparelhos pode causar instabilidade, lentidão ou perda de conexão.
- [ ] Perfis privados, inclusive chaves, são armazenados criptografados e podem ser recuperados após reinstalação; se recuperação falha, usuário recebe erro visível. Somente a conta autenticada proprietária recupera seus segredos; serviço pode descriptografar para entrega; fluxo administrativo não precisa exibir chaves privadas.
- [ ] App/API e dashboard/API usam transporte autenticado e criptografado.
- [ ] Ao bloquear conta, app/API negam acesso imediatamente e a mesma operação lógica tenta remover todos os peers da configuração WireGuard ativa e persistente.
- [ ] Dashboard informa sucesso do bloqueio somente depois que servidor confirma remoção de todos os peers; se confirmação faltar ou houver falha, exibe pendente/falha, nunca sucesso.
- [ ] Depois de confirmada a remoção, peers da conta deixam de ser aceitos pelo servidor, inclusive túnel já ativo e perfil importado via QR no WireGuard de iPhone/iPad.
- [ ] `vpn.darcanix.com` é usado exclusivamente como endpoint VPN; `dashboard-vpn` é o nome indicado para o dashboard na VPS, com FQDN completo e hospedagem de API a confirmar.
- [ ] Sem Li e aceito para a versão inicial publicada dos termos, backend não entrega perfis privados nem QR. Após aceite, persiste JSON com conta, versão/hash do texto, horário UTC e identificador do evento; falha de gravação não libera acesso.
- [ ] Backend registra conta/perfil, IP de origem e horário UTC em autenticações e chamadas autenticadas app/API e nos eventos VPN observáveis. Não guarda histórico de sites, destinos ou conteúdo em seus componentes, inclusive logs de infraestrutura sob seu controle.

## TDD selection

- TDD: `yes`, escolha explícita do dono.
- A escolha foi preservada na aprovação do contrato; os ciclos locais serão test-first.

## Validation boundary

O contrato e os critérios foram aprovados pelo dono nesta conversa; planejamento, implementação e testes locais estão autorizados. Nenhum teste, build, integração live ou acesso ao servidor WireGuard foi executado até esta aprovação. O dono abrirá uma sessão SSH depois para levantamento, o que não autoriza por si só integração live. DNS, deploy, produção e uso de dados reais exigem autorização específica. Antes de publicar os termos, preencher responsável, contato e prazo de retenção e verificar o comportamento real dos logs.

## Authority required

- Autorização específica para SSH, integração live WireGuard, alteração de DNS/TLS e publicação em produção.
- A aprovação autoriza código e testes locais com TDD `yes`; não autoriza SSH, integração live ou produção.

## Failure and recovery

Falha ao recuperar perfil criptografado deve ser apresentada claramente ao usuário. Bloqueio de conta nega imediatamente app/API; se o servidor não confirmar a remoção de todos os peers, dashboard mantém estado pendente ou exibe falha, sem sucesso falso. Prazo numérico, timeout e retry não foram definidos. Regras de credencial inválida, perda/rotação de chave, indisponibilidade de backend/WireGuard, slots duplicados e recuperação de senha também precisam ser especificadas antes da implementação correspondente.

Falha ao persistir o comprovante JSON do aceite impede a entrega de perfil/QR e gera erro visível com opção de nova tentativa.

## Assumptions

Nenhuma suposição material adotada. Decisões ainda ausentes estão listadas abaixo.

## Owner decisions

- Três projetos na raiz: `front-vpn/` em Flutter, `back-vpn/` em Go e `dashboard-admin-vpn/` em Next.js.
- Existe serviço WireGuard no servidor do dono; o público inicial é um grupo de amigos e a solução deve ser simples.
- Administrador acessa dashboard por senha e é o único que cria contas e perfis extras; cada amigo faz login no app e recebe dados de conexão automaticamente.
- App Flutter é somente Android, mínimo Android 7.0/API 24, controla o túnel no app por API nativa.
- Primeiro uso seleciona o perfil principal; seleção não conecta; ao iniciar conexão, app usa último perfil selecionado e persiste a escolha entre sessões.
- Cada conta inicia com perfil principal; admin adiciona quaisquer extras; amigo autenticado vê seus perfis e pode mostrar QR de qualquer um, inclusive principal, para importar no app oficial WireGuard em iPhone/iPad. Não haverá app Flutter iOS.
- Slots são perfis, não dispositivos: slots diferentes têm peers/chaves diferentes; copiar QR reutiliza o mesmo peer/chave e é para uso alternado. App avisa claramente que simultaneidade pode causar instabilidade, lentidão ou perda de conexão.
- Backend armazena perfis e chaves privadas criptografados para recuperação pós-reinstalação; conta autenticada proprietária recupera apenas os próprios segredos e o serviço pode descriptografá-los para entrega. O fluxo administrativo não precisa exibir chaves privadas.
- TDD `yes` escolhido explicitamente.
- Bloqueio nega app/API imediatamente e tenta remover peers de configuração WireGuard ativa e persistente na mesma operação lógica. Dashboard só sinaliza conclusão após confirmação do servidor para todos; sem confirmação, pendente/falha. Após sucesso, perfis deixam de ser aceitos, inclusive cópias QR em clientes WireGuard de iPhone/iPad. Tática/timeout/retry ficam por especificar após SSH; nenhum SLA numérico decidido.
- `vpn.darcanix.com` é exclusivo do endpoint VPN. `dashboard-vpn` é o nome indicado para dashboard na VPS; FQDN completo e hospedagem de API pendentes.
- O dono abrirá sessão SSH depois; localização/OS do servidor seguem pendentes e nenhuma integração live está autorizada.
- Respostas anteriores de formulário foram invalidadas como cliques acidentais; decisões diretas posteriores acima têm precedência.

## Decisões pendentes

1. **Chaves e recuperação técnica:** geração, mecanismo de criptografia, credenciais operacionais e processo de recuperação/rotação ainda não especificados; não há decisão de cifra, KMS ou chave mestra.
2. **Slots:** limite de perfis extras e fluxo operacional no dashboard, mantendo a regra confirmada de criação apenas pelo administrador.
3. **Propagação técnica do bloqueio:** após acesso app/API ser negado imediatamente, definir tática, timeout/retry e como confirmar remoção de todos os peers após o levantamento SSH. Sem SLA numérico decidido; sem confirmação, dashboard permanece pendente/falha.
4. **Servidor WireGuard:** OS, interface de controle, fonte de verdade dos peers e alterações permitidas serão levantados em sessão SSH futura.
5. **Hosts:** `vpn.darcanix.com` fica exclusivo do endpoint VPN. `dashboard-vpn` aponta pelo nome para dashboard na VPS, mas falta confirmar FQDN completo e se API será hospedada nela.
6. **Deploy/TLS:** definir publicação e gestão DNS/certificados para os hosts aprovados.
7. **Termos antes da publicação:** preencher identidade/contato do responsável e prazo de retenção, além de conferir que logs de app/API, WireGuard, firewall e DNS correspondem ao texto aprovado. A aprovação atual cobre o aceite da primeira versão; uma política de reaceite após revisões futuras exigirá decisão própria.

## Amendments

```yaml
amendment:
  id: terms-and-access-metadata-001
  resulting_revision: 2
  reason: Owner added electronic acceptance of initial terms, JSON proof, access metadata, and no stored browsing history before approving the complete contract.
  evidence:
    - Owner's direct approval and instruction to begin orchestration in this chat on 2026-09-29.
    - docs/features/vpn-access/mvp/terms-of-use-draft.md
  changed_clauses:
    - scope, observable behavior, interfaces/data, constraints, acceptance criteria, validation and authority boundaries
  preserved_clauses:
    - Existing Android, account/profile, QR, encryption, revocation, endpoint, and TDD decisions.
  owner_approved_at: 2026-09-29T03:10:24Z # source-registration timestamp
```
