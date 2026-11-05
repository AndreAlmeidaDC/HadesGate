package stream

import (
	"strings"

	"github.com/hadesgate/hadesgate/internal/anonymizer/vault"
)

// Detokenizer processes a stream of text fragments (or SSE deltas), replacing surrogate tokens
// with their original sensitive values from the vault, correctly handling tokens that are split
// across chunk boundaries.
type Detokenizer struct {
	sessionID   string
	vault       vault.Vault
	pending     string // holds potential incomplete token prefix (e.g., "[PER")
	maxTokenLen int
}

// NewDetokenizer creates a streaming detokenizer for a given session.
func NewDetokenizer(sessionID string, v vault.Vault) *Detokenizer {
	return &Detokenizer{
		sessionID:   sessionID,
		vault:       v,
		pending:     "",
		maxTokenLen: 64, // includes the per-request namespace used by surrogate tokens
	}
}

// Feed receives a new text fragment, reconciles any split tokens from previous chunks,
// replaces completed tokens with plaintext, and returns the portion that is safe to emit immediately.
func (d *Detokenizer) Feed(chunk string) string {
	if chunk == "" && d.pending == "" {
		return ""
	}

	combined := d.pending + chunk
	d.pending = ""

	// Check if there is an open bracket '[' that could be the start of a token
	lastOpen := strings.LastIndex(combined, "[")
	lastClose := strings.LastIndex(combined, "]")

	var toProcess string
	if lastOpen != -1 && lastOpen > lastClose {
		// Include leading '[' if the model emitted double brackets "[[..."
		for lastOpen > 0 && combined[lastOpen-1] == '[' {
			lastOpen--
		}
		// Hold potential incomplete token in pending, including a trailing ']' if double-bracketed next chunk
		candidateLen := len(combined) - lastOpen
		if candidateLen <= d.maxTokenLen {
			// Hold candidate in pending for next chunk
			toProcess = combined[:lastOpen]
			d.pending = combined[lastOpen:]
		} else {
			// Exceeded max token length, it's just plain text containing a '['
			toProcess = combined
		}
	} else {
		toProcess = combined
	}

	if toProcess == "" {
		return ""
	}

	// Detokenize the safe portion
	return d.replaceTokens(toProcess)
}

// Flush flushes any remaining buffered text when the stream terminates.
func (d *Detokenizer) Flush() string {
	if d.pending == "" {
		return ""
	}
	remaining := d.pending
	d.pending = ""
	return d.replaceTokens(remaining)
}

// Restore replaces complete placeholders without changing streaming state.
func (d *Detokenizer) Restore(text string) string {
	return d.replaceTokens(text)
}

// Clone creates an independent streaming buffer over the same request mappings.
func (d *Detokenizer) Clone() *Detokenizer {
	return NewDetokenizer(d.sessionID, d.vault)
}

// GetKnownPlaintexts returns a lookup map of all plaintexts restored for this session.
func (d *Detokenizer) GetKnownPlaintexts() map[string]bool {
	if d.vault == nil || d.sessionID == "" {
		return nil
	}
	mappings := d.vault.GetAllSessionMappings(d.sessionID)
	known := make(map[string]bool, len(mappings))
	for _, val := range mappings {
		known[val] = true
	}
	return known
}

// replaceTokens replaces all surrogate tokens in the string with their real plaintext values,
// including resilient matching for LLM-mutated placeholders (case changes, double brackets, or stripped namespaces).
func (d *Detokenizer) replaceTokens(text string) string {
	if !strings.Contains(text, "[") {
		return text
	}
	mappings := d.vault.GetAllSessionMappings(d.sessionID)
	if len(mappings) == 0 {
		return text
	}

	res := text
	for token, plainText := range mappings {
		// 1. Double-bracketed variant: [[PG_ns_TYPE_1]] -> plainText
		doubleTok := "[" + token + "]"
		if strings.Contains(res, doubleTok) {
			res = strings.ReplaceAll(res, doubleTok, plainText)
		}
		// 2. Exact match: [PG_ns_TYPE_1] -> plainText
		if strings.Contains(res, token) {
			res = strings.ReplaceAll(res, token, plainText)
		}
		// 3. Case-insensitive fallback if the LLM lowercased or altered token casing
		if strings.Contains(res, "[") {
			res = replaceCaseInsensitive(res, doubleTok, plainText)
			res = replaceCaseInsensitive(res, token, plainText)
		}
		// 4. Namespace-stripped fallback (e.g., [HG_CPF_1], [PG_CPF_1] or [CPF_1]) if LLM abbreviated the placeholder
		if strings.Contains(res, "[") && (strings.HasPrefix(token, "[HG_") || strings.HasPrefix(token, "[PG_")) && strings.HasSuffix(token, "]") {
			prefix := token[:4]              // "[HG_" or "[PG_"
			inner := token[4 : len(token)-1] // "<ns>_<TYPE>_<idx>"
			if sep := strings.IndexByte(inner, '_'); sep > 0 {
				shortTypeIdx := inner[sep+1:] // "<TYPE>_<idx>"
				shortPrefixed := prefix + shortTypeIdx + "]"
				shortBare := "[" + shortTypeIdx + "]"
				if _, exists := mappings[shortPrefixed]; !exists {
					res = replaceCaseInsensitive(res, "["+shortPrefixed+"]", plainText)
					res = replaceCaseInsensitive(res, shortPrefixed, plainText)
				}
				if _, exists := mappings[shortBare]; !exists {
					res = replaceCaseInsensitive(res, "["+shortBare+"]", plainText)
					res = replaceCaseInsensitive(res, shortBare, plainText)
				}
			}
		}
	}
	return res
}

func replaceCaseInsensitive(haystack, needle, replacement string) string {
	if needle == "" || len(haystack) < len(needle) {
		return haystack
	}
	lowerHaystack := strings.ToLower(haystack)
	lowerNeedle := strings.ToLower(needle)
	idx := strings.Index(lowerHaystack, lowerNeedle)
	if idx == -1 {
		return haystack
	}

	var sb strings.Builder
	sb.Grow(len(haystack))
	cursor := 0
	for idx != -1 {
		sb.WriteString(haystack[cursor : cursor+idx])
		sb.WriteString(replacement)
		cursor += idx + len(needle)
		if cursor >= len(lowerHaystack) {
			break
		}
		idx = strings.Index(lowerHaystack[cursor:], lowerNeedle)
	}
	if cursor < len(haystack) {
		sb.WriteString(haystack[cursor:])
	}
	return sb.String()
}
