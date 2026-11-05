package detector

import (
	"regexp"
	"strconv"
	"unicode"
)

// dniRegex matches Spanish DNI candidates (8 digits followed by a letter, optional dots and hyphen).
var dniRegex = regexp.MustCompile(`(?i)\b\d{1,2}(?:\.?\d{3}){2}[- ]?[A-Z]\b`)

// nieRegex matches Spanish NIE candidates (starts with X, Y, or Z, followed by 7 digits and a letter).
var nieRegex = regexp.MustCompile(`(?i)\b[XYZ][- ]?\d{7}[- ]?[A-Z]\b`)

// dniControlChars is the official Modulo 23 control character table for Spanish DNI and NIE.
const dniControlChars = "TRWAGMYFPDXBNJZSQVHLCKE"

// DNIDetector detects Spanish DNI and NIE identifiers using official Modulo 23 verification.
type DNIDetector struct{}

// NewDNIDetector creates a new DNIDetector.
func NewDNIDetector() *DNIDetector {
	return &DNIDetector{}
}

// Type returns EntityDNI.
func (d *DNIDetector) Type() EntityType {
	return EntityDNI
}

// ContextWords returns keywords that provide contextual evidence for Spanish DNI / NIE.
func (d *DNIDetector) ContextWords() []string {
	return []string{
		"dni", "nie", "nif", "documento de identidad", "identidad", "españa", "spain", "residencia", "extranjero",
	}
}

// Detect scans input text for valid Spanish DNI and NIE numbers.
func (d *DNIDetector) Detect(text string) []Match {
	if text == "" {
		return nil
	}

	var matches []Match
	claimed := make(map[int]bool)

	// 1. Scan for NIE candidates
	nieIndices := nieRegex.FindAllStringIndex(text, -1)
	for _, idx := range nieIndices {
		start, end := idx[0], idx[1]
		candidate := text[start:end]
		if ValidateNIE(candidate) {
			ctxText := ExtractSurroundingContext(text, start, end, 5)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())

			score := 0.98
			if len(ctxFound) > 0 {
				score = 1.0
			}

			matches = append(matches, Match{
				Type:            EntityDNI,
				SubType:         "NIE",
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

	// 2. Scan for DNI candidates
	dniIndices := dniRegex.FindAllStringIndex(text, -1)
	for _, idx := range dniIndices {
		start, end := idx[0], idx[1]
		if claimed[start] || claimed[end-1] {
			continue
		}

		candidate := text[start:end]
		if ValidateDNI(candidate) {
			ctxText := ExtractSurroundingContext(text, start, end, 5)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())

			score := 0.98
			if len(ctxFound) > 0 {
				score = 1.0
			}

			matches = append(matches, Match{
				Type:            EntityDNI,
				SubType:         "DNI",
				Value:           candidate,
				Start:           start,
				End:             end,
				Score:           score,
				ContextFound:    ctxFound,
				ValidatorPassed: true,
			})
		}
	}

	return matches
}

// ValidateDNI checks if candidate is a valid 8-digit Spanish DNI matching Modulo 23.
func ValidateDNI(raw string) bool {
	var digits []rune
	var letter rune

	for _, r := range raw {
		if unicode.IsDigit(r) {
			digits = append(digits, r)
		} else if unicode.IsLetter(r) {
			letter = unicode.ToUpper(r)
		}
	}

	if len(digits) != 8 || letter == 0 {
		return false
	}

	num, err := strconv.Atoi(string(digits))
	if err != nil {
		return false
	}

	expectedLetter := rune(dniControlChars[num%23])
	return letter == expectedLetter
}

// ValidateNIE checks if candidate is a valid Spanish NIE matching Modulo 23.
func ValidateNIE(raw string) bool {
	var clean []rune
	for _, r := range raw {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			clean = append(clean, unicode.ToUpper(r))
		}
	}

	if len(clean) != 9 {
		return false
	}

	first := clean[0]
	last := clean[8]

	var prefixDigit string
	switch first {
	case 'X':
		prefixDigit = "0"
	case 'Y':
		prefixDigit = "1"
	case 'Z':
		prefixDigit = "2"
	default:
		return false
	}

	numStr := prefixDigit + string(clean[1:8])
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return false
	}

	expectedLetter := rune(dniControlChars[num%23])
	return last == expectedLetter
}
