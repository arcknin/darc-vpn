---
schema_version: 1
flow: orchestration-flow
feature: vpn-access
task: mvp
plan_revision: 3
contract_revision: 2
contract_sha256: 477a54dadf1fd064c5b4e2fe8c355b8d51ac157ca33537e7f8393cfb6972b683
status: frozen
tdd: "yes"
implementation_authorized_at: 2026-09-29T03:10:24Z
repositories:
  - front-vpn/
  - back-vpn/
  - dashboard-admin-vpn/
---

# Task Code Plan: vpn-access MVP

## Execution summary

Construir três projetos independentes, sem inicializar Git: `back-vpn/` será a fonte do contrato HTTP, da autorização e do estado; `dashboard-admin-vpn/` consumirá apenas operações administrativas sem segredos de perfil; `front-vpn/` será o cliente Android e manterá a integração nativa do túnel atrás de uma ponte testável. O backend usará persistência SQLite migrada, perfis WireGuard cifrados por registro, senhas com hash resistente, sessões opacas e adaptadores explícitos para arquivos de aceite, relógio, identificadores, IP de origem e controle WireGuard.

Os lotes B01-B04 constroem a API e os invariantes de segurança. B05 e B06 podem avançar em paralelo quando seus contratos dependentes estiverem verdes; B07 acrescenta o túnel Android; B08 fecha apenas a evidência local dos três projetos. Cada lote executa ciclos `expected-red -> implementação mínima -> relevant-green -> refactor mantendo verde`. Nenhum lote inclui publicação, infraestrutura externa ou integração live.

## Architecture and ownership

- `back-vpn/` é dono do OpenAPI versionado, modelo de autorização, estado de conta/perfil, geração de pares WireGuard, criptografia em repouso, comprovantes de aceite, metadados de acesso e máquina de estados de bloqueio/revogação.
- `dashboard-admin-vpn/` é dono da sessão e experiência administrativa. Seus modelos de resposta excluem material privado de perfis por construção.
- `front-vpn/` é dono da sessão do amigo, aceite, seleção persistida, revelação de perfil/QR e estado observado do túnel Android. Somente o identificador do perfil selecionado pode ir para armazenamento comum; token fica em armazenamento seguro e configuração/QR secreta não vai para logs, analytics ou preferências comuns.
- O OpenAPI em `back-vpn/` é o contrato entre projetos. Clientes gerados ou adaptadores tipados nos dois frontends devem falhar em CI local se divergirem do documento.
- O controlador WireGuard do backend é uma porta. Nesta revisão, a implementação executável de desenvolvimento é determinística e simulada; ela permite provar transições e falhas sem alegar efeito externo.
- A ponte Android expõe `preparePermission`, `start`, `stop` e um fluxo de estado efetivo (`disconnected`, `connecting`, `connected`, `disconnecting`, `failed`). A UI deriva desse fluxo e nunca do retorno de um botão.

## Tasks

