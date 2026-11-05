package detector

import (
	"regexp"
)

// passportRegex matches Brazilian Federal Police passport numbers (2 letters + 6 digits).
var passportRegex = regexp.MustCompile(`\b[A-Z]{2}[0-9]{6}\b`)

// PassportDetector identifies Brazilian passport numbers.
type PassportDetector struct{}

// NewPassportDetector creates a new PassportDetector.
func NewPassportDetector() *PassportDetector {
	return &PassportDetector{}
}

// Type returns EntityPassport.
func (d *PassportDetector) Type() EntityType {
	return EntityPassport
}

// ContextWords returns keywords providing contextual evidence for passports.
func (d *PassportDetector) ContextWords() []string {
	return []string{
		"passaporte", "passport", "policia federal", "dpf",
		"viagem", "imigracao", "imigração", "embarque", "internacional",
	}
}

// Detect scans text for passport numbers.
func (d *PassportDetector) Detect(text string) []Match {
	indices := passportRegex.FindAllStringIndex(text, -1)
	if len(indices) == 0 {
		return nil
	}

	matches := make([]Match, 0, len(indices))
	for _, idx := range indices {
		candidate := text[idx[0]:idx[1]]
		ctxText := ExtractSurroundingContext(text, idx[0], idx[1], 5)
		ctxFound := FindContextKeywords(ctxText, d.ContextWords())

		score := 0.60
		if len(ctxFound) > 0 {
			score = 0.95
		}

		matches = append(matches, Match{
			Type:            EntityPassport,
			Value:           candidate,
			Start:           idx[0],
			End:             idx[1],
			Score:           score,
			ContextFound:    ctxFound,
			ValidatorPassed: len(ctxFound) > 0,
		})
	}

	return matches
}
