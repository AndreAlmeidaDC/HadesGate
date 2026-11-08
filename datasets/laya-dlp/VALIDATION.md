# Validação — 08/11/2026

```text
node datasets/laya-dlp/generate.mjs --check
OK: 281 linhas, 33 categorias; cobertura positiva/negativa e offsets UTF-8 validados.

CGO_ENABLED=0 go vet ./cmd/... ./internal/...
PASS (exit 0)

CGO_ENABLED=0 go test -count=1 ./cmd/... ./internal/...
PASS — todos os pacotes

CGO_ENABLED=0 go run ./cmd/pii-benchmark -min-precision 0.90 -min-recall 0.90
cases: 135; true_positives: 96; false_positives: 0; false_negatives: 0
precision: 1; recall: 1; failed_cases: []; failed_entity_types: []

cd frontend && npx.cmd --no-install tsc --noEmit
PASS (exit 0)
```

`make quality` não pôde ser invocado: `make` não está instalado neste Windows.
Os comandos equivalentes foram executados diretamente em PowerShell, com
`CGO_ENABLED=0` para Go, sem carregar `.env`. As métricas acima pertencem ao
gate determinístico existente, **não** a um modelo Laya treinado. O treinamento
e a avaliação independente do Laya não foram executados.
