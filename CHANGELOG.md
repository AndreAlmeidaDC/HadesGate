# Changelog

Todas as alterações notáveis neste projeto serão documentadas neste arquivo.

O formato é baseado no [Keep a Changelog](https://keepachangelog.com/pt-BR/1.0.0/),
e este projeto adere ao [Semantic Versioning](https://semver.org/lang/pt-BR/).

---

## [Unreleased]

### Adicionado (Added)
- **Fase 17 — BYOK Transparente Seguro**:
  - Identificação de tenant por fingerprint HMAC-SHA256 da chave original do provedor, sem persistência da credencial BYOK.
  - API e Console para cadastro e revogação de credenciais por tenant (`/api/provider-credentials`).
  - Suporte transparente a OpenAI-compatible, Anthropic e Gemini mantendo a chave no header nativo do SDK.
  - Rotas com credencial na URL desabilitadas por padrão, com migração temporária via `HADESGATE_ALLOW_LEGACY_URL_KEYS`.
  - Separação estrita entre privilégios administrativos e chaves globais/de tenant.
  - Gates de `govulncheck` e `npm audit`, Go 1.27.3, Next.js 16 e React 19.
  - DLP incremental em streams SSE com buffers independentes para conteúdo e argumentos de ferramentas fragmentados em OpenAI-compatible, Responses, Anthropic e Gemini.
  - Isolamento de consultas e exclusão de traces por tenant; somente o administrador pode consultar ou apagar a telemetria global.
  - CORS fechado por padrão, preflight não autorizado rejeitado e Private Network Access condicionado a uma origem permitida.
- **Fase 11 — Endurecimento de Segurança, DLP de Saída & Métricas Prometheus**:
  - Middleware de segurança e CORS estrito (`internal/web/security.go`) configurável via `HADESGATE_CORS_ALLOWED_ORIGINS` com suporte a extensões de navegador (`chrome-extension://*`) e headers de proteção OWASP (`nosniff`, `DENY`, `strict-origin-when-cross-origin`).
  - Rate Limiter contínuo Token Bucket (`internal/proxy/gateway.go`) com recarga fracionada temporal, substituindo o algoritmo de janela fixa e eliminando rajadas na virada de janela.
  - DLP de Saída para respostas JSON e streaming SSE (Egress Scanning em `internal/anonymizer/anonymizer.go` e `internal/proxy/upstream.go`): inspeção incremental para mascarar novos segredos ou PIIs alucinados como `[HG_EGRESS_<TYPE>_N]`, inclusive quando fragmentados entre eventos ou argumentos de ferramentas (`HADESGATE_ENABLE_EGRESS_SCAN`).
  - Exportador nativo de métricas Prometheus (`internal/web/server.go`) nos endpoints `/metrics` e `/api/metrics` com contadores por tipo de entidade, latências e streams ativos sem tocar dados sensíveis.
- **Fase 12 — Vault Criptografado em RAM & Webhooks de Auditoria**:
  - Vault Criptografado em Memória RAM (*Encrypted-in-RAM Vault* em `internal/anonymizer/vault/memory.go`): criptografia simétrica efêmera AES-256-GCM com chave de 256 bits volátil gerada no boot (`crypto/rand`), autenticação AAD vinculada a `sessionID:token` contra transplante de tokens e zeroing seguro na destruição de sessões e encerramento.
  - Despachante assíncrono de Webhooks de segurança e auditoria (`internal/webhook/dispatcher.go`): envio em segundo plano não-bloqueante para Discord, Slack, Microsoft Teams e SIEM (Elastic, Splunk, Syslog) em eventos de bloqueio por DLP (`block`), vazamentos de saída (`egress_leak`) e estouro de taxa (`rate_limit`), com suporte a assinatura HMAC-SHA256 (`HADESGATE_WEBHOOK_SECRET`).
- **Fase 13 — Enterprise Multitenancy & Central de Políticas Dinâmicas no Painel**:
  - Central de Configurações no Console Web (`frontend/components/settings.tsx`): 4ª aba `[04] POLÍTICAS & CONFIGURAÇÕES` com estética Industrial Brutalism para gestão interativa de regras de DLP, multitenancy, webhooks e proxy.
  - Persistência Dinâmica em SQLite (`internal/config/store.go`): tabelas `system_config` e `tenants` no SQLite local permitindo enxugar o `.env` para focar somente em credenciais de boot.
  - Recarga a Quente em Tempo Real (*Zero-Downtime Hot Reload*): reconfiguração atômica em memória (`sync.RWMutex`) em `Gateway`, `PolicyEngine`, `WebhookDispatcher` e `UpstreamDispatcher` via APIs protegidas (`/api/config`, `/api/tenants`) sem interrupção de conexões ou streams SSE ativos.
  - Gestão de Multitenancy & Quotas: cadastro, edição e revogação dinâmica de tenants com chaves de proxy isoladas, limites de taxa de requisições por minuto e políticas personalizadas.
  - Testador Interativo de Webhook (`/api/webhooks/test`): botão no painel para envio síncrono de ping de teste com retorno imediato de sucesso ou código HTTP.
- **HUD Flutuante Compacto no Browser Shield (Extensão)**:
  - Painel flutuante nos chats de IA colapsado por padrão exibindo apenas o ícone e contagem de tokens, expandindo suavemente sob demanda para não cobrir o campo de texto ou botões de envio.

### Em Planejamento (Planned)
- Suporte a empacotamento da extensão HadesGate™ Browser Shield para Firefox (Gecko / WebExtensions).
- Novos detectores internacionais de conformidade (US SSN/EIN, UK NINO, identificadores europeus).
- Recarga a quente de políticas em tempo de execução sem interrupção de serviço (*hot-reload* via `SIGHUP` / API).

## [2.0.0] - 2026-11-04

### Rebrand & Identidade Visual (Rebranding & Industrial Brutalism)
- **Transição Completa para HadesGate™**:
  - Rebranding estrutural de *PrivaGate* para **HadesGate** no módulo Go (`github.com/hadesgate/hadesgate`), CLI (`cmd/hadesgate`), documentação e assets.
  - Formato de tokens opacos atualizado para `[HG_<ns>_<TYPE>_<N>]` com namespace imprevisível (`crypto/rand`) exclusivo por requisição, tornando impossível a correlação cruzada entre conversas.
  - Headers HTTP padronizados para `X-HadesGate-Request-ID` e `X-HadesGate-Tenant-ID`.
  - Prefixos de variáveis de ambiente unificados sob `HADESGATE_*`, mantendo fallback retrocompatível total para chaves legadas `PRIVAGATE_*`.
- **Console Web em Industrial Brutalism & Tactical Telemetry**:
  - Interface do dashboard redesenhada em Next.js 14 com estética tática industrial: bordas de 0px (`rounded-none`), divisórias expostas com precisão de 1px (`gap-px`, hairlines), tipografia de impacto (*Archivo Black* e *JetBrains Mono*) e paleta com roxo primário `#7e22ce`.
  - Dark mode padrão com simulação de scanlines CRT e toggle completo para Light Mode.
  - Suporte nativo completo a internacionalização (pt-BR e en) em `frontend/lib/i18n.ts`.

### Extensão de Navegador (HadesGate™ Browser Shield — Manifest V3)
- **HadesGate™ Browser Shield para Navegadores Chromium**:
  - Extensão nativa Manifest V3 em `extension/` compatível com Google Chrome, Brave, Microsoft Edge, Arc e Vivaldi.
  - **Interceptação de Colagem (`Ctrl+V`)**: Intercepta a colagem de texto no chat de IA, anonimizando dados sensíveis via HadesGate local/remoto em microssegundos antes do envio.
  - **Botão Tático Flutuante & Atalho (`Alt+H`)**: Overlay no canto do chat para mascarar o texto digitado sob demanda antes da submissão.
  - **Auto-Restauração no DOM (`MutationObserver`)**: Conforme os eventos SSE da IA chegam com os tokens `[HG_...]`, a extensão substitui os tokens de volta pelos valores reais diretamente na tela, mantendo o mapeamento apenas na RAM da aba. O provedor de IA na nuvem nunca tem acesso aos dados reais.
  - **Diretiva Anti-Recusa Automática**: Injeta automaticamente na primeira mensagem a instrução do sistema que orienta o modelo a responder normalmente sobre os dados parametrizados sem emitir avisos ou recusas.
  - **Compatibilidade Ampla com Chats Web**: Suporte testado a ChatGPT, Claude.ai, Google Gemini, Google AI Studio, DeepSeek, Perplexity, Grok e GitHub Copilot.
  - **Popup de Controle Operacional**: Visualização em tempo real do status de conexão (`ONLINE // PROTEGIDO`), configuração de endpoint e validação de credenciais.

### Suporte Multi-Protocolo (OpenAI, Anthropic & Google Gemini)
- **Google Gemini Native API**:
  - Suporte aos endpoints `/v1beta/models/{model}:generateContent` e `:streamGenerateContent?alt=sse`, bem como rotas com prefixo `/gemini/`.
  - Injeção automática da diretiva de placeholders no schema de system instructions do Gemini (`InjectGeminiSystemInstruction`).
  - Suporte a roteamento transparente de modelos Gemini através do formato padrão `/v1/chat/completions`.
- **Anthropic Messages API**:
  - Endpoints dedicados `/v1/messages` e `/anthropic/v1/messages`.
  - Forçamento de `Accept-Encoding: identity` no upstream para evitar o recebimento de chunks comprimidos com gzip em streams SSE.
  - Remoção automática de headers de navegador (`Origin`, `Referer`, `Sec-Fetch-*`) prevenindo recusas da Anthropic por verificação de CORS.
  - Injeção da diretiva de placeholders via `InjectAnthropicSystemInstruction`.
- **OpenAI Responses API & Completions**:
  - Suporte a `/v1/responses` e `/responses` além do tradicional `/v1/chat/completions`.
  - Extratores dedicados para conteúdo multimodal (imagens/texto) e argumentos de *tool calls* em streaming e não-streaming.
  - Preservação estrita de todos os campos e opções (*choices*) da estrutura SSE original.
- **Descoberta de Modelos**:
  - Endpoints `/v1/models` e `/models` para listagem e roteamento de catálogo de modelos ativos.

### Motor de Anonimização, Novos Detectores & Resiliência
- **Detector IBAN Internacional (`iban.go`)**: Validação algorítmica ISO 13616 Mod-97 com suporte a separação de blocos e descascamento de pontuação trailing.
- **Detector de Endereços IP (`ip.go`)**: Identificação de IPv4 e IPv6 com filtro de exclusão para numeração de versões semânticas (ex.: `v1.2.3.4`).
- **Telefones Internacionais e Brasileiros (`phone.go`)**: Validação de formato internacional E.164 e verificação estrita de DDDs e nono dígito móvel nacional.
- **Detector de CEP (`cep.go`)**: Identificação de códigos postais brasileiros com validação de formato e contexto postal.
- **Expansão de API Keys e Segredos Corporativos (`apikey.go`, `secret.go`)**:
  - Detectores para chaves de Groq, xAI, GitLab, NPM, PyPI, SendGrid, HuggingFace, Slack, Stripe, AWS, Google AI Studio e OpenAI/Anthropic.
  - Identificação de tokens de autenticação `Bearer`, senhas em proximidade contextual, chaves SSH inline, URIs de banco de dados com credenciais e certificados PEM.
- **Aprimoramento do Detector de Nomes Brasileiros (`name.go`)**:
  - Expansão de dicionários contextuais: saudações formais, honoríficos, cargos, verbos de cobrança/faturamento e preposições conectivas.
  - Suporte a detecção de nomes próprios seguidos imediatamente de documentos sem vírgula (ex.: "cobrança para Rodolfo Bandeira de CPF: ...").
  - Tratamento seguro de fronteiras de runes UTF-8 em `ExtractSurroundingContext`.
- **Deduplicação Canônica de Entidades**: Entidades com formatos equivalentes no mesmo prompt (como `529.982.247-25` e `52998224725`) recebem o mesmo token identificador para preservar a integridade lógica e de cálculo do LLM.
- **Detokenizador com Janela Deslizante Resiliente**: Reconstrução confiável de tokens fracionados entre eventos SSE sucessivos, com capacidade de recuperação para tokens em caixa baixa ou com remoção de namespace gerados por LLMs.
- **Sidecar Semântico Opcional**: Camada de detecção via modelo externo HTTP, desativada por padrão (`SEMANTIC_DETECTOR_ENABLED=false`), com circuit breaker, timeout rigoroso e fallback transparente para o motor determinístico.

### Telemetria SQLite Zero-PII
- **Persistência de Telemetria sem CGO**:
  - Armazenamento em SQLite em Go puro (`modernc.org/sqlite`, `CGO_ENABLED=0`), sem dependências nativas de C.
  - **Invariante Zero-PII Estrita**: Zero dados sensíveis em disco. Apenas métricas estatísticas, categorias detectadas e o prompt já sanitizado são gravados. Mapeamentos e valores originais nunca tocam o banco de dados.
  - Métricas operacionais em tempo real: volume de requisições, taxa de mascaramento, tempo de scan em microssegundos, requisições por hora, provedor, modelo, idioma e distribuição de PIIs/segredos.
  - Purga mandatória e imediata da sessão na RAM ao término da resposta (`defer DeleteSession`).

### Políticas de Privacidade, Governança & Multi-Tenancy
- **Ações Granulares por Entidade**: Definição de regras `mask` (mascarar com token), `block` (bloquear requisição) ou `route_local` (desviar requisição sensível para modelo local Ollama).
- **Separação de Credenciais**: `HADESGATE_ADMIN_KEY` para painel e APIs de gestão; `HADESGATE_PROXY_KEY` exclusiva para encaminhamento de requisições de IA.
- **Suporte a Multi-Tenancy**: Chaves de proxy isoladas por tenant (`HADESGATE_TENANT_PROXY_KEYS`) e políticas por entidade específicas por tenant (`HADESGATE_TENANT_ENTITY_ACTIONS`).
- **Limitação de Taxa (Rate Limit)**: Controle opcional de requisições por minuto por tenant, implementado com estruturas estáticas em RAM sem criar goroutines.
- **Allowlist Estrita de Upstreams Dinâmicos**: `HADESGATE_DYNAMIC_UPSTREAM_ALLOWED_HOSTS` bloqueia tentativas de SSRF através do header `X-Upstream-Url`.

### Ferramentas de Desenvolvedor & Qualidade
- **Scripts de Interceptação para Terminal**:
  - `scripts/enable-hadesgate.ps1` e `scripts/disable-hadesgate.ps1` para interceptação transparente de ferramentas de IA (Aider, Claude Code, Cursor, scripts locais) via variáveis de ambiente do sistema.
- **Gate de Acurácia Automatizado no CI**:
  - `cmd/pii-benchmark` avalia o corpus LGPD versionado (`benchmarks/pii-accuracy/corpus.json`) exigindo precisão e recall $\ge 0.90$ (atingindo 100% de acurácia no estado atual).

### Removido (Removed)
- **Remoção de Subsistema de Custos e Precificação**:
  - Eliminado todo o registro de preços de provedores, estimativa de tokens de custo financeiro e rotas de sincronização de feeds externos (`/api/pricing`), focando o produto 100% na segurança, anonimização e performance Zero-PII.

## [1.4.0] - 2026-09-17

### Segurança & Endurecimento (Security Hardening & Access Control)
- **Autenticação Mandatória no Painel Web e Middleware Go (`golang-security`)**:
  - **Login Obrigatório (URL + API Key)**: Interface de login do dashboard passa a exigir estritamente URL válida e API Key não vazia. Credenciais são validadas contra o backend antes de desbloquear a interface.
  - **Middleware Unificado de Autorização (`requireAuth`)**: Proteção com status `401 Unauthorized` em todas as rotas internas de API (`/api/traces`, `/api/vault`, `/api/pricing`, `/api/pricing/sync`, `/api/test-anonymize`), suportando tanto `Authorization: Bearer <key>` quanto o header padrão `X-API-Key: <key>`.
  - **Prevenção a Ataques de Temporização (*Timing-Attacks*)**: Comparação criptográfica estrita em tempo constante (`crypto/subtle.ConstantTimeCompare`) com checagem de comprimento não-nulo, impedindo bypass por strings vazias ou análise de tempo de resposta.
  - **Geração Automática de Chave Segura**: Se `PRIVAGATE_ADMIN_KEY` não for configurada no boot, o servidor Go gera automaticamente uma chave criptográfica forte (32 caracteres hex via `crypto/rand`) e a exibe no console, garantindo que o gateway nunca inicie em modo desprotegido.
  - **Headers de Segurança HTTP (OWASP)**: Injeção de `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `X-XSS-Protection: 1; mode=block` e `Referrer-Policy: strict-origin-when-cross-origin`.
  - **Proteção contra Exaustão de Recursos (DoS / OOM)**: Adicionado `http.MaxBytesReader` limitando payloads de entrada (10MB no proxy `/v1/chat/completions`, 2MB em `/api/test-anonymize` e 16KB em `/api/auth/verify`).
  - **Sanitização de Status**: `/api/status` sem autenticação retorna apenas sinal de vida e versão; métricas detalhadas de consumo de RAM, goroutines e contadores de requisições requerem autorização prévia.
  - **Frontend shadcn/ui**: Atualização dos componentes `Login`, `Observability` e `Playground` com propagação de headers de autorização e tratamento de sessão.
  - **Testes Unitários de Segurança (`internal/web/server_test.go`)**: Suite automatizada cobrindo testes de verificação de chave, rejeição de credenciais vazias, proteção 401 e headers de resposta.

## [1.3.0] - 2026-09-17

### Adicionado (Added)
- **Detecção e Anonimização de API Keys, Segredos Corporativos e Credenciais Criptográficas (Arquitetura Gitleaks / Trufflehog em Go)**:
  - **Identificação Determinística de Chaves de Provedores Líderes (`EntityAPIKey` / `[API_KEY_X]`)**:
    - **OpenAI**: Chaves de projeto (`sk-proj-[...]`) e chaves padrão legadas (`sk-[...]`).
    - **Anthropic**: Chaves Claude (`sk-ant-[...]`).
    - **AWS**: Access Key IDs (`AKIA[...]`, `ASIA[...]`).
    - **GitHub**: Personal Access Tokens clássicos (`ghp_[...]`) e Fine-Grained Tokens (`github_pat_[...]`).
    - **Google / Gemini**: Chaves de API de serviços de IA (`AIza[...]`).
    - **Slack & Stripe**: Tokens de bot/usuário (`xoxb-[...]`, `xoxp-[...]`) e chaves secretas de pagamento (`sk_live_[...]`).
    - **HuggingFace**: Tokens de API (`hf_[...]`).
  - **Cálculo de Entropia de Shannon sem Alocação na Heap (`CalculateShannonEntropy`)**:
    - Análise matemática da aleatoriedade de strings em variáveis/tokens genéricos (`api_key =`, `Bearer`, `token :`), aceitando apenas tokens com $H \ge 3.2$ bits/char e eliminando falso-positivos em sub-100 nanossegundos.
  - **Identificação de Segredos de Infraestrutura e Criptografia (`EntitySecret` / `[SECRET_X]`)**:
    - **Chaves Privadas PEM**: Certificados criptográficos RSA, DSA, EC, OpenSSH e PGP (`BEGIN [...] PRIVATE KEY`).
    - **URIs de Banco de Dados com Credenciais**: Strings de conexão para PostgreSQL, MySQL, MariaDB, MongoDB e Redis contendo usuário e senha expostos.
    - **JSON Web Tokens (JWT)**: Tokens estruturados com cabeçalho, payload e assinatura Base64URL (`eyJ[...]`).
  - **Conformidade Estrita com PCI-DSS e Microsoft Presidio**: Finais de 4 dígitos de cartões de crédito mantidos como dados públicos de conferência, concentrando proteção apenas em credenciais sensíveis e PANs completos (Luhn).
  - **Frontend shadcn/ui & Playground**:
    - Badges semânticos com estilos dedicados para `api_key` (âmbar/laranja) e `secret` (carmesim/vermelho com glow).
    - Novo preset de testes de segredos corporativos no Playground.

## [1.2.0] - 2026-09-17

### Adicionado (Added)
- **Engine Inteligente de Reconhecimento e Anonimização de Dados Sensíveis (Arquitetura LiteLLM / Microsoft Presidio em Go)**:
  - **Janela de Proximidade Contextual (*Context-Aware Enhancers*)**: Análise heurística bidirecional de $\pm 5$ palavras ao redor dos candidatos a dados sensíveis (`ExtractSurroundingContext` e `FindContextKeywords`), aumentando a certeza de identificação sem penalidade de latência.
  - **Sistema de Pontuação de Confiança (*Confidence Scores: 0.0 a 1.0*)**: Classificação probabilística para cada detecção (1.0 para validações matemáticas algorítmicas, 0.85-0.95 para padrões com termos de suporte, 0.50-0.65 para números isolados) com filtro customizável (`ANONYMIZER_MIN_CONFIDENCE`, padrão `0.70`), prevenindo falso-positivos em dados operacionais.
  - **Desambiguação e Resolução de Conflitos por Maior Confiança**: Algoritmo de reivindicação de intervalos ordenado por score (`Score` decrescente $\rightarrow$ maior comprimento), garantindo que sobreposições sejam resolvidas em favor da entidade mais específica.
  - **Novos Detectores Especializados da Realidade Brasileira**:
    - **Chave PIX**: Detecção das 5 modalidades oficiais do Banco Central (Chaves Aleatórias EVP/UUID v4, CPF, CNPJ, E-mail e Telefone em contexto PIX).
    - **RG (Registro Geral / Identidade)**: Padrões estaduais (SP, RJ, MG, etc.) com exigência de termos de suporte contextual.
    - **CNH (Carteira Nacional de Habilitação)**: Validação matemática oficial dos 11 dígitos com cálculo dos 2 dígitos verificadores pelo algoritmo Módulo 11 do CONTRAN.
    - **Dados Bancários (Agência e Conta)**: Detecção de agências e contas correntes/poupança associadas a instituições financeiras nacionais (Itaú, Bradesco, Santander, Caixa, Banco do Brasil, Nubank, Inter, C6, etc.).
    - **Passaporte Brasileiro**: Padrão oficial da Polícia Federal (`[A-Z]{2}[0-9]{6}`).
  - **Allowlist (Whitelist) & Denylist (Blacklist)**:
    - `ANONYMIZER_ALLOWLIST`: Termos institucionais públicos (ex: e-mails de suporte, dados de homologação) que nunca devem ser mascarados.
    - `ANONYMIZER_DENYLIST`: Segredos industriais e termos confidenciais com prioridade máxima de anonimização (`[CONFIDENTIAL_X]`).
  - **Reconhecedores Customizados Pluggáveis via JSON (`ANONYMIZER_CUSTOM_FILE`)**: Suporte a inclusão de regras declarativas externas com regex, score base e palavras de contexto sem necessidade de recompilar o Go.
  - **Visualização Completa no Playground com shadcn/ui**:
    - Novos presets de testes LGPD (PIX, Dados Bancários, CNH, RG, Passaporte).
    - Badges temáticos para cada entidade com percentual de confiança (`95% conf.`) e palavras de contexto identificadas.

## [1.1.0] - 2026-09-17

### Adicionado (Added)
- **Engine de Precificação Dinâmica de Modelos de IA (Arquitetura LiteLLM / Portkey)**:
  - Registro em memória thread-safe (`PricingRegistry`) com tempo de busca inferior a 50 nanossegundos (`sync.RWMutex`).
  - Sincronização automática em background a cada 24 horas a partir do feed canônico comunitário (`PRICING_FEED_URL` / LiteLLM JSON com 300+ modelos).
  - Fallback automático para base offline embutida no binário (`//go:embed catalog/embedded_prices.json`), assegurando 100% de disponibilidade em ambientes desconectados ou air-gapped.
  - Suporte a sobreposição de tarifas personalizadas (`PRICING_CUSTOM_FILE`) para empresas com contratos de desconto negociados.
  - Endpoints REST de gerenciamento: `GET /api/pricing` (status do catálogo e tabela) e `POST /api/pricing/sync` (recarga sob demanda).
  - Modal interativo no Dashboard com busca instantânea por modelo/provedor, badges de fonte e botão "Sincronizar Agora".
  - Resolução inteligente de modelos por maior prefixo compatível e fine-tunes (`gpt-4o-2024-08-06`, `ft:gpt-4o:...`).

## [1.0.0] - 2026-09-17

### Adicionado (Added)
- **PrivaGate Core Gateway**: Reverse proxy em Go de alta performance com compatibilidade direta à API da OpenAI (`POST /v1/chat/completions`).
- **Motor Especializado de Anonimização LGPD**:
  - Validação matemática de CPF via algoritmo oficial Módulo 11 da Receita Federal (rejeição de sequências inválidas ou repetidas).
  - Validação matemática de CNPJ via pesos oficiais Módulo 11.
  - Validação de Cartões de Crédito via algoritmo de Luhn (Módulo 10) cobrindo Visa, Mastercard, Elo, Amex e Hipercard.
  - Validação de Telefones com verificação da tabela de DDDs oficiais e nono dígito móvel.
  - Detecção contextual e fonética de Nomes Próprios brasileiros.
  - Detecção de E-mails (RFC 5322) e CEPs brasileiros (`\d{5}-?\d{3}`).
- **Streaming Detokenizer em Tempo Real**:
  - Algoritmo de janela circular deslizante (*Sliding-Window Buffer*) para Server-Sent Events (SSE).
  - Resolução do problema de *Split-Token Boundary* (ex: token `[PERSON_1]` fragmentado em múltiplos chunks SSE como `[PER` e `SON_1]`) com latência de ~3 microssegundos por chunk.
- **Vault Epêmero em Memória RAM**:
  - Armazenamento volátil de tokens com limpeza periódica em background (*Garbage Collection* com TTL configurável via `VAULT_TTL_SECONDS`).
  - Zero persistência de dados sensíveis em disco.
- **Dashboard de Observabilidade Prática (Estilo Langfuse)**:
  - Ring buffer em memória volátil retendo os últimos traces de requisições.
  - Visualização de KPIs: volume de traces, contagem de PIIs protegidas, latência média e modelos ativos.
  - Tabela de traces com filtros em tempo real por modelo, busca textual, auto-refresh a cada 3s e limpeza com 1 clique.
  - Modal de inspeção detalhada com prompt sanitizado e mapeamento do cofre de tokens.
- **Métricas e Custos de IA (Padrão de Mercado)**:
  - Estimador de tokens de entrada e saída (Prompt Tokens, Completion Tokens, Total Tokens).
  - Tabela de precificação oficial da indústria para OpenAI (GPT-4o, GPT-4o-mini, GPT-4 Turbo, GPT-3.5, o1), Anthropic (Claude 3.5 Sonnet, Claude 3 Haiku, Opus), Google Gemini (1.5 Pro, 1.5 Flash), Groq (Llama 3 70B/8B) e modelos locais ($0.00).
  - Cálculo automático e exibição de custos estimados em USD (`cost_usd`) em cards, tabela e modal.
- **Autenticação Segura por URL + API Key**:
  - Tela de login dedicada para controle de acesso ao painel com suporte a ambientes locais e instâncias remotas.
  - Comparação em tempo constante (`crypto/subtle.ConstantTimeCompare`) no gateway.
- **Playground & Dual-Stream Inspector**:
  - Interface interativa com presets pré-configurados de testes LGPD e inspeção lado a lado (Original vs Sanitizado vs Restaurado).
- **Guia de Integração Multi-Linguagem**:
  - Exemplos prontos para copiar em Python (OpenAI SDK), cURL, Node.js e LangChain.
- **Infraestrutura Docker Multi-Stage**:
  - Imagem única estática baseada em Alpine Linux (~15 MB) com Next.js embutido (`go:embed`).
  - Consumo de apenas ~3 MB de memória RAM em execução contínua.

### Modificado (Changed)
- Menu superior redesenhado para visual minimalista, substituindo headers com poluição visual.
- Navegação dividida em três seções focadas: *Observabilidade & Logs*, *Playground & Testes* e *Como Integrar*.

### Removido (Removed)
- Removido o badge de indicador de conexão `localhost:8080` da barra de navegação para maximizar o minimalismo.
- Removido o componente estático de visualização de detectores LGPD do painel de administração.
