package detector

import (
	"regexp"
	"strings"
	"unicode"
)

// cepRegex matches Brazilian postal codes (CEP: 12345-678 or 12345678).
var cepRegex = regexp.MustCompile(`\b\d{5}-?\d{3}\b`)

// CEPDetector detects Brazilian postal codes (CEP).
type CEPDetector struct{}

// NewCEPDetector creates a new instance of CEPDetector.
func NewCEPDetector() *CEPDetector {
	return &CEPDetector{}
}

// Type returns EntityCEP.
func (d *CEPDetector) Type() EntityType {
	return EntityCEP
}

// ContextWords returns keywords that provide contextual evidence for postal codes (CEP).
func (d *CEPDetector) ContextWords() []string {
	return []string{"cep", "endereco", "bairro", "cidade", "logradouro", "codigo postal", "entrega", "rua", "av", "avenida"}
}

// Detect scans text for valid CEP patterns.
func (d *CEPDetector) Detect(text string) []Match {
	indices := cepRegex.FindAllStringIndex(text, -1)
	if len(indices) == 0 {
		return nil
	}

	matches := make([]Match, 0, len(indices))
	for _, idx := range indices {
		candidate := text[idx[0]:idx[1]]
		if ValidateCEP(candidate) {
			ctxText := ExtractSurroundingContext(text, idx[0], idx[1], 5)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())

			// Formatted CEP (e.g. 12345-678) has higher innate structure than raw 8 digits
			score := 0.50
			if strings.Contains(candidate, "-") {
				score = 0.85
			}
			if len(ctxFound) > 0 {
				if strings.Contains(candidate, "-") {
					score = 0.95
				} else {
					score = 0.90
				}
			}

			matches = append(matches, Match{
				Type:            EntityCEP,
				Value:           candidate,
				Start:           idx[0],
				End:             idx[1],
				Score:           score,
				ContextFound:    ctxFound,
				ValidatorPassed: true,
			})
		}
	}
	return matches
}

// ValidateCEP validates a CEP number.
func ValidateCEP(raw string) bool {
	var sb strings.Builder
	for _, r := range raw {
		if unicode.IsDigit(r) {
			sb.WriteRune(r)
		}
	}
	digits := sb.String()
	if len(digits) != 8 {
		return false
	}

	// Reject all zeros
	if digits == "00000000" {
		return false
	}

	return true
}
