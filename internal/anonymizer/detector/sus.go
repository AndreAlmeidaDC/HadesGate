package detector

import (
	"fmt"
	"regexp"
	"unicode"
)

// susFormattedRegex matches formatted 15-digit CNS numbers (e.g. 123 4567 8901 2345 or 123-4567-8901-2345).
var susFormattedRegex = regexp.MustCompile(`\b[12789]\d{2}[ -]?\d{4}[ -]?\d{4}[ -]?\d{4}\b`)

// susRawRegex matches continuous 15-digit CNS numbers.
var susRawRegex = regexp.MustCompile(`\b[12789]\d{14}\b`)

// SUSDetector detects Brazilian Cartão Nacional de Saúde (CNS / SUS) numbers.
type SUSDetector struct{}

// NewSUSDetector creates a new SUSDetector.
func NewSUSDetector() *SUSDetector {
	return &SUSDetector{}
}

// Type returns EntitySUS.
func (d *SUSDetector) Type() EntityType {
	return EntitySUS
}

// ContextWords returns healthcare context keywords for Brazilian SUS.
func (d *SUSDetector) ContextWords() []string {
	return []string{
		"sus", "cartao sus", "cartão sus", "cns", "cartao nacional de saude", "cartão nacional de saúde",
		"saude", "saúde", "ubs", "postinho", "ministerio da saude", "ministério da saúde", "paciente",
		"prontuario", "prontuário", "atendimento sus",
	}
}

// Detect scans input text for Brazilian SUS numbers.
func (d *SUSDetector) Detect(text string) []Match {
	if text == "" {
		return nil
	}

	var matches []Match
	claimed := make(map[int]bool)

	// 1. Scan formatted candidates
	formattedIndices := susFormattedRegex.FindAllStringIndex(text, -1)
	for _, idx := range formattedIndices {
		start, end := idx[0], idx[1]
		candidate := text[start:end]
		if ValidateSUS(candidate) {
			ctxText := ExtractSurroundingContext(text, start, end, 5)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())

			score := 0.96
			if len(ctxFound) > 0 {
				score = 1.0
			}

			matches = append(matches, Match{
				Type:            EntitySUS,
				Value:           candidate,
				Start:           start,
				End:             end,
				Score:           score,
				ContextFound:    ctxFound,
				ValidatorPassed: true,
			})
			for k := start; k < end; k++ {
				claimed[k] = true
			}
		}
	}

	// 2. Scan continuous 15-digit candidates (require context to prevent collisions)
	rawIndices := susRawRegex.FindAllStringIndex(text, -1)
	for _, idx := range rawIndices {
		start, end := idx[0], idx[1]
		if claimed[start] || claimed[end-1] {
			continue
		}

		candidate := text[start:end]
		if ValidateSUS(candidate) {
			ctxText := ExtractSurroundingContext(text, start, end, 6)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())

			if len(ctxFound) == 0 {
				continue
			}

			matches = append(matches, Match{
				Type:            EntitySUS,
				Value:           candidate,
				Start:           start,
				End:             end,
				Score:           0.92,
				ContextFound:    ctxFound,
				ValidatorPassed: true,
			})
		}
	}

	return matches
}

// ValidateSUS verifies whether candidate is a valid 15-digit Cartão Nacional de Saúde per DATASUS specification.
func ValidateSUS(raw string) bool {
	digits := make([]int, 0, 15)
	for _, r := range raw {
		if unicode.IsDigit(r) {
			digits = append(digits, int(r-'0'))
		}
	}

	if len(digits) != 15 {
		return false
	}

	first := digits[0]

	// 1. Definitive cards: start with 1 or 2
	if first == 1 || first == 2 {
		sum := 0
		for i := 0; i < 11; i++ {
			sum += digits[i] * (15 - i)
		}

		remainder := sum % 11
		dv := 11 - remainder
		if dv == 11 {
			dv = 0
		}

		var expectedSuffix string
		if dv == 10 {
			sum2 := sum + 2
			remainder2 := sum2 % 11
			dv2 := 11 - remainder2
			expectedSuffix = fmt.Sprintf("001%d", dv2)
		} else {
			expectedSuffix = fmt.Sprintf("000%d", dv)
		}

		actualSuffix := fmt.Sprintf("%d%d%d%d", digits[11], digits[12], digits[13], digits[14])
		return actualSuffix == expectedSuffix
	}

	// 2. Provisional cards: start with 7, 8, or 9
	if first == 7 || first == 8 || first == 9 {
		sum := 0
		for i := 0; i < 15; i++ {
			sum += digits[i] * (15 - i)
		}
		return sum%11 == 0
	}

	return false
}

