# HadesGate Roadmap

> Roadmap oficial do produto **HadesGate™**. Acompanhamento de fases concluídas e planejamento de novos recursos.

## Princípios do Produto (Product Principles)

1. **Leve por padrão (Lightweight by default)**: o núcleo permanece um único binário Go com processamento estritamente em memória RAM.
2. **Caminho rápido determinístico (Fast deterministic path)**: detectores de formato, checksum, contexto e segredos executam localmente em sub-milissegundos, sem inferência de IA no caminho crítico.
3. **Privacidade por padrão (Privacy by default)**: mapeamentos de texto puro são voláteis, isolados por requisição (`crypto/rand`) e excluídos permanentemente de logs e telemetria.
4. **IA local opcional (Optional local AI)**: detecção semântica pode ser ativada via sidecar externo HTTP, mas é desativada por padrão.
5. **Degradação segura (Safe degradation)**: falhas ou ausência de componentes opcionais mantêm o pipeline de proteção determinística 100% ativo.

## Phase 1 — Correctness and Isolation

- [x] Use unpredictable, request-namespaced surrogate tokens (`[HG_<ns>_<TYPE>_<N>]`).
- [x] Prevent token reuse for different values in the same request context.
- [x] Isolate vault mappings from client-provided session identifiers.
- [x] Purge request mappings immediately after the response completes (`defer DeleteSession`).
- [x] Remove plaintext token maps from observability traces.
- [x] Restrict dynamic upstream URLs with explicit provider allowlists.
- [x] Add lightweight concurrency and response-size limits.
- [x] Add tenant-aware request-rate policies after proxy/admin credentials are separated.

## Phase 2 — Protocol Compatibility

- [x] Introduce protocol-specific text extractors.
- [x] Support OpenAI multimodal message content.
- [x] Support OpenAI Responses API (`/v1/responses`).
- [x] Preserve all SSE fields and choices while restoring text.
- [x] Support streamed and non-streamed tool-call arguments.
- [x] Add Anthropic Messages support (`/v1/messages`).
- [x] Add Google Gemini native and streaming API support (`/v1beta/models/` and `/gemini/` routes with `InjectGeminiSystemInstruction`).

## Phase 3 — Privacy Policies

- [x] Add per-entity actions: `mask`, `block`, and `route_local`.
- [x] Separate proxy credentials (`HADESGATE_PROXY_KEY`) from administrative credentials (`HADESGATE_ADMIN_KEY`).
- [x] Add tenant-scoped policies and safe audit metadata.
- [x] Add an optional local-provider route for sensitive requests (`route_local` pointing to Ollama/local).

## Phase 4 — Optional Semantic Detection

The semantic layer is an opt-in sidecar and is never required by the gateway.

- [x] Default: `SEMANTIC_DETECTOR_ENABLED=false`.
- [x] No model download or extra process when disabled.
- [x] Strict time budget and circuit breaker when enabled.
- [x] Deterministic detectors always run first.
- [x] Deterministic-only fallback on sidecar failure.
- [x] Support ambiguous entities such as names, organizations, and locations.

## Phase 5 — Measurable Detection Quality

- [x] Build a Portuguese/LGPD accuracy corpus with positive and negative controls.
- [x] Gate CI on deterministic detector precision and recall (`pii-benchmark`).
- [x] Track latency and allocations with Go benchmarks.
- [x] Keep semantic accuracy/performance separate by leaving the sidecar disabled in core gates.

## Phase 6 — Detector & Anonymizer Superiority (PasteGuard Parity+)

- [x] Add ISO 13616 Mod-97 `IBAN` detector with trailing-token peeling (`iban.go`).
- [x] Add IPv4 and IPv6 `IP_ADDRESS` detector with semantic-version exclusion (`ip.go`).
- [x] Expand `API_KEY` & `SECRET` detectors for Bearer headers, contextual passwords, inline SSH keys, and modern provider keys (Groq, xAI, GitLab, NPM, PyPI, SendGrid).
- [x] Add international E.164 phone detection alongside Brazilian DDD validation (`phone.go`).
- [x] Add Brazilian postal code `CEP` detector (`cep.go`).
- [x] Harden UTF-8 rune boundary safety in `ExtractSurroundingContext` and regex-based denylist scanning.
- [x] Enforce deterministic-over-semantic overlap precedence (`IsSemantic`) and context-aware CPF/CNH disambiguation.
- [x] Add canonical entity deduplication (`529.982.247-25` == `52998224725`, case-insensitive names/emails) and resilient LLM detokenization (`[[...]]`, lowercase, and namespace-stripped placeholders).

