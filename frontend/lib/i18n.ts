export type Language = "pt-BR" | "en";

export interface Translations {
  common: {
    telemetry: string;
    playground: string;
    sdkIntegration: string;
    settings: string;
    light: string;
    dark: string;
    exit: string;
    refresh: string;
    purge: string;
    emptyPayload: string;
    loading: string;
    clear: string;
    page: string;
    of: string;
    previous: string;
    next: string;
    rowsPerPage: string;
    all: string;
  };
  login: {
    clearanceRequired: string;
    authUnit: string;
    zeroLeakShield: string;
    subheading: string;
    gatewayEndpoint: string;
    adminKey: string;
    enterConsole: string;
    authenticating: string;
    invalidCredentials: string;
    connectionError: string;
    deterministicEngine: string;
    hologramUnit: string;
    hologramStatus: string;
    hologramProtocol: string;
  };
  observability: {
    kpiRequests: string;
    kpiMasked: string;
    kpiScanTime: string;
    kpiRate: string;
    totalLabel: string;
    securedRatio: string;
    microsecAvg: string;
    perHour: string;
    providerMatrix: string;
    searchPlaceholder: string;
    liveStream: string;
    noTracesTitle: string;
    noTracesDesc: string;
    thTime: string;
    thProviderModel: string;
    thLang: string;
    thPiiDetected: string;
    thSecrets: string;
    thAction: string;
    viewPrompt: string;
    maskedPromptTitle: string;
    sanitizedPayloadSub: string;
    detectedTypes: string;
    traceLatency: string;
    status: string;
    paginationShowing: string;
  };
  playground: {
    title: string;
    subtitle: string;
    inputPayload: string;
    targetModel: string;
    inputPlaceholder: string;
    executeInterception: string;
    intercepting: string;
    maskedPayloadTitle: string;
    reconstitutedStreamTitle: string;
    scanTime: string;
    securedItems: string;
    detectedTags: string;
    rawUpstreamWire: string;
  };
  codeSnippets: {
    title: string;
    subtitle: string;
    tabPython: string;
    tabGemini: string;
    tabNode: string;
    tabCurl: string;
    copy: string;
    copied: string;
  };
  settings: {
    title: string;
    subtitle: string;
    tabPolicies: string;
    tabTenants: string;
    tabWebhooks: string;
    tabProxy: string;
    saveChanges: string;
    saving: string;
    savedSuccess: string;
    testWebhook: string;
    testingWebhook: string;
    webhookSuccess: string;
    webhookFailed: string;
    addTenant: string;
    tenantId: string;
    tenantName: string;
    proxyKey: string;
    rateLimit: string;
    actions: string;
    edit: string;
    editTenant: string;
    saveTenant: string;
    cancel: string;
    delete: string;
    copyUrl: string;
    copyGlobalUrl: string;
    globalProxyTitle: string;
    globalProxyDesc: string;
    rotateKey: string;
    urlCopied: string;
    inheritGlobal: string;
    customPolicies: string;
    overrideDesc: string;
    numericIdHelp: string;
    generateKey: string;
    unlimited: string;
    entityColumn: string;
    policyColumn: string;
    mask: string;
    block: string;
    routeLocal: string;
    disabled: string;
    disabledWarning: string;
    allFamilies: string;
    familyBatchAction: string;
    setAllMask: string;
    setAllBlock: string;
    setAllRouteLocal: string;
    setAllDisabled: string;
    familyConfidence: string;
    applyToAllInGroup: string;
    customThreshold: string;
    defaultThreshold: string;
    anvisaCatalog: string;
    checkAnvisaSync: string;
    checkingAnvisaSync: string;
    anvisaUpToDate: string;
    egressScanTitle: string;
    egressScanDesc: string;
    ollamaUrlTitle: string;
    corsOriginsTitle: string;
    providerCredentials: string;
    providerCredentialDesc: string;
    provider: string;
    credentialLabel: string;
    providerKey: string;
    registerCredential: string;
    noCredentials: string;
    revokeCredential: string;
    secretConfigured: string;
    secretNotConfigured: string;
    secretUnchanged: string;
    newSecretPlaceholder: string;
    keyCopiedOnce: string;
  };
}

