package detector

import (
	"regexp"
	"unicode"
)

// nhsFormattedRegex matches formatted 10-digit NHS numbers (e.g. 123 456 7890, 123-456-7890, or 123 456 789 0).
var nhsFormattedRegex = regexp.MustCompile(`\b\d{3}[ -]\d{3}[ -]\d{4}\b`)

// nhsRawRegex matches 10 consecutive digits that could represent an unformatted NHS number.
var nhsRawRegex = regexp.MustCompile(`\b\d{10}\b`)

// NHSDetector detects UK National Health Service (NHS) numbers using official Modulo 11 verification.
type NHSDetector struct{}

// NewNHSDetector creates a new NHSDetector.
func NewNHSDetector() *NHSDetector {
	return &NHSDetector{}
}

// Type returns EntityNHS.
func (d *NHSDetector) Type() EntityType {
	return EntityNHS
}

// ContextWords returns keywords that provide contextual evidence for NHS numbers.
func (d *NHSDetector) ContextWords() []string {
	return []string{
		"nhs", "nhs number", "national health service", "patient", "gp surgery", "hospital", "clinic",
	}
}

// Detect scans input text for valid NHS numbers.
func (d *NHSDetector) Detect(text string) []Match {
	if text == "" {
		return nil
	}

	var matches []Match
	claimed := make(map[int]bool)

	// 1. Scan formatted NHS numbers (XXX XXX XXXX)
	formattedIndices := nhsFormattedRegex.FindAllStringIndex(text, -1)
	for _, idx := range formattedIndices {
		start, end := idx[0], idx[1]
		candidate := text[start:end]
		if ValidateNHS(candidate) {
			ctxText := ExtractSurroundingContext(text, start, end, 5)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())

			score := 0.96
			if len(ctxFound) > 0 {
				score = 1.0
			}

			matches = append(matches, Match{
				Type:            EntityNHS,
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

	// 2. Scan unformatted 10-digit NHS numbers (only with context to prevent collisions with 10-digit numbers/phones)
	rawIndices := nhsRawRegex.FindAllStringIndex(text, -1)
	for _, idx := range rawIndices {
		start, end := idx[0], idx[1]
		if claimed[start] || claimed[end-1] {
			continue
		}

		candidate := text[start:end]
		if ValidateNHS(candidate) {
			ctxText := ExtractSurroundingContext(text, start, end, 6)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())

			if len(ctxFound) == 0 {
				continue
			}

			matches = append(matches, Match{
				Type:            EntityNHS,
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

// ValidateNHS verifies a 10-digit NHS number using the official Modulo 11 checksum algorithm.
func ValidateNHS(raw string) bool {
	digits := make([]int, 0, 10)
	for _, r := range raw {
		if unicode.IsDigit(r) {
			digits = append(digits, int(r-'0'))
		}
	}

	if len(digits) != 10 {
		return false
	}

	// Calculate weighted sum for the first 9 digits
	// Weights: 10, 9, 8, 7, 6, 5, 4, 3, 2
	sum := 0
	for i := 0; i < 9; i++ {
		sum += digits[i] * (10 - i)
	}

	remainder := sum % 11
	checkDigit := 11 - remainder
	if checkDigit == 11 {
		checkDigit = 0
	} else if checkDigit == 10 {
		// A check digit of 10 is invalid per NHS standard
		return false
	}

	return checkDigit == digits[9]
}
