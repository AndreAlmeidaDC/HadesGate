@AGENTS.md

## Só para o Claude Code

- Abra a sessão na raiz do repositório: `.claude/settings.json`, com os hooks automáticos do projeto, só carrega a partir dali.
- Toda sessão aberta na raiz recebe uma linha que começa com `[harness hadesgate]`. Sem essa linha no contexto, avise o usuário que as travas automáticas do projeto não carregaram.
- O hook de fim de turno (`Stop`) valida automaticamente o diff da sessão (`gofmt`, `go vet`, `go test`, `pii-benchmark` quando detectores/corpus mudam e `tsc --noEmit` quando `frontend/` muda). Um bloqueio traz a saída exata da falha; corrija antes de encerrar.

