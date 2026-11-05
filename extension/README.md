# HadesGate™ Browser Shield (WebExtensions API — Manifest V3)

Extensão de navegador *Zero-PII* (Chrome, Brave, Edge, Arc, Vivaldi, Firefox e derivados Gecko) que conecta os chats web de IA (**ChatGPT**, **Claude.ai**, **Google Gemini**, **AI Studio**, **DeepSeek**, **Perplexity**, **Grok** e **GitHub Copilot**) ao seu motor **HadesGate** local (`http://localhost:8080`) ou remoto.

---

## O que ela faz

1. **Interceptação de Colagem (`Ctrl+V`):** ao colar qualquer texto no campo de mensagem da IA, a extensão consulta o HadesGate em microssegundos, substitui dados sensíveis (CPF, CNPJ, PIX, contas, cartões, IBAN, e-mails, telefones, nomes, API keys e senhas) por tokens `[HG_<ns>_<TYPE>_<N>]` e guarda o mapeamento apenas na RAM da aba.
2. **Botão Tático no Chat (`Alt+H`):** barra flutuante *Industrial Brutalism* no canto inferior direito do chat (`[ HADESGATE // MASCARAR (ALT+H) ]`) para anonimizar o texto digitado antes de enviar.
3. **Diretiva Anti-Recusa Automática:** anexa automaticamente no primeiro prompt mascarado da conversa a diretiva que instrui o ChatGPT/Claude/Gemini a tratar os tokens `[HG_...]` como dados reais já preenchidos.
4. **Auto-Restauração na Tela (`MutationObserver`):** conforme a IA responde em streaming com os tokens `[HG_...]`, a extensão substitui os tokens de volta pelos seus dados originais diretamente no DOM da página — a nuvem nunca vê o dado real, mas você lê a resposta completa na tela.
5. **Telemetria Unificada:** todas as interceptações aparecem em tempo real na aba **Telemetria** do Console HadesGate (`http://localhost:8080`) com zero dados sensíveis gravados em disco.

---

## Como instalar

### Chrome / Brave / Edge / Opera / Arc (Chromium)

1. Abra `chrome://extensions` (ou `brave://extensions` / `edge://extensions`).
2. Ative a chave **Modo do desenvolvedor** (*Developer mode*) no canto superior direito.
3. Clique em **Carregar sem compactação** (*Load unpacked*) e selecione a pasta [`extension`](file:///c:/dev/HadesGate/extension) deste repositório (`c:\dev\HadesGate\extension`).
4. Clique no ícone do **HadesGate™** na barra do navegador:
   - **Endpoint:** `http://localhost:8080`
   - **Chave de Acesso:** sua chave configurada no `.env` do HadesGate
   - Clique em **SALVAR & VALIDAR CONEXÃO** (`ONLINE // PROTEGIDO`).

### Mozilla Firefox / LibreWolf / Tor Browser (Gecko)

1. Abra uma nova aba e digite `about:debugging#/runtime/this-firefox`.
2. Na seção **Extensões temporárias** (*Temporary Extensions*), clique em **Carregar extensão temporária...** (*Load Temporary Add-on...*).
3. Selecione o arquivo [`extension/manifest.firefox.json`](file:///c:/dev/HadesGate/extension/manifest.firefox.json) (ou o arquivo empacotado `dist/extension/hadesgate-shield-firefox.xpi`).
4. O escudo **HadesGate™** será ativado com o ID `shield@hadesgate.dev` e total compatibilidade com a WebExtensions API da Mozilla.
5. Clique no ícone do HadesGate na barra de ferramentas e valide a conexão com o endpoint local.

---

## Empacotamento para Lojas (Chrome Web Store & Firefox AMO)

Para gerar os pacotes de distribuição para as lojas oficiais de extensões:

```bash
# Gera os pacotes hadesgate-shield-chrome.zip e hadesgate-shield-firefox.xpi em dist/extension/
make extension
# ou diretamente via Go:
go run ./cmd/package-extension
```

Os artefatos gerados serão:
- `dist/extension/hadesgate-shield-chrome.zip`: pronto para upload no Chrome Web Store Developer Dashboard.
- `dist/extension/hadesgate-shield-firefox.xpi`: pronto para assinatura e publicação no Firefox Add-ons (AMO).