```yaml
- id: B01
  goal: "Criar a fundação do backend e congelar o contrato HTTP local do MVP."
  repository: back-vpn/
  depends_on: []
  allowed_scope:
    - "back-vpn/go.mod"
    - "back-vpn/go.sum"
    - "back-vpn/cmd/api/**"
    - "back-vpn/internal/platform/**"
    - "back-vpn/internal/domain/**"
    - "back-vpn/api/openapi.yaml"
    - "back-vpn/test/contract/**"
    - "back-vpn/README.md"
  changes:
    - "Inicializar um módulo Go sem criar repositório Git; separar domínio, aplicação, portas e adaptadores HTTP/persistência."
    - "Definir no OpenAPI autenticação de amigo e administrador, consulta/aceite dos termos, listagem e recuperação de perfis próprios, criação administrativa de conta e perfil extra, bloqueio e consulta do estado de revogação."
    - "Definir erros estáveis e sem enumeração de conta: credencial inválida, não autenticado, proibido, conta bloqueada, aceite pendente, perfil ausente, recuperação impossível, persistência de aceite falhou e revogação não confirmada."
    - "Separar DTO público de perfil, DTO privado entregue apenas ao proprietário e DTO administrativo sem chave privada, preshared key ou configuração exportável."
    - "Criar portas para armazenamento transacional, cifragem, hash de senha, sessões, relógio, IDs, arquivos de aceite, metadados de acesso, geração de chaves e controle WireGuard; fornecer somente fakes locais onde o ambiente externo seria necessário."
    - "Reservar `vpn.darcanix.com` exclusivamente no campo Endpoint dos perfis; não definir host do dashboard ou da API."
  validation:
    - "Expected-red: testes de contrato falham para rotas, esquemas, ownership e estados ainda ausentes."
    - "Relevant-green: parser/linter do OpenAPI e testes Go confirmam esquemas distintos para amigo/admin, ausência de segredos nos modelos administrativos e endpoint VPN fixo."
    - "Executar `go test ./...`, `go vet ./...` e `gofmt` sem diferenças."
  done_when:
    - "Módulo compila, OpenAPI é válido e todas as operações aprovadas têm pedido, resposta, erro e regra de autorização representados."
    - "Nenhuma decisão de host externo, política futura de reaceite ou limite de slots foi introduzida."
  parallelizable: false

- id: B02
  goal: "Entregar identidade, contas e perfis recuperáveis com isolamento e criptografia em repouso."
  repository: back-vpn/
  depends_on: [B01]
  allowed_scope:
    - "back-vpn/go.mod"
    - "back-vpn/go.sum"
    - "back-vpn/internal/auth/**"
    - "back-vpn/internal/accounts/**"
    - "back-vpn/internal/profiles/**"
    - "back-vpn/internal/storage/**"
    - "back-vpn/internal/crypto/**"
    - "back-vpn/internal/http/**"
    - "back-vpn/migrations/**"
    - "back-vpn/test/**"
  changes:
    - "Implementar SQLite com migrações para administradores, contas, sessões, perfis e estado de aceite/revogação; IDs são opacos e todas as consultas de amigo incluem `account_id`."
    - "Criar senha administrativa e de amigo com Argon2id, comparação em tempo constante e mensagens genéricas; tokens de sessão aleatórios ficam persistidos somente como digest e têm papel/conta explícitos."
    - "Revalidar status da conta em toda requisição autenticada, sem depender apenas de claims em token, para que o bloqueio corte app/API imediatamente."
    - "Ao criar uma conta, gerar exatamente um perfil principal. Perfis extras só podem ser criados por administrador e não há limite de aplicação inventado."
    - "Gerar um par de chaves independente por perfil no backend com primitiva WireGuard mantida; nunca reutilizar chaves entre slots."
    - "Cifrar a configuração privada de cada perfil com AES-256-GCM, nonce aleatório por gravação e associação autenticada a conta/perfil/versão; obter a chave mestra por configuração injetada e não persistir seu valor no repositório."
    - "Falhar fechado em chave mestra ausente, tag inválida, versão desconhecida ou corrupção; o erro de recuperação chega ao proprietário sem vazar material ou detalhe criptográfico."
    - "Entregar configuração privada somente ao proprietário autenticado e somente por endpoint sujeito ao gate de aceite que B03 completará; o administrador recebe metadados públicos."
  validation:
    - "Expected-red: testes falham para login, papel, criação do principal, extras distintos, acesso cruzado, sessão bloqueada, ciphertext e recuperação corrompida."
    - "Relevant-green: testes de unidade e integração SQLite provam principal automático, extras admin-only, chaves públicas distintas, nenhum segredo em linhas/DTOs administrativos e round-trip cifrado apenas com chave correta."
    - "Testes negativos provam recusa de criação pelo amigo, consulta de outra conta, token inexistente, papel errado e conta bloqueada."
    - "Executar `go test -race ./...`, `go vet ./...` e migração up em banco temporário novo."
  done_when:
    - "Conta e perfis sobrevivem a reinício do processo local e podem ser recuperados após novo login sem texto privado no banco."
    - "Nenhum endpoint administrativo serializa PrivateKey, PresharedKey ou configuração WireGuard completa."
  parallelizable: false

- id: B03
  goal: "Aplicar o aceite inicial e registrar apenas os metadados de acesso aprovados."
  repository: back-vpn/
  depends_on: [B02]
  allowed_scope:
    - "back-vpn/cmd/api/**"
    - "back-vpn/internal/terms/**"
    - "back-vpn/internal/audit/**"
    - "back-vpn/internal/http/**"
    - "back-vpn/internal/storage/**"
    - "back-vpn/test/**"
    - "back-vpn/testdata/terms/**"
  changes:
    - "Substituir no composition root `cmd/api` o handler reservado de B01 por `internal/http.New`, injetando a persistência, autenticação, contas e perfis de B02 junto dos serviços de termos e auditoria de B03; `internal/platform/httpapi` não precisa ser alterado."
    - "Carregar versão inicial, bytes do texto e hash esperado por configuração. Não usar a minuta incompleta como texto publicável; fixtures completas e fictícias servem apenas aos testes locais."
    - "Exigir aceite antes de qualquer resposta com configuração privada ou capacidade de QR; listagem/recuperação protegida retorna `terms_acceptance_required` até a persistência bem-sucedida."
    - "Persistir comprovante JSON com account_id, versão, SHA-256 exato dos bytes apresentados, timestamp UTC e event_id em diretório configurável fora do build, usando criação atômica, permissões restritas e nomes derivados de ID seguro."
    - "Só registrar aceite no estado consultável depois da gravação do JSON; falha de diretório, escrita, rename ou sincronização mantém o gate fechado e permite nova tentativa sem sucesso falso."
    - "Implementar log estruturado mínimo com event_id, tipo, account_id quando conhecido, profile_id quando aplicável, source_ip e timestamp UTC para autenticações e chamadas autenticadas."
    - "Resolver IP pelo endereço remoto por padrão; cabeçalhos de proxy não são confiados sem configuração explícita. Proibir corpos, configuração WireGuard, tokens, termos completos, destinos e conteúdo em logs."
    - "Receber eventos VPN observáveis somente pela porta autenticada do controlador, com o mesmo conjunto mínimo de campos; o fake local permite testar o formato sem produzir alegação de observação live."
  validation:
    - "Expected-red: o executável local ainda responde pelo handler reservado ou os cenários ATDD falham para login, entrega antes do aceite, hash/versão divergentes, falha de arquivo e campos proibidos em logs."
    - "Relevant-green: testes com diretório temporário provam JSON parseável, conteúdo exato, atomicidade, permissões restritas e gate liberado somente após sucesso."
    - "Testes de captura de logs cobrem login, chamada autenticada e evento VPN simulado; procuram e recusam PrivateKey, PresharedKey, Authorization, corpos, host/destino e payload de conteúdo."
    - "Iniciar `cmd/api` em loopback com SQLite, termos, chave e diretório temporários; testes HTTP black-box confirmam rotas reais de login, conta/perfis, leitura/aceite dos termos e recuperação protegida, sem resposta 501 do handler reservado."
    - "Executar `go test -race ./...` e `go vet ./...`."
  done_when:
    - "Sem aceite inicial válido, nenhuma configuração privada pode sair da API; falha de comprovante é visível e retentável."
    - "A implementação local produz somente os metadados aprovados e tem testes que impedem regressão de segredo/conteúdo."
    - "O binário `cmd/api` compõe e serve por HTTP local os fluxos B02+B03 reais; sucesso não depende de invocar `internal/http` apenas em testes de pacote."
  parallelizable: false

- id: B04
  goal: "Entregar bloqueio imediato e revogação confirmável como máquina de estados local."
  repository: back-vpn/
  depends_on: [B03]
  allowed_scope:
    - "back-vpn/cmd/api/**"
    - "back-vpn/internal/accounts/**"
    - "back-vpn/internal/revocation/**"
    - "back-vpn/internal/wireguard/**"
    - "back-vpn/internal/http/**"
    - "back-vpn/internal/storage/**"
    - "back-vpn/test/**"
  changes:
    - "Estender o composition root `cmd/api` entregue por B03 para injetar serviço, armazenamento e controlador fake de revogação nas rotas de bloqueio e consulta de estado de `internal/http`, sem reintroduzir o handler reservado."
    - "Na transação de bloqueio, marcar a conta como bloqueada e invalidar suas sessões antes de chamar qualquer adaptador; pedidos subsequentes app/API falham imediatamente."
    - "Registrar uma operação durável com snapshot de todos os profile_ids/public keys da conta e estado por peer. A mesma operação lógica faz uma tentativa imediata pela porta WireGuard após o commit de bloqueio."
    - "Modelar `pending`, `failed` e `completed`; `completed` exige confirmação explícita de remoção ativa e persistente para todos os peers do snapshot. Ausência, resposta parcial ou erro nunca vira sucesso."
    - "Tornar o comando idempotente para não reativar conta nem perder o conjunto de peers em repetição de pedido/processo; não definir SLA, timeout numérico ou política automática de retries."
    - "Fornecer fake determinístico capaz de confirmar todos, falhar, omitir confirmação e responder parcialmente. Não criar adaptador de ambiente live nesta revisão."
    - "Expor ao dashboard somente conta bloqueada, estado agregado, resultado por perfil e erro sanitizado; chaves privadas continuam ausentes."
  validation:
    - "Expected-red: testes pelo executável local falham enquanto as rotas reais de bloqueio/estado não estiverem compostas, além dos cenários de corte imediato, confirmação parcial, erro, repetição e conclusão falsa."
    - "Relevant-green: integração SQLite + fake prova que bloqueio persiste antes da tentativa, todas as sessões são recusadas, todos os peers são enviados e somente confirmação dupla de cada peer conclui."
    - "Teste concorrente prova que duas solicitações não criam operações conflitantes nem deixam sessão utilizável."
    - "Iniciar `cmd/api` em loopback com dependências temporárias e controlador fake; testes HTTP black-box bloqueiam uma conta, comprovam recusa imediata da sessão e consultam `pending`, `failed` e `completed` sem sucesso falso."
    - "Executar `go test -race ./...` e `go vet ./...`."
  done_when:
    - "O estado local nunca apresenta bloqueio concluído sem confirmação de todos os peers e mantém pendente/falha de forma consultável."
    - "A limitação de prova local está explícita: confirmação do fake valida o protocolo e a máquina de estados, não o efeito externo."
    - "O binário `cmd/api` expõe as rotas reais de bloqueio e estado com a composição B04; testes de pacote isolados não são a única evidência."
  parallelizable: false

- id: B05
  goal: "Entregar o dashboard administrativo local sem exposição de segredos."
  repository: dashboard-admin-vpn/
  depends_on: [B01, B02, B04]
  allowed_scope:
    - "dashboard-admin-vpn/package.json"
    - "dashboard-admin-vpn/package-lock.json"
    - "dashboard-admin-vpn/next.config.*"
    - "dashboard-admin-vpn/src/**"
    - "dashboard-admin-vpn/app/**"
    - "dashboard-admin-vpn/tests/**"
    - "dashboard-admin-vpn/README.md"
  changes:
    - "Inicializar Next.js/TypeScript sem criar repositório Git e gerar/validar tipos a partir do OpenAPI B01."
    - "Implementar login por senha, sessão administrativa em cookie HttpOnly/Secure/SameSite, logout e guardas de rota; não persistir token em localStorage."
    - "Implementar criação/listagem de contas, visualização do perfil principal, criação de perfis extras sem limite inventado e ação de bloqueio."
    - "Exibir `pending`, `failed` e `completed` exatamente conforme a API; nunca converter resposta parcial, timeout de UI ou erro de rede em conclusão."
    - "Não renderizar, solicitar, armazenar ou registrar configuração privada, PrivateKey, PresharedKey ou QR."
    - "Exigir URL HTTPS fora do modo local de teste/desenvolvimento e não assumir FQDN do painel ou localização da API."
  validation:
    - "Expected-red: testes falham para acesso anônimo, usuário comum, criação de conta/principal, slot extra e três estados de bloqueio."
    - "Relevant-green: testes de componente e Playwright com API contratual provam fluxos admin-only, sessão segura, estados fiéis e ausência de campos secretos no DOM, storage e logs."
    - "Validar cliente contra `back-vpn/api/openapi.yaml`; executar lint, typecheck, testes e build de produção local."
  done_when:
    - "Administrador executa todas as operações aprovadas e o dashboard jamais mostra conclusão prematura ou material privado."
    - "Build não contém hostname externo presumido nem credencial."
  parallelizable: true

- id: B06
  goal: "Entregar no Flutter os fluxos de login, aceite, perfis, seleção e QR."
  repository: front-vpn/
  depends_on: [B01, B02, B03]
  allowed_scope:
    - "front-vpn/pubspec.yaml"
    - "front-vpn/pubspec.lock"
    - "front-vpn/lib/**"
    - "front-vpn/android/**"
    - "front-vpn/test/**"
    - "front-vpn/integration_test/**"
    - "front-vpn/README.md"
  changes:
    - "Inicializar Flutter apenas para Android com minSdk 24, sem criar repositório Git, e validar o cliente tipado contra o OpenAPI B01."
    - "Implementar login do amigo, token em armazenamento seguro, logout e tratamento explícito de credencial inválida, conta bloqueada e indisponibilidade."
    - "Mostrar exatamente a versão/texto/hash informados pela API, exigir ação `Li e aceito` e não buscar/renderizar configuração privada antes da confirmação persistida; falha permite nova tentativa."
    - "Listar somente perfis da conta. No primeiro uso selecionar o principal; persistir apenas profile_id; mudar seleção não chama a ponte de conexão. Se o ID salvo não existir mais, voltar ao principal sem conectar."
    - "Recuperar o segredo do perfil sob demanda e falhar visivelmente quando a API não puder recuperá-lo. Não registrar ou persistir configuração privada em preferências comuns."
    - "Antes de revelar configuração ou QR, mostrar aviso de que uso simultâneo do mesmo perfil/chave pode causar instabilidade, lentidão ou perda de conexão e que o uso previsto é alternado."
    - "Gerar QR localmente a partir da mesma configuração entregue, sem criar outro peer/chave; limpar referências em memória ao fechar a tela na medida permitida pelo runtime."
    - "Restringir base URL a HTTPS fora de testes/desenvolvimento e desabilitar cleartext no manifesto Android de release."
  validation:
    - "Expected-red: widget/unit tests falham para gate de aceite, principal padrão, seleção sem conexão, persistência, fallback, QR principal/extras e aviso obrigatório."
    - "Relevant-green: testes com fake API provam que nenhum segredo é pedido antes do aceite, seleção não conecta e QR round-trip preserva o mesmo perfil/peer/chave."
    - "Teste de segurança procura PrivateKey/PresharedKey/token em logs e SharedPreferences; teste contratual valida modelos contra OpenAPI."
    - "Executar `dart format --output=none --set-exit-if-changed .`, `flutter analyze`, `flutter test` e `flutter build apk --debug`."
  done_when:
    - "Fluxos aprovados funcionam em teste Android API 24+ e não existe artefato iOS/macOS."
    - "QR de qualquer perfil próprio é produzido com aviso e sem gerar nova identidade WireGuard."
  parallelizable: true

- id: B07
  goal: "Integrar a ponte Android ao backend de túnel WireGuard e refletir estado efetivo."
  repository: front-vpn/
  depends_on: [B06]
  allowed_scope:
    - "front-vpn/lib/features/tunnel/**"
    - "front-vpn/android/app/**"
    - "front-vpn/android/**"
    - "front-vpn/test/**"
    - "front-vpn/integration_test/**"
  changes:
    - "Integrar biblioteca WireGuard Android mantida por uma ponte nativa isolada; usar VpnService.prepare para obter permissão e alimentar o backend com a configuração do perfil selecionado."
    - "Solicitar conexão somente por ação explícita. Negação/cancelamento/revogação de permissão mantém estado desconectado e mostra mensagem clara."
    - "Mapear callbacks/consulta do backend de túnel ao fluxo de estado efetivo; pedido aceito não equivale a conectado e desconexão observada atualiza imediatamente a UI."
    - "Ligar usa o último profile_id selecionado; desligar afeta o túnel controlado pelo app. Trocar seleção durante túnel ativo não inicia nem migra automaticamente."
    - "Evitar configuração/segredos em logs da ponte, mensagens de erro e estado serializado."
  validation:
    - "Expected-red: testes falham para permissão negada/revogada, start sem seleção, callback de falha, desconexão externa e seleção que conecta sozinha."
    - "Relevant-green: unit/widget tests com ponte fake e testes instrumentados Android provam transições baseadas em callback, principal/última seleção e ausência de estado conectado falso."
    - "Build e teste instrumentado rodam em API 24 e em uma API Android atual; análise estática e testes Flutter permanecem verdes."
    - "Sem endpoint live, validar somente preparação, passagem de configuração, callbacks e máquina de estados; não registrar handshake como evidência."
  done_when:
    - "A UI mostra conectado apenas após confirmação da ponte e volta a desconectado quando permissão/estado efetivo assim indicar."
    - "Selecionar perfil continua sem efeito colateral de conexão."
  parallelizable: true

- id: B08
  goal: "Fechar a evidência local integrada e a matriz ATDD sem extrapolar para ambiente live."
  repository: back-vpn/
  depends_on: [B03, B04, B05, B07]
  allowed_scope:
    - "back-vpn/test/acceptance/**"
    - "back-vpn/testdata/**"
    - "back-vpn/scripts/local-acceptance/**"
  changes:
    - "Criar harness descartável com SQLite temporário, diretório temporário de comprovantes, chave de teste, termos fictícios completos e controlador WireGuard fake."
    - "Executar cenários API completos: admin cria conta/principal/extra, amigo aceita termos, recupera só seus perfis, bloqueio corta sessão e cada resposta fake produz pending/failed/completed corretamente."
    - "Publicar fixtures de resposta derivadas do OpenAPI para os testes dos dois clientes e rejeitar drift de schema."
    - "Validar payload WireGuard/QR por parse e round-trip local, preservando chave/peer e `vpn.darcanix.com`, sem alegar importação ou conexão live."
    - "Gerar matriz de evidência local que mapeia cada critério do contrato a teste/comando e marca separadamente o que exige prova de publicação."
  validation:
    - "Expected-red: o harness falha antes de conectar todos os fluxos e ao detectar drift entre OpenAPI e clientes."
    - "Relevant-green: `go test -race ./...`, `go vet ./...`, suíte/lint/typecheck/build Next.js, `flutter analyze`, `flutter test` e APK debug passam na mesma revisão dos três diretórios."
    - "Executar Playwright contra API local descartável para o fluxo administrativo; executar integração Flutter contra API local/fake contratual e ponte de túnel fake."
    - "Inspecionar artefatos e logs de teste para ausência de senha, token, PrivateKey, PresharedKey, destinos e conteúdo."
  done_when:
    - "Todos os critérios comprováveis localmente estão verdes e têm evidência reproduzível; resultados simulados são rotulados como simulados."
    - "A matriz mantém abertas as provas live e o gate de publicação, sem convertê-los em sucesso local."
  parallelizable: false
```

