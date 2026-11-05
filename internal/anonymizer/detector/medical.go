package detector

import (
	"regexp"
	"strings"
	"unicode"
)

// npiRegex matches 10-digit National Provider Identifier candidates.
var npiRegex = regexp.MustCompile(`\b[12]\d{9}\b`)

// deaRegex matches Drug Enforcement Administration registration number format (2 letters + 7 digits).
var deaRegex = regexp.MustCompile(`(?i)\b[A-Z]{2}\d{7}\b`)

// mrnPrefixRegex matches medical record numbers preceded by explicit clinical identifiers.
var mrnPrefixRegex = regexp.MustCompile(`(?i)\b(?:mrn|medical record(?: number)?|patient id|patient #|prontu[aá]rio(?: do paciente)?|prontuario n[oº]?|record #)\s*[:#\-]?\s*([A-Z0-9\-]{5,14})\b`)

// MedicalDetector detects HIPAA-regulated medical identifiers: NPI, DEA, and Medical Record Numbers (MRN).
type MedicalDetector struct{}

// NewMedicalDetector creates a new MedicalDetector.
func NewMedicalDetector() *MedicalDetector {
	return &MedicalDetector{}
}

// Type returns EntityMedicalID.
func (d *MedicalDetector) Type() EntityType {
	return EntityMedicalID
}

// ContextWords returns healthcare and HIPAA context keywords.
func (d *MedicalDetector) ContextWords() []string {
	return []string{
		"npi", "national provider identifier", "dea", "dea number", "prescriber", "physician",
		"doctor", "mrn", "medical record", "patient", "hipaa", "prontuario", "prontuário",
		"hospital", "clinic", "prescription", "receita medica",
	}
}

// Detect scans input text for HIPAA healthcare identifiers.
func (d *MedicalDetector) Detect(text string) []Match {
	if text == "" {
		return nil
	}

	var matches []Match
	claimed := make(map[int]bool)

	// 1. Scan for MRN (Medical Record Number) with preceding context marker
	mrnMatches := mrnPrefixRegex.FindAllStringSubmatchIndex(text, -1)
	for _, idx := range mrnMatches {
		if len(idx) < 4 {
			continue
		}
		valStart, valEnd := idx[2], idx[3]
		candidate := text[valStart:valEnd]

		// Ensure candidate has at least some digits or letters and is not just punctuation
		if len(strings.TrimSpace(candidate)) >= 5 {
			ctxText := ExtractSurroundingContext(text, valStart, valEnd, 6)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())

			matches = append(matches, Match{
				Type:            EntityMedicalID,
				SubType:         "MRN",
				Value:           candidate,
				Start:           valStart,
				End:             valEnd,
				Score:           0.95,
				ContextFound:    ctxFound,
				ValidatorPassed: true,
			})
			for k := valStart; k < valEnd; k++ {
				claimed[k] = true
			}
		}
	}

	// 2. Scan for DEA numbers (2 letters + 7 digits)
	deaIndices := deaRegex.FindAllStringIndex(text, -1)
	for _, idx := range deaIndices {
		start, end := idx[0], idx[1]
		if claimed[start] || claimed[end-1] {
			continue
		}

		candidate := text[start:end]
		if ValidateDEA(candidate) {
			ctxText := ExtractSurroundingContext(text, start, end, 5)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())

			score := 0.95
			if len(ctxFound) > 0 {
				score = 1.0
			}

			matches = append(matches, Match{
				Type:            EntityMedicalID,
				SubType:         "DEA",
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

	// 3. Scan for NPI numbers (10 digits starting with 1 or 2, with 80840 Luhn verification)
	npiIndices := npiRegex.FindAllStringIndex(text, -1)
	for _, idx := range npiIndices {
		start, end := idx[0], idx[1]
		if claimed[start] || claimed[end-1] {
			continue
		}

		candidate := text[start:end]
		if ValidateNPI(candidate) {
			ctxText := ExtractSurroundingContext(text, start, end, 6)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())

			// Strictly require healthcare context for 10-digit NPIs to prevent false positives with generic numbers
			if len(ctxFound) == 0 {
				continue
			}

			matches = append(matches, Match{
				Type:            EntityMedicalID,
				SubType:         "NPI",
				Value:           candidate,
				Start:           start,
				End:             end,
				Score:           0.95,
				ContextFound:    ctxFound,
				ValidatorPassed: true,
			})
		}
	}

	return matches
}

// ValidateNPI checks whether candidate is a valid 10-digit NPI using the ISO 80840 Luhn algorithm.
func ValidateNPI(candidate string) bool {
	if len(candidate) != 10 {
		return false
	}
	if candidate[0] != '1' && candidate[0] != '2' {
		return false
	}

	// Full 15-digit number prefixed with 80840
	full := "80840" + candidate
	return luhnCheck(full)
}

// ValidateDEA verifies candidate string using the official DEA checksum.
// DEA formula: (digit1 + digit3 + digit5) + 2*(digit2 + digit4 + digit6); check digit is last digit of sum.
func ValidateDEA(candidate string) bool {
	upper := strings.ToUpper(strings.TrimSpace(candidate))
	if len(upper) != 9 {
		return false
	}

	// First two must be letters
	if !unicode.IsLetter(rune(upper[0])) || !unicode.IsLetter(rune(upper[1])) {
		return false
	}

	// Next 7 must be digits
	var digits [7]int
	for i := 0; i < 7; i++ {
		r := rune(upper[2+i])
		if !unicode.IsDigit(r) {
			return false
		}
		digits[i] = int(r - '0')
	}

	sum1 := digits[0] + digits[2] + digits[4]
	sum2 := (digits[1] + digits[3] + digits[5]) * 2
	total := sum1 + sum2
	checkDigit := total % 10

	return checkDigit == digits[6]
}

// luhnCheck performs a standard Luhn mod 10 check on a numeric string.
func luhnCheck(s string) bool {
	sum := 0
	alternate := false

	for i := len(s) - 1; i >= 0; i-- {
		n := int(s[i] - '0')
		if n < 0 || n > 9 {
			return false
		}

		if alternate {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}

		sum += n
		alternate = !alternate
	}

	return sum%10 == 0
}

