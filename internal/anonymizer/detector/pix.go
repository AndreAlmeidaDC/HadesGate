package detector

import (
	"regexp"
	"strings"
)

// uuidEVPRegex matches standard UUID v4 formats used for Brazilian PIX random keys (Chave EVP).
var uuidEVPRegex = regexp.MustCompile(`\b[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\b`)

// contextualPIXPrefixRegex captures explicit PIX key assignments (e.g. "chave pix: 12345678900", "pix joao@empresa.com").
var contextualPIXPrefixRegex = regexp.MustCompile(`(?i)\b(?:chave\s+pix|pix(?:\s+copia\s+e\s+cola)?)\s*(?::|=|\sé\s|\sfoi\s)?\s*([a-zA-Z0-9._%+\-@()]+)`)

// PIXDetector identifies Brazilian Central Bank PIX payment keys.
type PIXDetector struct{}

// NewPIXDetector creates a new PIXDetector.
func NewPIXDetector() *PIXDetector {
	return &PIXDetector{}
}

// Type returns EntityPIX.
func (d *PIXDetector) Type() EntityType {
	return EntityPIX
}

// ContextWords returns keywords that provide contextual evidence for PIX keys.
func (d *PIXDetector) ContextWords() []string {
	return []string{"pix", "chave pix", "pix copia e cola", "transferencia pix", "pagamento pix", "pagar pix"}
}

// Detect scans text for PIX keys (Random EVP UUIDs and contextual keys).
func (d *PIXDetector) Detect(text string) []Match {
	var matches []Match

	// 1. Detect EVP Random Keys (UUID v4)
	evpIndices := uuidEVPRegex.FindAllStringIndex(text, -1)
	for _, idx := range evpIndices {
		val := text[idx[0]:idx[1]]
		ctxText := ExtractSurroundingContext(text, idx[0], idx[1], 5)
		ctxFound := FindContextKeywords(ctxText, d.ContextWords())

		// A UUID is only recognized as a PIX key if PIX context is present
		score := 0.40
		if len(ctxFound) > 0 {
			score = 0.95
		}

		matches = append(matches, Match{
			Type:            EntityPIX,
			Value:           val,
			Start:           idx[0],
			End:             idx[1],
			Score:           score,
			ContextFound:    ctxFound,
			ValidatorPassed: len(ctxFound) > 0,
		})
	}

	// 2. Detect explicit contextual PIX keys (e.g., "chave pix: email@exemplo.com", "pix: (11) 98765-4321")
	submatches := contextualPIXPrefixRegex.FindAllStringSubmatchIndex(text, -1)
	for _, sub := range submatches {
		if len(sub) >= 4 && sub[2] != -1 && sub[3] != -1 {
			val := strings.TrimSpace(text[sub[2]:sub[3]])
			if len(val) >= 4 {
				// Don't duplicate if already matched as EVP
				alreadyMatched := false
				for _, m := range matches {
					if m.Start == sub[2] && m.End == sub[3] {
						alreadyMatched = true
						break
					}
				}
				if !alreadyMatched {
					ctxText := ExtractSurroundingContext(text, sub[2], sub[3], 5)
					ctxFound := FindContextKeywords(ctxText, d.ContextWords())
					matches = append(matches, Match{
						Type:            EntityPIX,
						Value:           val,
						Start:           sub[2],
						End:             sub[3],
						Score:           0.95, // Explicit contextual labeling
						ContextFound:    ctxFound,
						ValidatorPassed: true,
					})
				}
			}
		}
	}

	return matches
}
