package stream

import (
	"fmt"
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

func TestStreamDetokenizer_RestoresTokenAtEveryChunkBoundary(t *testing.T) {
	v := vault.NewMemoryVault(5 * time.Minute)
	defer v.Close()

	const sessionID = "test-all-boundaries"
	const token = "[HG_ab12cd34_PERSON_1]"
	const plaintext = "João da Silva"
	v.Put(sessionID, token, plaintext, "PERSON", 5*time.Minute)

	assertChunks := func(t *testing.T, chunks []string) {
		t.Helper()
		detok := NewDetokenizer(sessionID, v)
		var output strings.Builder
		for _, chunk := range chunks {
			output.WriteString(detok.Feed(chunk))
		}
		output.WriteString(detok.Flush())
		if got, want := output.String(), "antes "+plaintext+" depois"; got != want {
			t.Fatalf("chunks %q: got %q, want %q", chunks, got, want)
		}
	}

	streamText := "antes " + token + " depois"
	for cut := 1; cut < len(streamText); cut++ {
		t.Run(fmt.Sprintf("cut-%d", cut), func(t *testing.T) {
			assertChunks(t, []string{streamText[:cut], streamText[cut:]})
		})
	}

	byteChunks := make([]string, 0, len(streamText))
	for index := 0; index < len(streamText); index++ {
		byteChunks = append(byteChunks, streamText[index:index+1])
	}
	t.Run("one-byte-chunks", func(t *testing.T) {
		assertChunks(t, byteChunks)
	})
}
