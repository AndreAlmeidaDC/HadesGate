package detector

import (
	"regexp"
	"strings"
	"unicode"
)

// cpfRegex matches formatted (123.456.789-00) or unformatted (12345678900) CPF patterns.
var cpfRegex = regexp.MustCompile(`\b\d{3}\.?\d{3}\.?\d{3}-?\d{2}\b`)

var cpfCompetingCNHKeywords = []string{
	"cnh", "habilitacao", "habilitação", "carteira de motorista", "renach", "detran", "condutor",
}

// CPFDetector detects valid Brazilian CPFs using official MOD 11 verification.
type CPFDetector struct{}

// NewCPFDetector creates a new instance of CPFDetector.
func NewCPFDetector() *CPFDetector {
	return &CPFDetector{}
}

// Type returns the EntityType for CPF.
func (d *CPFDetector) Type() EntityType {
	return EntityCPF
}

// ContextWords returns keywords that provide contextual evidence for CPFs.
func (d *CPFDetector) ContextWords() []string {
	return []string{"cpf", "cadastro de pessoas fisicas", "cadastro de pessoas físicas", "titular", "documento", "receita federal"}
}

// Detect scans the input text for valid CPF numbers.
func (d *CPFDetector) Detect(text string) []Match {
	indices := cpfRegex.FindAllStringIndex(text, -1)
	if len(indices) == 0 {
		return nil
	}

	matches := make([]Match, 0, len(indices))
	for _, idx := range indices {
		candidate := text[idx[0]:idx[1]]
		if ValidateCPF(candidate) {
			ctxText := ExtractSurroundingContext(text, idx[0], idx[1], 5)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())

			isFormatted := strings.Contains(candidate, ".") || strings.Contains(candidate, "-")
			score := 0.96
			if isFormatted || len(ctxFound) > 0 {
				score = 1.0
			} else if len(FindContextKeywords(ctxText, cpfCompetingCNHKeywords)) > 0 {
				// Unformatted 11-digit number in an explicit CNH context should yield to CNHDetector
				score = 0.85
			}

			matches = append(matches, Match{
				Type:            EntityCPF,
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

// ValidateCPF verifies if a given string contains a valid Brazilian CPF with correct check digits.
func ValidateCPF(raw string) bool {
	// Extract digits only
	digits := make([]int, 0, 11)
	for _, r := range raw {
		if unicode.IsDigit(r) {
			digits = append(digits, int(r-'0'))
		}
	}

	if len(digits) != 11 {
		return false
	}

	// Reject all-equal numbers (e.g. 111.111.111-11, 000.000.000-00)
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

	// Compute first check digit
	sum1 := 0
	for i := 0; i < 9; i++ {
		sum1 += digits[i] * (10 - i)
	}
	d1 := (sum1 * 10) % 11
	if d1 == 10 {
		d1 = 0
	}
	if d1 != digits[9] {
		return false
	}

	// Compute second check digit
	sum2 := 0
	for i := 0; i < 10; i++ {
		sum2 += digits[i] * (11 - i)
	}
	d2 := (sum2 * 10) % 11
	if d2 == 10 {
		d2 = 0
	}
	return d2 == digits[10]
}
