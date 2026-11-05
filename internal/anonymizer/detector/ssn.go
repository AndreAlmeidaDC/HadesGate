package detector

import (
	"regexp"
	"unicode"
)

// ssnFormattedRegex matches standard formatted US Social Security Numbers (XXX-XX-XXXX or XXX XX XXXX).
var ssnFormattedRegex = regexp.MustCompile(`\b\d{3}[- ]\d{2}[- ]\d{4}\b`)

// ssnRawRegex matches continuous 9-digit numbers that could be an unformatted SSN.
var ssnRawRegex = regexp.MustCompile(`\b\d{9}\b`)

// SSNDetector detects US Social Security Numbers per Social Security Administration (SSA) rules.
type SSNDetector struct{}

// NewSSNDetector creates a new SSNDetector.
func NewSSNDetector() *SSNDetector {
	return &SSNDetector{}
}

// Type returns EntitySSN.
func (d *SSNDetector) Type() EntityType {
	return EntitySSN
}

// ContextWords returns keywords that provide contextual evidence for US SSN.
func (d *SSNDetector) ContextWords() []string {
	return []string{
		"ssn", "social security", "social-security", "social sec", "soc sec", "taxpayer id", "tin",
	}
}

// Detect scans input text for US Social Security Numbers.
func (d *SSNDetector) Detect(text string) []Match {
	if text == "" {
		return nil
	}

	var matches []Match
	claimed := make(map[int]bool)

	// 1. First scan for formatted SSNs (XXX-XX-XXXX or XXX XX XXXX)
	formattedIndices := ssnFormattedRegex.FindAllStringIndex(text, -1)
	for _, idx := range formattedIndices {
		start, end := idx[0], idx[1]
		candidate := text[start:end]
		if ValidateSSN(candidate) {
			ctxText := ExtractSurroundingContext(text, start, end, 5)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())

			score := 0.95
			if len(ctxFound) > 0 {
				score = 1.0
			}

			matches = append(matches, Match{
				Type:            EntitySSN,
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

	// 2. Scan for unformatted 9-digit numbers ONLY when accompanied by explicit SSN context keywords
	rawIndices := ssnRawRegex.FindAllStringIndex(text, -1)
	for _, idx := range rawIndices {
		start, end := idx[0], idx[1]
		if claimed[start] || claimed[end-1] {
			continue
		}

		candidate := text[start:end]
		if ValidateSSN(candidate) {
			ctxText := ExtractSurroundingContext(text, start, end, 6)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())

			// Strictly require context for unformatted 9-digit numbers to avoid false positives with order IDs or timestamps
			if len(ctxFound) == 0 {
				continue
			}

			matches = append(matches, Match{
				Type:            EntitySSN,
				Value:           candidate,
				Start:           start,
				End:             end,
				Score:           0.90,
				ContextFound:    ctxFound,
				ValidatorPassed: true,
			})
		}
	}

	return matches
}

// ValidateSSN verifies whether candidate conforms to official SSA structural and range rules.
func ValidateSSN(raw string) bool {
	// Extract digits only
	digits := make([]byte, 0, 9)
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if unicode.IsDigit(rune(c)) {
			digits = append(digits, c)
		}
	}

	if len(digits) != 9 {
		return false
	}

	// Reject all-equal numbers (e.g. 000-00-0000, 111-11-1111)
	allEqual := true
	for i := 1; i < 9; i++ {
		if digits[i] != digits[0] {
			allEqual = false
			break
		}
	}
	if allEqual {
		return false
	}

	// Parse area (first 3), group (middle 2), serial (last 4)
	area := int(digits[0]-'0')*100 + int(digits[1]-'0')*10 + int(digits[2]-'0')
	group := int(digits[3]-'0')*10 + int(digits[4]-'0')
	serial := int(digits[5]-'0')*1000 + int(digits[6]-'0')*100 + int(digits[7]-'0')*10 + int(digits[8]-'0')

	// Rule 1: Area cannot be 000, 666, or 900-999
	if area == 0 || area == 666 || area >= 900 {
		return false
	}

	// Rule 2: Group cannot be 00
	if group == 0 {
		return false
	}

	// Rule 3: Serial cannot be 0000
	if serial == 0 {
		return false
	}

	// Rule 4: Known dummy / test numbers (e.g. 078-05-1120 used in advertising)
	s := string(digits)
	if s == "078051120" || s == "219099999" || s == "123456789" || s == "987654321" {
		return false
	}

	return true
}
