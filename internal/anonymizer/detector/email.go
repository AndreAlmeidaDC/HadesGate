package detector

import (
	"regexp"
)

// emailRegex matches standard email address patterns.
var emailRegex = regexp.MustCompile(`\b[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}\b`)

// EmailDetector detects email addresses.
type EmailDetector struct{}

// NewEmailDetector creates a new instance of EmailDetector.
func NewEmailDetector() *EmailDetector {
	return &EmailDetector{}
}

// Type returns EntityEmail.
func (d *EmailDetector) Type() EntityType {
	return EntityEmail
}

// ContextWords returns keywords that provide contextual evidence for email addresses.
func (d *EmailDetector) ContextWords() []string {
	return []string{"email", "e-mail", "contato", "correio", "mensagem", "enviar para"}
}

// Detect scans text for email addresses.
func (d *EmailDetector) Detect(text string) []Match {
	indices := emailRegex.FindAllStringIndex(text, -1)
	if len(indices) == 0 {
		return nil
	}

	matches := make([]Match, 0, len(indices))
	for _, idx := range indices {
		val := text[idx[0]:idx[1]]
		ctxText := ExtractSurroundingContext(text, idx[0], idx[1], 5)
		ctxFound := FindContextKeywords(ctxText, d.ContextWords())
		score := 0.90
		if len(ctxFound) > 0 {
			score = 0.95
		}
		matches = append(matches, Match{
			Type:            EntityEmail,
			Value:           val,
			Start:           idx[0],
			End:             idx[1],
			Score:           score,
			ContextFound:    ctxFound,
			ValidatorPassed: true,
		})
	}
	return matches
}
