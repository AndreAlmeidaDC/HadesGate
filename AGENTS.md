# HadesGate: guia para agentes

Gateway reverso de privacidade Zero-PII em Go para APIs de LLM (OpenAI Chat Completions, OpenAI Responses, Anthropic Messages e Google Gemini). O produto compreende o binário Go (`cmd/hadesgate` e `internal/`), o gate de acurácia (`cmd/pii-benchmark` + `benchmarks/pii-accuracy/corpus.json`) e o console web (`frontend/`, Next.js com export estático embutido via `internal/web/embed.go`). Ambiente de desenvolvimento: `CONTRIBUTING.md`.

## Propriedade do código

| Caminho | Papel | Como mexer |
|---|---|---|
| `internal/anonymizer/`, `internal/anonymizer/detector/` | motor de anonimização e detectores PII/segredos | exige teste `*_test.go` + caso positivo/negativo em `benchmarks/pii-accuracy/corpus.json` mantendo precisão e recall ≥ `0.90` |
| `internal/proxy/`, `internal/protocol/`, `internal/stream/`, `internal/policy/` | gateway HTTP, extratores multi-protocolo, detokenizador SSE e políticas | exige teste in-package cobrindo streaming e não-streaming |
| `internal/observability/`, `internal/config/`, `internal/web/server.go` | telemetria SQLite Zero-PII, env vars e API do painel | exige teste in-package; se mudar resposta de `/api/*`, alinhe `frontend/` |
| `frontend/` | console Industrial Brutalism (Next.js 14) | siga `frontend/AGENTS.md` e `.agents/skills/industrial-brutalist-ui/SKILL.md` |
| `internal/web/dist/` | artefato estático gerado pelo `next build` no `Dockerfile` | **somente leitura**; mantenha apenas `internal/web/dist/.gitkeep` versionado |

## Invariantes de segurança (Zero-PII — inegociáveis)

1. **Zero-PII em disco e logs:** jamais persista prompts originais não mascarados, valores reais de entidades ou `token_map` no SQLite (`internal/observability/trace.go`), em arquivos de log ou em respostas da API de telemetria (`/api/traces`). Apenas métricas, categorias detectadas e o `SanitizedPrompt` já tokenizado podem ser gravados.
2. **Isolamento efêmero por requisição:** tokens seguem `[HG_<ns>_<TYPE>_<N>]` com namespace aleatório (`crypto/rand`) exclusivo por requisição, armazenado apenas no `MemoryVault` em RAM e purgado obrigatoriamente ao fim da requisição (`defer DeleteSession`). Headers enviados pelo cliente nunca selecionam sessões existentes no vault.
3. **Binário estático puro Go (`CGO_ENABLED=0`):** o SQLite usa `modernc.org/sqlite`. Não adicione pacotes que exijam CGO, `gcc` ou bibliotecas nativas externas no caminho principal.
4. **Sem exposição de segredos ou variáveis internas na UI pública:** telas não autenticadas (como `frontend/components/login.tsx`) e erros HTTP públicos nunca devem exibir nomes de variáveis de ambiente sensíveis (ex.: `HADESGATE_ADMIN_KEY`) nem chaves de upstream.
5. **Comparação de credenciais em tempo constante:** toda validação de chave (`AdminKey`, `ProxyKey`, `TenantKeys`) deve usar `subtle.ConstantTimeCompare`.

## Antes de afirmar, confira

- Símbolo Go ou TypeScript: use LSP (`gopls` / TS Server) ou `go doc` para localizar definições e referências reais antes de alterar assinaturas.
- Rotas expostas: `cmd/hadesgate/main.go` e `internal/web/server.go` são a fonte da verdade.
- Detectores ativos e flags de ambiente: confira `internal/config/config.go` e `.env.example`.
- Busque com `git grep` ou `rg`; evite varrer `frontend/node_modules`, `frontend/.next`, `.air` ou `.devlogs`.

## Comandos

- `make quality` valida o estado completo de entrega: `go vet`, `go test -count=1` (`CGO_ENABLED=0`), `pii-benchmark` (`-min-precision 0.90 -min-recall 0.90`) e `npx tsc --noEmit` no `frontend/`.
- `make test-race` roda `go test -race` (requer compilador C apenas para o detector de race do Go, como ocorre no CI).
- `make bench` mede latência sub-milissegundo e alocações em `internal/anonymizer`.
- Pronto quer dizer `make quality` verde, com a saída mostrada.

## Corrigir bug

1. Reproduza o problema primeiro com um teste unitário in-package que falhe pelo sintoma (sem acesso à rede externa: use `httptest.NewServer` para simular upstreams OpenAI/Anthropic).
2. Prove que o teste falha no código da base com `node .claude/scripts/red-at-base.mjs TestNome` e que passa depois da correção.
3. Problemas que dependam de chave real de provedor externo (OpenAI/Anthropic): crie o teste com payload/headers reais capturados em `httptest`; se não for possível reproduzir localmente, informe `NÃO REPRODUZIDO` e peça validação ao usuário.

