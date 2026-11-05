package anonymizer

import (
	"testing"
	"time"

	"github.com/hadesgate/hadesgate/internal/anonymizer/vault"
	"github.com/hadesgate/hadesgate/internal/stream"
)

func BenchmarkAnonymize(b *testing.B) {
	v := vault.NewMemoryVault(5 * time.Minute)
	defer v.Close()

	cfg := DefaultConfig()
	engine := NewEngine(cfg, v)

	text := "Envie um e-mail para João da Silva, CPF 529.982.247-25, cobrando R$ 500. Contato: joao.silva@empresa.com.br ou celular (11) 98765-4321."

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		engine.Anonymize("bench-sess", text)
	}
}

func BenchmarkStreamingDetokenizer(b *testing.B) {
	v := vault.NewMemoryVault(5 * time.Minute)
	defer v.Close()

	sessionID := "bench-stream"
	v.Put(sessionID, "[PERSON_1]", "João da Silva", "PERSON", 5*time.Minute)
	v.Put(sessionID, "[CPF_1]", "529.982.247-25", "CPF", 5*time.Minute)

	chunks := []string{
		"Olá ",
		"[PER",
		"SON_1], seu C",
		"PF é [",
		"CPF_1]. Obrigado!",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		detok := stream.NewDetokenizer(sessionID, v)
		for _, chunk := range chunks {
			_ = detok.Feed(chunk)
		}
		_ = detok.Flush()
	}
}