export const translations: Record<Language, Translations> = {
  "pt-BR": {
    common: {
      telemetry: "TELEMETRIA",
      playground: "PLAYGROUND",
      sdkIntegration: "SDK_INTEGRAÇÃO",
      settings: "POLÍTICAS & CONFIGURAÇÕES",
      light: "CLARO",
      dark: "ESCURO",
      exit: "SAIR",
      refresh: "ATUALIZAR",
      purge: "LIMPAR LOGS",
      emptyPayload: "[ PAYLOAD VAZIO ]",
      loading: "CARREGANDO...",
      clear: "LIMPAR",
      page: "PÁGINA",
      of: "DE",
      previous: "ANTERIOR",
      next: "PRÓXIMO",
      rowsPerPage: "LINHAS POR PÁGINA",
      all: "TODOS",
    },
    login: {
      clearanceRequired: "HADESGATE™ // AUTORIZAÇÃO DE SEGURANÇA NECESSÁRIA",
      authUnit: "[ UNIDADE_AUTH // 00 ]",
      zeroLeakShield: "ESCUDO ZERO-LEAK",
      subheading: ">>> GATEWAY DE ANONIMIZAÇÃO DETERMINÍSTICA DE DADOS",
      gatewayEndpoint: "01 // ENDPOINT DO GATEWAY",
      adminKey: "02 // CHAVE DE ACESSO ADMIN",
      enterConsole: "ACESSAR CONSOLE DE TELEMETRIA",
      authenticating: "AUTENTICANDO...",
      invalidCredentials: "CREDENCIAS INVÁLIDAS PARA ESTA UNIDADE HADESGATE.",
      connectionError: "FALHA DE CONEXÃO COM O GATEWAY. VERIFIQUE A URL.",
      deterministicEngine: "MOTOR DETERMINÍSTICO // GO CGO=0",
      hologramUnit: "[ ESCUDO DE PRIVACIDADE // ZERO-PII ]",
      hologramStatus: "PROTEÇÃO DE INFERÊNCIA",
      hologramProtocol: "CRIPTOGRAFIA DETERMINÍSTICA EM RAM",
    },
    observability: {
      kpiRequests: "REQUISIÇÕES",
      kpiMasked: "MASCARADOS",
      kpiScanTime: "TEMPO DE SCAN",
      kpiRate: "TAXA",
      totalLabel: "total de interceptações",
      securedRatio: "dados assegurados",
      microsecAvg: "µs média determinística",
      perHour: "/hora em tempo real",
      providerMatrix: "MATRIZ_PROVEDORES >>>",
      searchPlaceholder: "FILTRAR POR PROVEDOR, MODELO OU ENTIDADE...",
      liveStream: "EM TEMPO REAL",
      noTracesTitle: "NENHUM DADO DE TELEMETRIA REGISTRADO",
      noTracesDesc: "Envie requisições através do proxy ou teste no Playground para visualizar as métricas de anonimização em tempo real.",
      thTime: "HORA",
      thProviderModel: "PROVEDOR / MODELO",
      thLang: "IDIOMA",
      thPiiDetected: "PII DETECTADO",
      thSecrets: "SEGREDOS",
      thAction: "AÇÃO",
      viewPrompt: "PROMPT",
      maskedPromptTitle: ">>> PROMPT MASCARADO // PAYLOAD SANITIZADO",
      sanitizedPayloadSub: "(Zero dados sensíveis trafegam para a IA ou são salvos em disco)",
      detectedTypes: "TIPOS DETECTADOS",
      traceLatency: "LATÊNCIA DO GATEWAY",
      status: "STATUS",
      paginationShowing: "EXIBINDO",
    },
    playground: {
      title: "SANDBOX DE ANONIMIZAÇÃO EM TEMPO REAL",
      subtitle: ">>> TESTE MASCARAMENTO DETERMINÍSTICO & DETOKENIZAÇÃO STREAMING SSE",
      inputPayload: "PAYLOAD DE ENTRADA // PROMPT DO USUÁRIO",
      targetModel: "MODELO ALVO:",
      inputPlaceholder: "INSIRA TEXTO COM CPF, CNPJ, IBAN, EMAIL, SENHA OU API KEY...",
      executeInterception: "EXECUTAR INTERCEPTAÇÃO",
      intercepting: "INTERCEPTANDO...",
      maskedPayloadTitle: "01 // PAYLOAD MASCARADO (ENVIADO AO PROVEDOR)",
      reconstitutedStreamTitle: "02 // STREAM SSE RECONSTITUÍDO (CLIENTE FINAL)",
      scanTime: "TEMPO DE SCAN:",
      securedItems: "ITENS ASSEGURADOS:",
      detectedTags: "TAGS DETECTADAS",
      rawUpstreamWire: "TRANSMISSÃO BRUTA DO PROVEDOR (LLM >>> HADESGATE):",
    },
    codeSnippets: {
      title: "INTEGRAÇÃO SDK // GUIA RÁPIDO",
      subtitle: ">>> CONEXÃO DIRETA COM O PROXY REVERSO HADESGATE",
      tabPython: "PYTHON (OPENAI SDK)",
      tabGemini: "GEMINI SDK",
      tabNode: "NODE.JS / TYPESCRIPT",
      tabCurl: "CURL CLI",
      copy: "COPIAR",
      copied: "COPIADO!",
    },
    settings: {
      title: "PAINEL DE CONFIGURAÇÕES & GOVERNANÇA",
      subtitle: ">>> CONTROLE DINÂMICO DE POLÍTICAS DLP, TENANTS, WEBHOOKS E PROXY EM TEMPO REAL",
      tabPolicies: "01 // POLÍTICAS DLP",
      tabTenants: "02 // MULTITENANCY & QUOTAS",
      tabWebhooks: "03 // WEBHOOKS & AUDITORIA",
      tabProxy: "04 // PROXY & AGENTES",
      saveChanges: "SALVAR ALTERAÇÕES",
      saving: "SALVANDO...",
      savedSuccess: "CONFIGURAÇÕES ATUALIZADAS COM SUCESSO!",
      testWebhook: "TESTAR DISPARO",
      testingWebhook: "ENVIANDO PING...",
      webhookSuccess: "WEBHOOK DE TESTE ENTREGUE COM SUCESSO (200 OK)!",
      webhookFailed: "FALHA NO DISPARO DO WEBHOOK: ",
      addTenant: "ADICIONAR NOVO TENANT",
      tenantId: "ID DO TENANT",
      tenantName: "NOME DO TENANT",
      proxyKey: "CHAVE DE PROXY",
      rateLimit: "LIMITE (REQ/MIN)",
      actions: "AÇÕES",
      edit: "EDITAR",
      editTenant: "EDITAR TENANT",
      saveTenant: "SALVAR ALTERAÇÕES",
      cancel: "CANCELAR",
      delete: "EXCLUIR",
      copyUrl: "COPIAR URL",
      copyGlobalUrl: "COPIAR URL GLOBAL",
      globalProxyTitle: "PROXY GLOBAL TRANSPARENTE",
      globalProxyDesc: "URL base segura para BYOK transparente. A chave original permanece no header do SDK e identifica o tenant por fingerprint HMAC.",
      rotateKey: "ROTACIONAR",
      urlCopied: "URL DO PROXY COPIADA!",
      inheritGlobal: "HERDAR (GLOBAL)",
      customPolicies: "POLÍTICAS DLP",
      overrideDesc: "Entidades com 'HERDAR' seguem a regra global. Selecione uma ação para forçar um comportamento exclusivo para este tenant.",
      numericIdHelp: "ID numérico sequencial primário (imutável)",
      generateKey: "GERAR CHAVE",
      unlimited: "ILIMITADO",
      entityColumn: "ENTIDADE / CATEGORIA",
      policyColumn: "AÇÃO APLICADA",
      mask: "MASCARAR (MASK)",
      block: "BLOQUEAR (BLOCK)",
      routeLocal: "ROTEAR LOCAL",
      disabled: "DESATIVAR DLP",
      disabledWarning: "ATENÇÃO: DESATIVAR ignora detecção, tokenização, métricas e bloqueio. Dados dessa categoria podem seguir em texto puro ao provedor.",
      allFamilies: "TODAS AS FAMÍLIAS",
      familyBatchAction: "AÇÕES EM LOTE",
      setAllMask: "TODOS MASK",
      setAllBlock: "TODOS BLOCK",
      setAllRouteLocal: "TODOS LOCAL",
      setAllDisabled: "TODOS SEM DLP",
      familyConfidence: "CONFIANÇA MÍNIMA (PESO DO LOTE)",
      applyToAllInGroup: "APLICAR A TODAS DO GRUPO",
      customThreshold: "SCORE MÍNIMO",
      defaultThreshold: "HERDADO",
      anvisaCatalog: "CATÁLOGO FARMACOLÓGICO ANVISA",
      checkAnvisaSync: "VERIFICAR ATUALIZAÇÕES ANVISA",
      checkingAnvisaSync: "VERIFICANDO...",
      anvisaUpToDate: "Catálogo ANVISA já está na versão mais recente!",
      egressScanTitle: "DLP DE SAÍDA (EGRESS SCANNING)",
      egressScanDesc: "Inspeciona bidirecionalmente respostas do LLM e mascara dados sensíveis ou credenciais não solicitadas.",
      ollamaUrlTitle: "URL BASE DE MODELOS LOCAIS / OLLAMA / AGENTES",
      corsOriginsTitle: "ORIGENS PERMITIDAS PARA CORS (SEPARADAS POR VÍRGULA)",
      providerCredentials: "CREDENCIAIS BYOK TRANSPARENTES",
      providerCredentialDesc: "A chave é usada apenas para calcular um fingerprint HMAC e nunca é persistida. Depois, altere somente a base_url do SDK.",
      provider: "PROVEDOR",
      credentialLabel: "RÓTULO",
      providerKey: "CHAVE ORIGINAL DO PROVEDOR",
      registerCredential: "REGISTRAR FINGERPRINT",
      noCredentials: "NENHUMA CREDENCIAL BYOK REGISTRADA",
      revokeCredential: "REVOGAR CREDENCIAL",
      secretConfigured: "SEGREDO CONFIGURADO // WRITE-ONLY",
      secretNotConfigured: "SEGREDO NÃO CONFIGURADO",
      secretUnchanged: "CONFIGURADO — DEIXE VAZIO PARA MANTER",
      newSecretPlaceholder: "INFORME UM NOVO SEGREDO",
      keyCopiedOnce: "NOVA CHAVE COPIADA — VISÍVEL SOMENTE AGORA",
    },
  },
  en: {
    common: {
      telemetry: "TELEMETRY",
      playground: "PLAYGROUND",
      sdkIntegration: "SDK_INTEGRATION",
      settings: "SETTINGS & POLICIES",
      light: "LIGHT",
      dark: "DARK",
      exit: "EXIT",
      refresh: "REFRESH",
      purge: "PURGE LOGS",
      emptyPayload: "[ EMPTY_PAYLOAD ]",
      loading: "LOADING...",
      clear: "CLEAR",
      page: "PAGE",
      of: "OF",
      previous: "PREV",
      next: "NEXT",
      rowsPerPage: "ROWS PER PAGE",
      all: "ALL",
    },
    login: {
      clearanceRequired: "HADESGATE™ // SECURITY CLEARANCE REQUIRED",
      authUnit: "[ AUTH_UNIT // 00 ]",
      zeroLeakShield: "ZERO-LEAK SHIELD",
      subheading: ">>> REAL-TIME DATA ANONYMIZATION GATEWAY",
      gatewayEndpoint: "01 // GATEWAY ENDPOINT",
      adminKey: "02 // ADMIN ACCESS KEY",
      enterConsole: "ENTER TELEMETRY CONSOLE",
      authenticating: "AUTHENTICATING...",
      invalidCredentials: "INVALID CREDENTIALS FOR THIS HADESGATE UNIT.",
      connectionError: "GATEWAY CONNECTION FAILED. CHECK TARGET URL.",
      deterministicEngine: "DETERMINISTIC ENGINE // GO CGO=0",
      hologramUnit: "[ PRIVACY SHIELD // ZERO-PII ]",
      hologramStatus: "INFERENCE PROTECTION",
      hologramProtocol: "DETERMINISTIC IN-MEMORY VAULT",
    },
    observability: {
      kpiRequests: "REQUESTS",
      kpiMasked: "MASKED",
      kpiScanTime: "SCAN TIME",
      kpiRate: "RATE",
      totalLabel: "total interceptions",
      securedRatio: "secured data points",
      microsecAvg: "µs deterministic avg",
      perHour: "/hour real-time",
      providerMatrix: "PROVIDER_MATRIX >>>",
      searchPlaceholder: "FILTER BY PROVIDER, MODEL OR ENTITY...",
      liveStream: "LIVE STREAM",
      noTracesTitle: "NO TELEMETRY TRACES RECORDED",
      noTracesDesc: "Send requests via the gateway proxy or run tests in the Playground to inspect anonymization metrics in real time.",
      thTime: "TIME",
      thProviderModel: "PROVIDER / MODEL",
      thLang: "LANG",
      thPiiDetected: "PII DETECTED",
      thSecrets: "SECRETS",
      thAction: "ACTION",
      viewPrompt: "PROMPT",
      maskedPromptTitle: ">>> MASKED PROMPT // SANITIZED PAYLOAD",
      sanitizedPayloadSub: "(Zero raw sensitive data sent upstream or persisted to disk)",
      detectedTypes: "DETECTED TYPES",
      traceLatency: "GATEWAY OVERHEAD",
      status: "STATUS",
      paginationShowing: "SHOWING",
    },
    playground: {
      title: "LIVE ANONYMIZATION SANDBOX",
      subtitle: ">>> TEST DETERMINISTIC MASKING & REAL-TIME SSE DETOKENIZATION",
      inputPayload: "INPUT_PAYLOAD // RAW_USER_PROMPT",
      targetModel: "TARGET_MODEL:",
      inputPlaceholder: "ENTER TEXT WITH CPF, CNPJ, IBAN, EMAIL, PASSWORD OR API KEY...",
      executeInterception: "EXECUTE INTERCEPTION",
      intercepting: "INTERCEPTING...",
      maskedPayloadTitle: "01 // MASKED PAYLOAD (SENT TO UPSTREAM)",
      reconstitutedStreamTitle: "02 // RECONSTITUTED SSE STREAM (CLIENT)",
      scanTime: "SCAN_TIME:",
      securedItems: "SECURED_ITEMS:",
      detectedTags: "DETECTED_TAGS",
      rawUpstreamWire: "RAW_UPSTREAM_WIRE (LLM >>> HADESGATE):",
    },
    codeSnippets: {
      title: "SDK INTEGRATION // QUICKSTART",
      subtitle: ">>> DIRECT REVERSE PROXY CONNECTION WITH HADESGATE",
      tabPython: "PYTHON (OPENAI SDK)",
      tabGemini: "GEMINI SDK",
      tabNode: "NODE.JS / TYPESCRIPT",
      tabCurl: "CURL CLI",
      copy: "COPY",
      copied: "COPIED!",
    },
    settings: {
      title: "CONFIGURATION & GOVERNANCE CONSOLE",
      subtitle: ">>> REAL-TIME DYNAMIC CONTROL OF DLP POLICIES, TENANTS, WEBHOOKS AND PROXY",
      tabPolicies: "01 // DLP POLICIES",
      tabTenants: "02 // MULTITENANCY & QUOTAS",
      tabWebhooks: "03 // WEBHOOKS & AUDIT",
      tabProxy: "04 // PROXY & AGENTS",
      saveChanges: "SAVE CHANGES",
      saving: "SAVING...",
      savedSuccess: "CONFIGURATION UPDATED SUCCESSFULLY!",
      testWebhook: "TEST DISPATCH",
      testingWebhook: "SENDING PING...",
      webhookSuccess: "TEST WEBHOOK DELIVERED SUCCESSFULLY (200 OK)!",
      webhookFailed: "WEBHOOK DELIVERY FAILED: ",
      addTenant: "ADD NEW TENANT",
      tenantId: "TENANT ID",
      tenantName: "TENANT NAME",
      proxyKey: "PROXY KEY",
      rateLimit: "LIMIT (REQ/MIN)",
      actions: "ACTIONS",
      edit: "EDIT",
      editTenant: "EDIT TENANT",
      saveTenant: "SAVE CHANGES",
      cancel: "CANCEL",
      delete: "DELETE",
      copyUrl: "COPY URL",
      copyGlobalUrl: "COPY GLOBAL URL",
      globalProxyTitle: "TRANSPARENT GLOBAL PROXY",
      globalProxyDesc: "Safe base URL for transparent BYOK proxying. The original provider key stays in the SDK header and identifies the tenant by HMAC fingerprint.",
      rotateKey: "ROTATE",
      urlCopied: "PROXY URL COPIED TO CLIPBOARD!",
      inheritGlobal: "INHERIT (GLOBAL)",
      customPolicies: "DLP POLICIES",
      overrideDesc: "Entities set to 'INHERIT' use the global policy. Choose an action to enforce tenant-specific behavior.",
      numericIdHelp: "Primary sequential numeric ID (immutable)",
      generateKey: "GENERATE KEY",
      unlimited: "UNLIMITED",
      entityColumn: "ENTITY / CATEGORY",
      policyColumn: "APPLIED ACTION",
      mask: "MASK",
      block: "BLOCK",
      routeLocal: "ROUTE LOCAL",
      disabled: "DISABLE DLP",
      disabledWarning: "WARNING: DISABLE skips detection, tokenization, metrics, and blocking. Data in this category may reach the provider as plaintext.",
      allFamilies: "ALL FAMILIES",
      familyBatchAction: "BATCH ACTIONS",
      setAllMask: "ALL MASK",
      setAllBlock: "ALL BLOCK",
      setAllRouteLocal: "ALL LOCAL",
      setAllDisabled: "ALL NO DLP",
      familyConfidence: "MIN CONFIDENCE (BATCH WEIGHT)",
      applyToAllInGroup: "APPLY TO ALL IN GROUP",
      customThreshold: "MIN SCORE",
      defaultThreshold: "INHERITED",
      anvisaCatalog: "ANVISA PHARMACOLOGICAL CATALOG",
      checkAnvisaSync: "CHECK ANVISA UPDATES",
      checkingAnvisaSync: "CHECKING...",
      anvisaUpToDate: "ANVISA catalog is already up to date!",
      egressScanTitle: "EGRESS DLP SCANNING",
      egressScanDesc: "Bidirectionally inspects LLM completions and masks new unmapped sensitive data or hallucinated credentials.",
      ollamaUrlTitle: "LOCAL MODELS / OLLAMA / AGENTS BASE URL",
      corsOriginsTitle: "CORS ALLOWED ORIGINS (COMMA-SEPARATED)",
      providerCredentials: "TRANSPARENT BYOK CREDENTIALS",
      providerCredentialDesc: "The key is used only to calculate an HMAC fingerprint and is never persisted. Afterwards, change only the SDK base_url.",
      provider: "PROVIDER",
      credentialLabel: "LABEL",
      providerKey: "ORIGINAL PROVIDER KEY",
      registerCredential: "REGISTER FINGERPRINT",
      noCredentials: "NO BYOK CREDENTIALS REGISTERED",
      revokeCredential: "REVOKE CREDENTIAL",
      secretConfigured: "SECRET CONFIGURED // WRITE-ONLY",
      secretNotConfigured: "SECRET NOT CONFIGURED",
      secretUnchanged: "CONFIGURED — LEAVE BLANK TO KEEP",
      newSecretPlaceholder: "ENTER A NEW SECRET",
      keyCopiedOnce: "NEW KEY COPIED — VISIBLE ONLY NOW",
    },
  },
};