## Phase 7 — Zero-Leak SQLite Telemetry & Anonymization-First Dashboard

- [x] Remove all AI model pricing, token estimation, and cost-tracking machinery to focus 100% on data anonymization.
- [x] Persist non-sensitive anonymization telemetry (`requests`, `masked`, `scan_time`, `rate /hour`, `provider`, `model`, `language`, `pii_types`, `secret_types`, and masked prompts) in pure-Go SQLite (`modernc.org/sqlite`, `CGO_ENABLED=0`) with zero raw PII or token maps stored.
- [x] Redesign embedded Next.js dashboard around objective anonymization KPIs and expandable masked-prompt inspection.

## Phase 8 — HadesGate Rebrand & Industrial Brutalist UI

- [x] Rebrand gateway module, CLI (`cmd/hadesgate`), tokens (`[HG_...]`), headers (`X-HadesGate-*`), config (`HADESGATE_*` with `PRIVAGATE_*` fallback), and Docker image to **HadesGate**.
- [x] Implement **Industrial Brutalism & Tactical Telemetry** UI (`0px` radius, `gap-px` exposed hairlines, `Archivo Black` + `JetBrains Mono`, purple primary accent, Dark Mode default with CRT scanlines + full Light Mode toggle).
- [x] Provide full bilingual UI support (PT-BR default and EN) via `frontend/lib/i18n.ts`.

## Phase 9 — HadesGate™ Browser Shield (Manifest V3)

- [x] Build Manifest V3 Web Extension in `extension/` for Chromium-based browsers (Chrome, Brave, Edge, Arc, Vivaldi).
- [x] Intercept clipboard paste events (`Ctrl+V`) on AI web chats and anonymize PII/secrets in microseconds via local or remote HadesGate.
- [x] Provide a tactical floating overlay button and hotkey (`Alt+H`) directly inside web chat interfaces for on-demand masking.
- [x] Restore tokens back to original values in the browser DOM via `MutationObserver` as SSE stream chunks arrive, ensuring zero plaintext reaches cloud providers while preserving user readability.
- [x] Automatically inject anti-refusal directives on the first conversational turn so models accept surrogate tokens as real data.
- [x] Support major web AI interfaces: ChatGPT, Claude.ai, Google Gemini, Google AI Studio, DeepSeek, Perplexity, Grok, and GitHub Copilot.
- [x] Include tactical popup UI with real-time status indicators (`ONLINE // PROTECTED`), endpoint configuration, and API key validation.

## Phase 10 — Contextual Name Detection, Document Apposition & 100% Accuracy Gate

- [x] Expand contextual proper name detection (`detector/name.go`) with extensive greeting, honorific, role, invoice, and action verb triggers.
- [x] Support Brazilian proper names immediately followed by document appositions without commas (e.g., `"cobrança para Rodolfo Bandeira de CPF: 09897684492"`).
- [x] Harden UTF-8 rune boundary slicing in context extraction routines.
- [x] Expand `benchmarks/pii-accuracy/corpus.json` with positive and negative controls maintaining 100% precision and recall (gate threshold $\ge 90\%$).
- [x] Provide Windows PowerShell environment scripts (`scripts/enable-hadesgate.ps1`, `scripts/disable-hadesgate.ps1`) for transparent CLI AI tooling interception (Aider, Claude Code, Cursor).

---

## 🔮 Em Planejamento & Futuro (Future Roadmap)

### Phase 17 — BYOK Transparente Seguro & Separação de Privilégios

