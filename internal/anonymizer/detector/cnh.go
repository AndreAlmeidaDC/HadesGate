package detector

import (
	"regexp"
	"unicode"
)

// cnhRegex matches 11-digit numbers formatted or unformatted.
var cnhRegex = regexp.MustCompile(`\b\d{11}\b`)

// CNHDetector identifies Brazilian Driver's License (CNH) numbers via official check-digit calculation.
type CNHDetector struct{}

// NewCNHDetector creates a new CNHDetector.
func NewCNHDetector() *CNHDetector {
	return &CNHDetector{}
}

// Type returns EntityCNH.
func (d *CNHDetector) Type() EntityType {
	return EntityCNH
}

// ContextWords returns keywords providing contextual evidence for CNH documents.
func (d *CNHDetector) ContextWords() []string {
	return []string{
		"cnh", "habilitacao", "habilitação", "carteira de motorista",
		"carteira nacional de habilitacao", "carteira nacional de habilitação", "renach", "detran", "condutor",
	}
}

// Detect scans text for valid Brazilian CNH numbers.
func (d *CNHDetector) Detect(text string) []Match {
	indices := cnhRegex.FindAllStringIndex(text, -1)
	if len(indices) == 0 {
		return nil
	}

	matches := make([]Match, 0, len(indices))
	for _, idx := range indices {
		candidate := text[idx[0]:idx[1]]
		if ValidateCNH(candidate) {
			ctxText := ExtractSurroundingContext(text, idx[0], idx[1], 5)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())

			score := 0.94
			if len(ctxFound) > 0 {
				score = 1.0
			}

			matches = append(matches, Match{
				Type:            EntityCNH,
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

// ValidateCNH verifies an 11-digit Brazilian Driver's License using the official CONTRAN Mod 11 algorithm.
func ValidateCNH(raw string) bool {
	digits := make([]int, 0, 11)
	for _, r := range raw {
		if unicode.IsDigit(r) {
			digits = append(digits, int(r-'0'))
		}
	}

	if len(digits) != 11 {
		return false
	}

	// Reject all identical digits
	allEqual := true
	for i := 1; i < 11; i++ {
		if digits[i] != digits[0] {
			allEqual = false
			break
		}
	}
	if allEqual {
		return false
	}

	// First verification digit
	sum1 := 0
	for i := 0; i < 9; i++ {
		sum1 += digits[i] * (9 - i)
	}
	v1 := sum1 % 11
	dsc := 0
	if v1 >= 10 {
		v1 = 0
		dsc = 2
	}
	if v1 != digits[9] {
		return false
	}

	// Second verification digit
	sum2 := 0
	for i := 0; i < 9; i++ {
		sum2 += digits[i] * (i + 1)
	}
	v2 := sum2 % 11
	if v2 >= 10 {
		v2 = 0
	} else {
		v2 -= dsc
		if v2 < 0 {
			v2 += 11
		}
	}

	return v2 == digits[10]
}