## Peça antes

- Subir `make dev`, `make server`, `air`, `make up` ou `docker compose up`: a porta `8080` frequentemente já está ocupada pelo container `hadesgate` em execução.
- Ler `.env`, arquivos `*.db` em `data/` ou logs locais com tráfego de teste do usuário.
- Alterar os limiares mínimos (`0.90`) do gate `cmd/pii-benchmark`.

## Armadilhas que geram código errado

- **Tokens quebrados entre chunks SSE:** provedores dividem `[HG_ab12cd34_PERSON_1]` em múltiplos eventos SSE (`[HG_ab12` + `cd34_PER` + `SON_1]`). Toda saída streaming deve passar pelo buffer de janela deslizante de `internal/stream/detokenizer.go`, incluindo `Flush()` no encerramento.
- **Upstream Anthropic (`internal/proxy/upstream.go`):**
  - Sempre force `Accept-Encoding: identity` na requisição upstream; sem isso, respostas com erro ou JSON comprimido chegam em gzip cru ao cliente/detokenizador.
  - Sempre remova headers de navegador (`Origin`, `Referer`, `Sec-Fetch-*`) ao encaminhar para `api.anthropic.com`; caso contrário a Anthropic rejeita requisições vindas do Playground com `401 CORS requests must set 'anthropic-dangerous-direct-browser-access' header`.
- **Diretiva `PlaceholderSystemInstruction` (`internal/protocol/extractor.go`):** quando `entities > 0`, o gateway injeta automaticamente a instrução de sistema nos 4 protocolos (`InjectChatCompletionsSystemInstruction`, `InjectResponsesSystemInstruction`, `InjectAnthropicSystemInstruction`, `InjectGeminiSystemInstruction`) para que o LLM trate `[HG_...]` como dados reais sem avisar ou recusar. Essa injeção deve ocorrer **após** capturar `SanitizedPrompt`, mantendo a telemetria limpa.
- **Deduplicação canônica:** valores equivalentes na mesma requisição (ex.: `529.982.247-25` e `52998224725`) devem receber o **mesmo** token para preservar o raciocínio lógico do modelo.
- **Embed do frontend (`internal/web/embed.go`):** `//go:embed all:dist` exige que `internal/web/dist/.gitkeep` exista na árvore local mesmo antes de rodar `next build`.

## Mudança de contrato

- Novo detector ou categoria de entidade: atualize `internal/anonymizer/detector/`, registre em `internal/anonymizer/engine.go`, classifique PII vs. Segredo em `internal/observability/`, adicione cobertura em `benchmarks/pii-accuracy/corpus.json`, documente em `.env.example` e `README.md`.
- Nova rota ou campo em `/api/*`: atualize `internal/web/server.go`, os testes em `internal/web/server_test.go` e os componentes correspondentes em `frontend/components/`.

## Glossário

- **Token / Placeholder:** marcador opaco `[HG_<namespace>_<TYPE>_<N>]` enviado ao provedor LLM (no Playground exibido visualmente de forma compacta como `[[TYPE_N]]`).
- **Vault (`MemoryVault`):** cofre volátil em RAM indexado pelo `request_id` interno; nunca toca o disco.
- **Detokenizer:** máquina de estados com janela deslizante que recompõe os valores reais no stream SSE de resposta.
- **Policy Action:** decisão por entidade/tenant (`mask`, `block` ou `route_local`).

## Estilo

- Go idiomático, formatado com `gofmt`, erros encapsulados com `fmt.Errorf("...: %w", err)`, sem panics em rotas HTTP.
- Preserve docstrings e comentários relevantes existentes.
- Commits em Conventional Commits, em inglês (`feat:`, `fix:`, `refactor:`, `test:`, `docs:`).
- Documentação do repositório e idioma padrão da interface em `pt-BR` (com suporte completo ao seletor `en` em `frontend/lib/i18n.ts`).

## Git

- **Nunca envie commits diretamente para a branch principal** (`main`, `master` ou equivalente), nem localmente nem no remoto. Todo trabalho deve ocorrer em uma branch dedicada.
- **Toda alteração deve ser entregue por Pull Request.** Abra o PR com contexto, escopo, testes executados e riscos conhecidos; não faça merge do próprio PR sem autorização explícita de um mantenedor.
- Quando não houver uma alteração pronta para revisão — por exemplo, para relatar bug, propor melhoria, registrar dúvida ou documentar bloqueio — **abra uma issue** em vez de deixar a demanda apenas em conversa informal.
- Ao concluir o trabalho, informe o link do PR ou da issue correspondente. Trabalho de colaborador sem PR ou issue não é considerado entregue.
- Traga `origin/main` antes de planejar mudanças grandes e antes de abrir PR.

## Em conflito

O código testado vence a documentação; atualize a documentação no mesmo PR ou aponte a divergência.