- [x] Separar autorização administrativa de chaves globais e de tenants, impedindo elevação de privilégio por `proxy_key`.
- [x] Autenticar o tenant por fingerprint HMAC-SHA256 da chave original do provedor, mantendo o fluxo de troca exclusiva da `base_url`.
- [x] Persistir apenas fingerprint, provedor, rótulo e dica curta; nunca a chave BYOK original.
- [x] Adicionar cadastro e revogação de credenciais BYOK no Console e na API administrativa.
- [x] Desabilitar por padrão URLs com credenciais (`/p/{key}` e `/t/{proxy_key}`), preservando uma flag temporária de migração.
- [x] Fixar toolchain Go corrigida, tornar o build npm determinístico e adicionar gates de vulnerabilidade ao CI.
- [x] Aplicar DLP de saída incremental a streams SSE, conteúdo e tool calls fragmentadas, preservando plaintexts legítimos restaurados pelo vault.
- [ ] Criptografar os demais segredos operacionais persistidos no SQLite e torná-los write-only na API.
- [x] Isolar consultas e purge de traces por tenant, mantendo consulta e purge global exclusivos do administrador.
- [x] Fechar CORS por padrão, rejeitar preflight não autorizado e conceder Private Network Access apenas a origens explicitamente permitidas.
- [ ] Bloquear SSRF por resolução DNS/IP, inclusive loopback, link-local, redes privadas e DNS rebinding fora de upstreams locais explicitamente autorizados.
- [ ] Aplicar recarga dinâmica de CORS, headers HTTP adicionais e bind de rede local por padrão com opt-in explícito para exposição externa.

### Phase 18 — Segredos Operacionais & Zero-Trust Runtime

- [ ] Criptografar segredos operacionais persistidos com AES-256-GCM e chave mestra externa ao SQLite.
- [ ] Tornar campos secretos write-only na API e no Console, com rotação sem exposição do valor atual.
- [ ] Adicionar proteção SSRF baseada em resolução DNS/IP e validação em cada redirecionamento/conexão.
- [ ] Introduzir `HADESGATE_BIND_ADDRESS` seguro por padrão e perfis explícitos para Docker/Kubernetes.
- [ ] Eliminar data races em recargas dinâmicas com snapshots imutáveis/atômicos de configuração.

### Phase 19 — Governança, Retenção & Auditoria Enterprise

- [ ] Configurar retenção de telemetria por tenant, expiração temporal e quotas de armazenamento.
- [ ] Adicionar papéis administrativos mínimos (`viewer`, `operator`, `admin`) sem reutilizar credenciais de proxy.
- [ ] Exportar auditoria assinada e Zero-PII para SIEM, com proteção contra replay e fila persistente opcional.
- [ ] Registrar trilha de alterações de políticas e credenciais somente com metadados, autor e timestamp.
- [ ] Disponibilizar backup e restauração criptografados das configurações, excluindo vault e plaintexts.

### Phase 20 — Resiliência, Compatibilidade & Operação

- [ ] Testes de caos para cancelamento de streams, upstream lento, desconexões e limites de memória.
- [ ] Matriz de compatibilidade automatizada por protocolo e SDK oficial, sem depender de chaves reais no CI.
- [ ] Health/readiness separados, shutdown drenando streams e métricas de saturação por tenant.
- [ ] Orçamento de performance contínuo para DLP de entrada/saída, streaming e SQLite.
- [ ] Política versionada de migração e remoção definitiva das rotas legadas com chave na URL.

### Phase 11 — Endurecimento de Segurança, DLP de Saída & Métricas Prometheus

- [x] **Refinamento de CORS**: Origens permitidas configuráveis via `HADESGATE_CORS_ALLOWED_ORIGINS` e isolamento restrito de rotas administrativas `/api/*` com headers de proteção OWASP.
- [x] **Rate Limiting Token Bucket**: Algoritmo contínuo de balde de tokens para eliminação do problema de rajadas na virada de janela (burst boundary).
- [x] **DLP de Saída (Egress Scanning)**: Inspeção bidirecional opcional em respostas JSON e streams SSE, inclusive conteúdo e argumentos de ferramentas fragmentados (`HADESGATE_ENABLE_EGRESS_SCAN`).
- [x] **Métricas Prometheus (`/metrics`)**: Endpoint nativo no formato padrão Prometheus (latências de scan e proxy, contadores por tipo de PII, streams ativos) mantendo o princípio Zero-PII.

### Phase 12 — Vault Criptografado em RAM & Webhooks de Auditoria

- [x] **Criptografia em Memória RAM (*Encrypted-in-RAM Vault*)**: Criptografia simétrica efêmera (AES-256-GCM com chave de 256 bits volátil gerada no boot e destruição com zeroing no encerramento) dos mapeamentos de tokens no heap com isolamento AAD contra transplante de tokens.
- [x] **Webhooks de Segurança em Tempo Real**: Disparo assíncrono não-bloqueante de eventos HTTP para SIEM (Syslog, Elastic, Splunk) e canais corporativos (Slack, Teams, Discord) com assinatura HMAC-SHA256 (`HADESGATE_WEBHOOK_URL`, `HADESGATE_WEBHOOK_SECRET`, `HADESGATE_WEBHOOK_EVENTS`) quando políticas com ação `block`, `rate_limit` ou `egress_leak` forem acionadas.

