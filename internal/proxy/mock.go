package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/hadesgate/hadesgate/internal/stream"
)

// MockLLM simulates an upstream LLM, intentionally emitting chunks with split tokens
// to demonstrate the sliding-window detokenizer in action.
type MockLLM struct{}

// NewMockLLM creates a new MockLLM instance.
func NewMockLLM() *MockLLM {
	return &MockLLM{}
}

// GenerateStream writes raw simulated upstream SSE chunks (with split tokens) to an io.Writer.
func (m *MockLLM) GenerateStream(w io.Writer, reqID, model, prompt string, tokens map[string]string) {
	// Construct an answer echoing the surrogate tokens
	var tokenMentions []string
	for tok := range tokens {
		tokenMentions = append(tokenMentions, tok)
	}

	intro := "Recebido com sucesso! "
	if len(tokenMentions) > 0 {
		intro += fmt.Sprintf("Processamos os dados do cliente: %s. ", strings.Join(tokenMentions, " e "))
	}
	outro := "A requisição foi devidamente registrada sob os protocolos de conformidade LGPD da empresa."

	fullResponse := intro + outro

	// Break into small fragments, intentionally splitting tokens in half to test chunk boundary reconciliation!
	fragments := splitWithTokenSplits(fullResponse)

	now := time.Now().Unix()
	for i, frag := range fragments {
		chunk := stream.ChatCompletionChunk{
			ID:      fmt.Sprintf("chatcmpl-mock-%s", reqID),
			Object:  "chat.completion.chunk",
			Created: now,
			Model:   model,
			Choices: []stream.Choice{
				{
					Index: 0,
					Delta: stream.Delta{
						Content: frag,
					},
					FinishReason: nil,
				},
			},
		}

		if i == len(fragments)-1 {
			stop := "stop"
			chunk.Choices[0].FinishReason = &stop
		}

		b, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "data: %s\n\n", string(b))

		// Micro delay to simulate streaming tokens
		time.Sleep(30 * time.Millisecond)
	}

	// Send terminal event
	fmt.Fprintf(w, "data: [DONE]\n\n")
}

// splitWithTokenSplits slices the text into small 3-7 character chunks, intentionally splitting tokens.
func splitWithTokenSplits(text string) []string {
	var chunks []string
	runes := []rune(text)
	chunkSize := 4

	for i := 0; i < len(runes); i += chunkSize {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))
	}
	return chunks
}
