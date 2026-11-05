package stream

import (
	"strings"
	"testing"
	"time"

	"github.com/hadesgate/hadesgate/internal/anonymizer/vault"
)

func TestStreamDetokenizer_SplitTokenReconciliation(t *testing.T) {
	v := vault.NewMemoryVault(5 * time.Minute)
	defer v.Close()

	sessionID := "test-session"
	v.Put(sessionID, "[PERSON_1]", "João da Silva", "PERSON", 5*time.Minute)
	v.Put(sessionID, "[CPF_1]", "529.982.247-25", "CPF", 5*time.Minute)

	detok := NewDetokenizer(sessionID, v)

	// Scenario: LLM emits tokens split across chunks
	// Chunk 1: "Olá "
	// Chunk 2: "[PER" (split here!)
	// Chunk 3: "SON_1], seu C"
	// Chunk 4: "PF é [" (split here!)
	// Chunk 5: "CPF_1]. Obrigado!"

	chunks := []string{
		"Olá ",
		"[PER",
		"SON_1], seu C",
		"PF é [",
		"CPF_1]. Obrigado!",
	}

	var emitted []string
	for _, chunk := range chunks {
		out := detok.Feed(chunk)
		if out != "" {
			emitted = append(emitted, out)
		}
	}
	if remaining := detok.Flush(); remaining != "" {
		emitted = append(emitted, remaining)
	}

	fullOutput := strings.Join(emitted, "")
	expected := "Olá João da Silva, seu CPF é 529.982.247-25. Obrigado!"

	if fullOutput != expected {
		t.Fatalf("Stream detokenization failed!\nExpected: %q\nGot:      %q", expected, fullOutput)
	}

	// Verify that neither [PER nor [PERSON_1] was leaked directly
	if strings.Contains(fullOutput, "[PER") || strings.Contains(fullOutput, "[CPF") {
		t.Fatalf("Leaked surrogate token in stream output: %s", fullOutput)
	}
}

func TestStreamDetokenizer_RegularBrackets(t *testing.T) {
	v := vault.NewMemoryVault(5 * time.Minute)
	defer v.Close()

	sessionID := "test-brackets"
	detok := NewDetokenizer(sessionID, v)

	// Text with regular brackets that are NOT tokens
	chunks := []string{
		"Confira os dados em [item 1] ",
		"e [item 2].",
	}

	var emitted []string
	for _, chunk := range chunks {
		out := detok.Feed(chunk)
		if out != "" {
			emitted = append(emitted, out)
		}
	}
	if remaining := detok.Flush(); remaining != "" {
		emitted = append(emitted, remaining)
	}

	fullOutput := strings.Join(emitted, "")
	expected := "Confira os dados em [item 1] e [item 2]."

	if fullOutput != expected {
		t.Fatalf("Expected: %q, Got: %q", expected, fullOutput)
	}
}

func TestStreamDetokenizer_ResilientLLMTokenMutations(t *testing.T) {
	v := vault.NewMemoryVault(5 * time.Minute)
	defer v.Close()

	sessionID := "test-resilient"
	v.Put(sessionID, "[HG_a1b2c3_CPF_1]", "529.982.247-25", "CPF", 5*time.Minute)
	v.Put(sessionID, "[HG_a1b2c3_EMAIL_1]", "maria@empresa.com", "EMAIL", 5*time.Minute)

	detok := NewDetokenizer(sessionID, v)

	// LLM mutates case, adds double brackets, or strips the session namespace
	chunks := []string{
		"Dados: [[HG_a1b2c3_",
		"CPF_1]], minúsculo [hg_a1b2c3_cpf_1] e abreviado [HG_EMAIL_1] / [EMAIL_1].",
	}

	var emitted []string
	for _, chunk := range chunks {
		if out := detok.Feed(chunk); out != "" {
			emitted = append(emitted, out)
		}
	}
	if rem := detok.Flush(); rem != "" {
		emitted = append(emitted, rem)
	}

	got := strings.Join(emitted, "")
	expected := "Dados: 529.982.247-25, minúsculo 529.982.247-25 e abreviado maria@empresa.com / maria@empresa.com."
	if got != expected {
		t.Fatalf("Resilient detokenization mismatch!\nExpected: %q\nGot:      %q", expected, got)
	}
}