### Phase 13 — Enterprise Multitenancy & Central de Políticas Dinâmicas no Painel

- [x] **Ambiente de Gestão de Políticas no Painel Web (`[04] POLÍTICAS & CONFIGURAÇÕES`)**: Interface tática Industrial Brutalism no console para controle em tempo real de regras de DLP por entidade, webhooks, roteamento de agentes/upstreams e tenants.
- [x] **Persistência Dinâmica em SQLite & Enxugamento do `.env`**: Migração das regras de negócio e tenants para o banco SQLite local (`data/hadesgate.db`), deixando o `.env` enxuto apenas para portas e chaves mestras de boot.
- [x] **Recarga a Quente em RAM (*Zero-Downtime Hot Reload*)**: Atualização atômica em memória (`sync.RWMutex`) via endpoints `/api/config` e `/api/tenants` sem interrupção de conexões ativas ou streams SSE.
- [x] **Gestão Multitenancy & Quotas**: Cadastro, edição e revogação dinâmica de tenants com chaves de proxy dedicadas, limites de requisições por minuto (`rate limit`) e políticas DLP granulares por departamento.
- [x] **Testador Interativo de Webhook**: Disparo de ping de teste direto pelo painel para validação imediata de integração com Discord, Slack, Teams ou SIEM.

### Phase 14 — Detectores Internacionais & Frameworks Regulatórios

- [x] **US Identifiers (SSN & EIN)**: Validação estrutural rigorosa de US Social Security Number (SSA area, group, serial e números de teste/dummies) e Employer Identification Number (IRS campus prefix validation).
- [x] **UK Identifiers (NINO & NHS)**: Validação de UK National Insurance Number (prefixos restritos HMRC, regras alfanuméricas e sufixo A-D) e UK National Health Service Number via Módulo 11 oficial.
- [x] **Identificadores Europeus (DNI/NIE, NIR, Codice Fiscale)**: Documentos de identidade espanhóis (DNI e NIE com algoritmo Módulo 23), NIR / Sécurité Sociale francês com Módulo 97 (INSEE), e Codice Fiscale italiano com tabela de paridade da Agenzia delle Entrate.
- [x] **Suite de Identificadores Médicos para Conformidade HIPAA**: Cobertura abrangente com NPI (National Provider Identifier com algoritmo Luhn 80840), DEA (Drug Enforcement Administration para prescrições controladas) e Prontuários Médicos (MRN / Patient ID).

### Phase 15 — Ecossistema, Empacotamento & Distribuição

- [x] **Empacotamento do HadesGate Browser Shield para Firefox Add-ons**: Compatibilidade nativa com motor Gecko e WebExtensions API via [`extension/manifest.firefox.json`](../extension/manifest.firefox.json) (ID `shield@hadesgate.dev`, `background.scripts`), documentação para `about:debugging` e ferramenta de empacotamento em Go (`cmd/package-extension`) gerando `hadesgate-shield-chrome.zip` e `hadesgate-shield-firefox.xpi`.
- [x] **Publicação de Binários Estáticos Multi-Plataforma**: Pipeline automatizado de compilação cruzada pura (`CGO_ENABLED=0`) para Linux (amd64/arm64), macOS (amd64/arm64 Apple Silicon) e Windows (amd64) via [`cmd/build-release`](../cmd/build-release), GitHub Actions ([`.github/workflows/release.yml`](../.github/workflows/release.yml)) com checksums `SHA256SUMS`, e manifesto [`.goreleaser.yml`](../.goreleaser.yml).
- [x] **Fórmula Homebrew Tap & Instalador Autônomo em Comando Único**:
  - Fórmula Homebrew Tap em [`Formula/hadesgate.rb`](../Formula/hadesgate.rb) e [`packaging/homebrew/hadesgate.rb`](../packaging/homebrew/hadesgate.rb) com suporte a serviço daemon (`brew services start hadesgate`).
  - Script instalador POSIX [`scripts/install.sh`](../scripts/install.sh) executável via `curl -fsSL https://get.hadesgate.dev | sh` com detecção de arquitetura/SO, download autenticado, validação de checksum SHA-256 e configuração de diretório `~/.hadesgate/`.

