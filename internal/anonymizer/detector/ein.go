package detector

import (
	"regexp"
	"strconv"
	"unicode"
)

// einFormattedRegex matches standard formatted US Employer Identification Numbers (XX-XXXXXXX).
var einFormattedRegex = regexp.MustCompile(`\b\d{2}-\d{7}\b`)

// einRawRegex matches unformatted 9-digit numbers that could be an EIN when accompanied by EIN context.
var einRawRegex = regexp.MustCompile(`\b\d{9}\b`)

// validEINPrefixes contains official 2-digit campus prefixes assigned by the IRS.
var validEINPrefixes = map[int]bool{
	1: true, 2: true, 3: true, 4: true, 5: true, 6: true,
	10: true, 11: true, 12: true, 13: true, 14: true, 15: true, 16: true,
	20: true, 21: true, 22: true, 23: true, 24: true, 25: true, 26: true, 27: true,
	30: true, 31: true, 32: true, 33: true, 34: true, 35: true, 36: true, 37: true, 38: true, 39: true,
	40: true, 41: true, 42: true, 43: true, 44: true, 45: true, 46: true, 47: true, 48: true,
	50: true, 51: true, 52: true, 53: true, 54: true, 55: true, 56: true, 57: true, 58: true, 59: true,
	60: true, 61: true, 62: true, 63: true, 64: true, 65: true, 66: true, 67: true, 68: true,
	71: true, 72: true, 73: true, 74: true, 75: true, 76: true, 77: true,
	80: true, 81: true, 82: true, 83: true, 84: true, 85: true, 86: true, 87: true, 88: true,
	90: true, 91: true, 92: true, 93: true, 94: true, 95: true,
	98: true, 99: true,
}

// EINDetector detects US Employer Identification Numbers (EIN / FEIN).
type EINDetector struct{}

// NewEINDetector creates a new EINDetector.
func NewEINDetector() *EINDetector {
	return &EINDetector{}
}

// Type returns EntityEIN.
func (d *EINDetector) Type() EntityType {
	return EntityEIN
}

// ContextWords returns keywords that provide contextual evidence for US EIN.
func (d *EINDetector) ContextWords() []string {
	return []string{
		"ein", "fein", "employer identification", "employer id", "federal tax", "tax id", "irs", "company tax",
	}
}

// Detect scans input text for US EIN numbers.
func (d *EINDetector) Detect(text string) []Match {
	if text == "" {
		return nil
	}

	var matches []Match
	claimed := make(map[int]bool)

	// 1. Scan formatted EINs (XX-XXXXXXX)
	formattedIndices := einFormattedRegex.FindAllStringIndex(text, -1)
	for _, idx := range formattedIndices {
		start, end := idx[0], idx[1]
		candidate := text[start:end]
		if ValidateEIN(candidate) {
			ctxText := ExtractSurroundingContext(text, start, end, 5)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())

			score := 0.95
			if len(ctxFound) > 0 {
				score = 1.0
			}

			matches = append(matches, Match{
				Type:            EntityEIN,
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

	// 2. Scan unformatted 9-digit numbers ONLY when explicit EIN context is found
	rawIndices := einRawRegex.FindAllStringIndex(text, -1)
	for _, idx := range rawIndices {
		start, end := idx[0], idx[1]
		if claimed[start] || claimed[end-1] {
			continue
		}

		candidate := text[start:end]
		if ValidateEIN(candidate) {
			ctxText := ExtractSurroundingContext(text, start, end, 6)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())

			// Strictly require context for unformatted EIN
			if len(ctxFound) == 0 {
				continue
			}

			matches = append(matches, Match{
				Type:            EntityEIN,
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

// ValidateEIN verifies if candidate string is a structurally valid IRS Employer Identification Number.
func ValidateEIN(raw string) bool {
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

	// First two digits represent the IRS campus prefix
	prefix, err := strconv.Atoi(string(digits[:2]))
	if err != nil || !validEINPrefixes[prefix] {
		return false
	}

	// Last 7 digits cannot be all zeros
	allZeros := true
	for i := 2; i < 9; i++ {
		if digits[i] != '0' {
			allZeros = false
			break
		}
	}
	if allZeros {
		return false
	}

	return true
}
