package detector

import (
	"regexp"
	"strings"
	"unicode"
)

// codiceFiscaleRegex matches the structural format of Italian Codice Fiscale.
var codiceFiscaleRegex = regexp.MustCompile(`(?i)\b[A-Z]{6}[0-9]{2}[A-EHLMPR-T][0-9]{2}[A-Z][0-9]{3}[A-Z]\b`)

// cfOddTable maps characters at odd positions (1st, 3rd, 5th, etc.) to their numeric weights.
var cfOddTable = map[rune]int{
	'0': 1, '1': 0, '2': 5, '3': 7, '4': 9, '5': 13, '6': 15, '7': 17, '8': 19, '9': 21,
	'A': 1, 'B': 0, 'C': 5, 'D': 7, 'E': 9, 'F': 13, 'G': 15, 'H': 17, 'I': 19, 'J': 21,
	'K': 2, 'L': 4, 'M': 18, 'N': 20, 'O': 11, 'P': 3, 'Q': 6, 'R': 8, 'S': 12, 'T': 14,
	'U': 16, 'V': 10, 'W': 22, 'X': 25, 'Y': 24, 'Z': 23,
}

// CodiceFiscaleDetector detects Italian Codice Fiscale identifiers.
type CodiceFiscaleDetector struct{}

// NewCodiceFiscaleDetector creates a new CodiceFiscaleDetector.
func NewCodiceFiscaleDetector() *CodiceFiscaleDetector {
	return &CodiceFiscaleDetector{}
}

// Type returns EntityCodiceFiscale.
func (d *CodiceFiscaleDetector) Type() EntityType {
	return EntityCodiceFiscale
}

// ContextWords returns keywords that provide contextual evidence for Italian Codice Fiscale.
func (d *CodiceFiscaleDetector) ContextWords() []string {
	return []string{
		"codice fiscale", "cf", "fiscale", "tessera sanitaria", "agenzia delle entrate", "italia", "italy",
	}
}

// Detect scans input text for Italian Codice Fiscale codes.
func (d *CodiceFiscaleDetector) Detect(text string) []Match {
	if text == "" {
		return nil
	}

	indices := codiceFiscaleRegex.FindAllStringIndex(text, -1)
	if len(indices) == 0 {
		return nil
	}

	var matches []Match
	for _, idx := range indices {
		start, end := idx[0], idx[1]
		candidate := text[start:end]

		if ValidateCodiceFiscale(candidate) {
			ctxText := ExtractSurroundingContext(text, start, end, 5)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())

			score := 0.98
			if len(ctxFound) > 0 {
				score = 1.0
			}

			matches = append(matches, Match{
				Type:            EntityCodiceFiscale,
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

// ValidateCodiceFiscale calculates the official Agenzia delle Entrate check character for a 16-character code.
func ValidateCodiceFiscale(raw string) bool {
	rawUpper := strings.ToUpper(strings.TrimSpace(raw))
	if len(rawUpper) != 16 {
		return false
	}

	runes := []rune(rawUpper)
	sum := 0

	for i := 0; i < 15; i++ {
		r := runes[i]
		pos := i + 1 // 1-based index

		if pos%2 == 1 {
			// Odd position (1, 3, 5, ..., 15)
			val, ok := cfOddTable[r]
			if !ok {
				return false
			}
			sum += val
		} else {
			// Even position (2, 4, 6, ..., 14)
			if unicode.IsDigit(r) {
				sum += int(r - '0')
			} else if unicode.IsLetter(r) {
				sum += int(r - 'A')
			} else {
				return false
			}
		}
	}

	expectedChar := rune('A' + (sum % 26))
	return runes[15] == expectedChar
}