### Phase 16 — Taxonomia Avançada de Dados Sensíveis (HIPAA PHI, LGPD Art. 11 & Corporativo)

Expansão da matriz de privacidade do HadesGate para cobrir dados pessoais sensíveis regulados, segredos de negócio, dados biométricos, dispositivos e registros governamentais avançados, agrupados em macro-famílias com políticas DLP granulares:

- [x] **Arquitetura de Macro-Famílias & Limiares Granulares de Confiança**:
  - Resolução em 3 níveis de confiança (`EntityThreshold` > `FamilyThreshold` > Global `ANONYMIZER_MIN_CONFIDENCE`).
  - Agrupamento tático no Painel de Políticas com ações em lote (`[TODOS MASK]`, `[TODOS BLOCK]`, `[TODOS ROUTE_LOCAL]`) e controle de peso por família ou entidade específica.
- [x] **Roteamento Local com Contexto Enriquecido de Auditoria**:
  - Injeção automática de headers de contexto (`X-HadesGate-Sensitive-Families`, `X-HadesGate-Sensitive-Types`, `X-HadesGate-Sensitive-Count`, `X-HadesGate-Route-Action`).
  - Injeção do bloco de raciocínio de auditoria (`BuildLocalReasoningInstruction`) para modelos locais (Ollama), auxiliando na contextualização e determinação de sensibilidade sem vazamento externo.
- [x] **PHI & Saúde Digital (HIPAA & LGPD Art. 11)**:
  - `SUS_CARD`: Cartão Nacional de Saúde / CNS (15 dígitos com validação Módulo 11 oficial do DATASUS/Ministério da Saúde para cartões definitivos iniciados em 1/2 e provisórios iniciados em 7/8/9).
  - `CID`: Códigos diagnósticos da Classificação Internacional de Doenças (CID-10 e CID-11).
  - `MEDICATION`: Reconhecimento farmacológico ultra-rápido via autômato Aho-Corasick em Go com o catálogo oficial da ANVISA embutido (24.214 termos e princípios ativos em apenas 112 KB gzip), verificação/atualização em streaming via HTTP HEAD/GET sem gravação do CSV bruto em disco, e guarda contra homônimos ambíguos.
- [x] **Identificadores Veiculares & Dispositivos / Hardware**:
  - `VEHICLE`: Placa de veículo automotor (padrão Mercosul `AAA9A99` e padrão cinza `AAA-9999`), RENAVAM (11 dígitos com validação Módulo 11 do DENATRAN) e Chassi/VIN (17 caracteres conforme ISO 3779 com verificação de checksum de pesos).
  - `DEVICE_ID`: Identificadores únicos de hardware: IMEI (15 dígitos com algoritmo Luhn e contexto de telecomunicações) e Endereço MAC (padrão IEEE 802 em notações com dois pontos ou hífens).
- [x] **Identificadores Jurídicos & Governamentais Complementares**:
  - `LEGAL`: Número Único de Processo Judicial do CNJ (`NNNNNNN-DD.AAAA.J.TR.OOOO` com validação oficial Módulo 97 da Resolução 65 do CNJ) e termos processuais sob segredo de justiça.
  - `GOVERNMENT_ID`: Título de Eleitor (12 dígitos com algoritmo Módulo 11 de duplo dígito verificador do TSE) e PIS/PASEP/NIT (11 dígitos com Módulo 11 oficial).
  - `CORPORATE`: Detecção e categorização de dados empresariais e contratos confidenciais.
- [ ] **Categorias Especiais e Proteção Reforçada (Etapa 3 - Próxima)**:
  - `SESSION_DATA`: Identificadores de sessão ativos em requisições (cookies `Set-Cookie`, `PHPSESSID`, tokens `X-CSRF-Token`).
  - `HR`: Matrícula funcional/colaborador, demonstrativos salariais (holerites) e avaliações de desempenho.
  - `BIOMETRIC`: Representações e referências a templates biométricos e reconhecimento facial.
  - `GENETIC_DATA`: Dados de sequenciamento de DNA/RNA e resultados de exames genéticos.
  - `SEXUAL_HEALTH`: Informações de saúde reprodutiva, histórico sexual e exames de IST.
  - `ETHNIC_DATA`: Declarações identificáveis de raça ou etnia associadas a indivíduos.
  - `CHILDREN`: Dados escolares e filiação de menores de idade com proteção estrita (ECA / COPPA).

