package detector

import (
	"regexp"
	"unicode"
)

// cnpjRegex matches formatted (12.345.678/0001-90) or unformatted (12345678000190) CNPJ patterns.
var cnpjRegex = regexp.MustCompile(`\b\d{2}\.?\d{3}\.?\d{3}/?\d{4}-?\d{2}\b`)

// CNPJDetector detects valid Brazilian CNPJs using official MOD 11 verification.
type CNPJDetector struct{}

// NewCNPJDetector creates a new instance of CNPJDetector.
func NewCNPJDetector() *CNPJDetector {
	return &CNPJDetector{}
}

// Type returns the EntityType for CNPJ.
func (d *CNPJDetector) Type() EntityType {
	return EntityCNPJ
}

// ContextWords returns keywords that provide contextual evidence for CNPJs.
func (d *CNPJDetector) ContextWords() []string {
	return []string{"cnpj", "cadastro nacional", "empresa", "razao social", "matriz", "filial", "receita federal", "sociedade"}
}

// Detect scans the input text for valid CNPJ numbers.
func (d *CNPJDetector) Detect(text string) []Match {
	indices := cnpjRegex.FindAllStringIndex(text, -1)
	if len(indices) == 0 {
		return nil
	}

	matches := make([]Match, 0, len(indices))
	for _, idx := range indices {
		candidate := text[idx[0]:idx[1]]
		if ValidateCNPJ(candidate) {
			ctxText := ExtractSurroundingContext(text, idx[0], idx[1], 5)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())
			matches = append(matches, Match{
				Type:            EntityCNPJ,
				Value:           candidate,
				Start:           idx[0],
				End:             idx[1],
				Score:           1.0,
				ContextFound:    ctxFound,
				ValidatorPassed: true,
			})
		}
	}
	return matches
}

var (
	cnpjWeights1 = []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	cnpjWeights2 = []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
)

// ValidateCNPJ verifies if a given string contains a valid Brazilian CNPJ.
func ValidateCNPJ(raw string) bool {
	digits := make([]int, 0, 14)
	for _, r := range raw {
		if unicode.IsDigit(r) {
			digits = append(digits, int(r-'0'))
		}
	}

	if len(digits) != 14 {
		return false
	}

	// Reject all identical digits
	allEqual := true
	for i := 1; i < 14; i++ {
		if digits[i] != digits[0] {
			allEqual = false
			break
		}
	}
	if allEqual {
		return false
	}

	// First digit check
	sum1 := 0
	for i := 0; i < 12; i++ {
		sum1 += digits[i] * cnpjWeights1[i]
	}
	rem1 := sum1 % 11
	d1 := 0
	if rem1 >= 2 {
		d1 = 11 - rem1
	}
	if d1 != digits[12] {
		return false
	}

	// Second digit check
	sum2 := 0
	for i := 0; i < 13; i++ {
		sum2 += digits[i] * cnpjWeights2[i]
	}
	rem2 := sum2 % 11
	d2 := 0
	if rem2 >= 2 {
		d2 = 11 - rem2
	}
	return d2 == digits[13]
}