## Dependency order and parallel workstreams

1. B01 congela a fundação e o contrato HTTP.
2. B02 implementa identidade e perfis.
3. B03 fecha aceite e auditoria; depois dele, B04 e B06 podem avançar em paralelo em repositórios distintos.
4. B05 inicia depois de B04 e pode coincidir com B06.
5. B07 continua o app Android depois de B06 e pode coincidir com B05.
6. B08 só começa quando backend, dashboard e app estão verdes em seus próprios limites.

## Assumptions

- Os três diretórios ainda não existem e serão inicializados como projetos independentes, sem `git init`, submódulos ou monorepo implícito.
- SQLite é suficiente para o grupo inicial e fica atrás de interfaces transacionais; trocar o mecanismo depois não deve alterar comportamento observável.
- AES-256-GCM por registro, Argon2id para senhas e tokens opacos aleatórios são táticas ordinárias que preservam o contrato. Parâmetros ficam centralizados, versionados e cobertos por testes; valores secretos nunca entram na fonte.
- A configuração privada nasce no backend; o app recebe o perfil existente e o QR serializa exatamente esse perfil. O dashboard nunca precisa do segredo.
- A minuta de termos é insumo editorial, não artefato publicável. Testes usam texto fictício completo; produção permanece fechada enquanto houver campos editoriais pendentes.
- Não há limite funcional de slots no contrato; a implementação não cria um número arbitrário.
- A máquina de revogação e seus fakes são entregáveis locais. Um adaptador live só pode ser planejado após existir evidência do ambiente e autoridade separada.
- Comunicação criptografada é garantida localmente por configuração e testes que recusam HTTP fora de modo local. A evidência de transporte real pertence ao gate de publicação.

