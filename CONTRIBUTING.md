# Contribuindo com o HadesGate

Este guia cobre o ambiente de desenvolvimento local (Windows, Linux e macOS), criação de novos detectores de PII/segredos e uso de agentes de IA para colaborar no projeto. Para deploy em produção, consulte o [README.md](README.md).

Ao enviar uma contribuição, você concorda em disponibilizá-la sob a mesma licença do projeto, **GNU AGPL v3.0 somente** (`AGPL-3.0-only`). Consulte [`LICENSE`](LICENSE).

## Requisitos

- **Go 1.26** ou superior. **Não é preciso compilador C (`CGO_ENABLED=0`)**: o HadesGate usa SQLite em Go puro (`modernc.org/sqlite`).
- **Node.js 20** ou superior e `npm` (Node 22+ recomendado caso queira rodar a suíte de testes dos hooks em `.claude/`).
- **[air](https://github.com/air-verse/air) v1.65.0+**: `go install github.com/air-verse/air@v1.65.0`.
- **GNU Make** com shell POSIX. Linux e macOS já possuem nativamente. No Windows, execute pelo **Git Bash** ou adicione `C:\Program Files\Git\usr\bin` ao seu `PATH` (sem isso o `make` invoca `cmd.exe` e comandos POSIX falham).
- **Docker**, opcional, para desenvolvimento em container (`make dev-docker`) e para validar a imagem final de produção (`make up`).

---

## Subir rapidamente com Hot Reload (`make dev`)

```bash
cp .env.example .env
make dev
```

`make dev` valida a presença do `.env` e inicia dois processos em paralelo:

| Processo | Porta | Recarrega quando | Log gravado em |
|---|---|---|---|
| Gateway Go (`air`) | `8080` | qualquer `.go` muda | `.devlogs/server.log` |
| Console Next.js (`next dev`) | `3000` | qualquer arquivo em `frontend/` muda | `.devlogs/web.log` |

Abra **http://localhost:3000**. Quando aberto na porta `3000`, o painel já preenche automaticamente o endpoint do gateway como `http://localhost:8080`. Basta autenticar com a chave configurada no seu `.env`.

- Na primeira execução, o `Makefile` instala automaticamente as dependências em `frontend/` (`npm ci`).
- Para subir apenas um dos lados: `make server` (só Go) ou `make web` (só Next.js).
- Para listar os logs de desenvolvimento: `make dev-logs`.

---

## Servidor em Container (sem Go instalado no host)

```bash
make dev-docker   # terminal 1: gateway Go com air em container
make web          # terminal 2: painel Next.js local com hot reload
```

O container usa `Dockerfile.dev` + `.air.docker.toml` e recompila automaticamente a cada alteração em `.go`. O banco SQLite de desenvolvimento fica isolado em um volume Docker (`dev-data`). Para encerrar, rode `make dev-docker-down`.

---

## Testes, Benchmark de Acurácia e Qualidade

Antes de abrir um Pull Request, garanta que o gate completo passe:

```bash
make quality
```

Ou execute cada etapa individualmente:

| Comando | O que valida |
|---|---|
| `make lint` | `CGO_ENABLED=0 go vet ./...` + `npx tsc --noEmit` no `frontend/` |
| `make test` | `CGO_ENABLED=0 go test -count=1 ./cmd/... ./internal/...` |
| `make test-race` | `go test -race -count=1 ./cmd/... ./internal/...` |
| `make accuracy` | Gate `cmd/pii-benchmark` exigindo **precisão ≥ 90%** e **recall ≥ 90%** no agregado e por tipo, além de mínimos de 100 casos, 25 controles negativos e 20 tipos |
| `make bench` | Microbenchmarks de latência e alocações (`-benchmem`) do motor de anonimização |

---

## Como adicionar ou melhorar um Detector de PII / Segredo

1. Crie ou edite o detector em `internal/anonymizer/detector/<nome>.go` priorizando validação determinística (checksum matemático como Módulo 11, Luhn, Mod-97 ou entropia/prefixo exato) para manter latência sub-milissegundo e baixo falso positivo.
2. Adicione testes unitários cobrindo casos válidos, inválidos e bordas em `internal/anonymizer/detector/<nome>_test.go`.
3. Registre o detector em `internal/anonymizer/engine.go` e sua flag de configuração em `internal/config/config.go` (se aplicável), além da classificação em `internal/observability/`.
4. Adicione exemplos positivos, negativos e de colisão em `benchmarks/pii-accuracy/corpus.json`, com IDs únicos e os campos opcionais `language`/`tags`, e rode `make accuracy` para comprovar que precisão e recall permanecem ≥ `0.90` globalmente e em cada tipo.

---

## Atalhos de Produção (Docker Compose)

| Alvo | Equivale a |
|---|---|
| `make build` | `docker compose build` |
| `make up` | `docker compose up -d --build` |
| `make down` | `docker compose down` |
| `make logs` | `docker compose logs -f hadesgate` |
| `make sh` | `docker compose exec hadesgate sh` |
| `make clean` | `docker compose down -v` (remove volumes locais) |

> **Atenção à porta `8080`:** se o container de produção (`make up`) estiver rodando na porta `8080`, pare-o com `make down` antes de iniciar `make dev` ou `make server`.

---

## Desenvolvimento com Agentes de IA

O HadesGate é preparado para colaboração assistida por agentes de IA (Antigravity, Claude Code, Cursor, Codex, Windsurf, GitHub Copilot):

- **Regras universais:** [`AGENTS.md`](AGENTS.md) na raiz e [`frontend/AGENTS.md`](frontend/AGENTS.md) para o painel web, além da skill visual em [`.agents/skills/industrial-brutalist-ui/SKILL.md`](.agents/skills/industrial-brutalist-ui/SKILL.md).
- **Claude Code:** [`CLAUDE.md`](CLAUDE.md) importa `@AGENTS.md` e `.claude/settings.json` ativa os hooks automáticos do projeto:
  - Ao abrir na raiz do repositório, o contexto recebe `[harness hadesgate] ativo.`.
  - Ao finalizar um turno com edições em `.go`, no corpus de benchmark ou em `frontend/`, o hook `.claude/hooks/verify.mjs` executa automaticamente `gofmt`, `go vet`, `go test`, `pii-benchmark` e `tsc --noEmit`.
  - Para testar o próprio harness localmente (requer Node 22+): `node --test ".claude/**/*.test.mjs"`.
  - Para desativar os hooks localmente na sua máquina sem afetar o repositório, adicione `"disableAllHooks": true` em `.claude/settings.local.json`.