## TDD execution

- Cada lote começa pelo menor teste observável que falha pela ausência do comportamento; a falha deve ser registrada antes da implementação.
- A implementação mínima deve tornar verdes apenas os cenários do lote; refatoração ocorre com a suíte relevante verde.
- Testes unitários cobrem regras puras; integração usa SQLite e filesystem temporários; testes de contrato cobrem OpenAPI/clientes; testes de UI cobrem estados; instrumentação Android cobre a ponte nativa. Fakes jamais são apresentados como prova de efeito externo.
- Nenhum lote separado de “escrever testes” ou aprovação intermediária será criado. Uma emenda ao contrato invalida este plano inteiro.

## Publication gates

- Preencher responsável, contato, versão, vigência e retenção no texto final; calcular o hash dos bytes exatos publicados e configurar o backend com essa versão/hash. Enquanto qualquer campo estiver pendente, não expor os termos nem liberar perfis.
- Verificar no ambiente de publicação os logs de app, API, WireGuard, sistema, firewall e DNS contra a promessa de não manter histórico de sites, destinos ou conteúdo; corrigir divergências antes de publicar.
- Obter evidência real de transporte autenticado/criptografado e de que a revogação confirmada retira todos os peers ativos e persistentes. Estados locais e fakes não satisfazem este gate.
- Confirmar os hosts externos e a localização da API antes de gerar configuração publicável. O plano local não presume esses valores.

## Residual risks

- Sem integração live, não há prova de handshake, importação em cliente Apple, observabilidade de eventos VPN ou revogação efetiva; apenas formato, protocolos e estados locais são verificáveis.
- A fonte de verdade e a interface do controlador WireGuard externo são desconhecidas. O adaptador local pode precisar de implementação específica sem alterar a porta ou os critérios.
- O comportamento de logs de infraestrutura não é inferível do código dos três projetos; a afirmação de privacidade permanece bloqueada até verificação no ambiente de publicação.
- Perda da chave mestra torna os perfis cifrados irrecuperáveis. Backup, rotação e recuperação operacional não foram aprovados e não são inventados neste MVP.
- Sem FQDN do dashboard/API, a política de origem, cookies e certificados só pode ser validada estruturalmente, não em tráfego publicado.
- O QR contém uma credencial reutilizável; testes podem evitar vazamento acidental, mas não impedem captura voluntária da tela nem estabilizam uso simultâneo.
