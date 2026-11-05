package detector

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// nirFormattedRegex matches formatted French NIR numbers (e.g. 1 85 12 75 108 042 36 or 1-85-12-75-108-042-36).
var nirFormattedRegex = regexp.MustCompile(`\b[1278]\s?\d{2}\s?\d{2}\s?(?:\d{2}|2[AB])\s?\d{3}\s?\d{3}\s?\d{2}\b`)

// nirRawRegex matches continuous 15-character NIR candidates.
var nirRawRegex = regexp.MustCompile(`\b[1278]\d{12}(?:\d{2}|2[AB]\d{10})\b`)

// NIRDetector detects French Numéro d'Inscription au Répertoire / Sécurité Sociale (NIR) using Modulo 97 verification.
type NIRDetector struct{}

// NewNIRDetector creates a new NIRDetector.
func NewNIRDetector() *NIRDetector {
	return &NIRDetector{}
}

// Type returns EntityNIR.
func (d *NIRDetector) Type() EntityType {
	return EntityNIR
}

// ContextWords returns keywords associated with French social security numbers.
func (d *NIRDetector) ContextWords() []string {
	return []string{
		"nir", "sécurité sociale", "securite sociale", "numéro de sécurité sociale", "insee", "carte vitale", "ameli", "secu",
	}
}

// Detect scans input text for French NIR numbers.
func (d *NIRDetector) Detect(text string) []Match {
	if text == "" {
		return nil
	}

	var matches []Match
	claimed := make(map[int]bool)

	// 1. Scan formatted NIR numbers
	formattedIndices := nirFormattedRegex.FindAllStringIndex(text, -1)
	for _, idx := range formattedIndices {
		start, end := idx[0], idx[1]
		candidate := text[start:end]
		if ValidateNIR(candidate) {
			ctxText := ExtractSurroundingContext(text, start, end, 5)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())

			score := 0.98
			if len(ctxFound) > 0 {
				score = 1.0
			}

			matches = append(matches, Match{
				Type:            EntityNIR,
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

	// 2. Scan continuous 15-digit NIR candidates (requires context)
	rawIndices := nirRawRegex.FindAllStringIndex(text, -1)
	for _, idx := range rawIndices {
		start, end := idx[0], idx[1]
		if claimed[start] || claimed[end-1] {
			continue
		}

		candidate := text[start:end]
		if ValidateNIR(candidate) {
			ctxText := ExtractSurroundingContext(text, start, end, 6)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())

			if len(ctxFound) == 0 {
				continue
			}

			matches = append(matches, Match{
				Type:            EntityNIR,
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

// ValidateNIR verifies candidate string using official INSEE Modulo 97 calculation.
func ValidateNIR(raw string) bool {
	var clean []rune
	for _, r := range raw {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			clean = append(clean, unicode.ToUpper(r))
		}
	}

	if len(clean) != 15 {
		return false
	}

	// Gender digit: 1 (male), 2 (female), 7 or 8 (temporary/specific)
	if clean[0] != '1' && clean[0] != '2' && clean[0] != '7' && clean[0] != '8' {
		return false
	}

	// Parse control key (last 2 digits)
	expectedKey, err := strconv.ParseInt(string(clean[13:15]), 10, 64)
	if err != nil {
		return false
	}

	baseStr := string(clean[:13])

	// Handle Corsica department (2A or 2B in characters 5-6)
	var num int64
	if strings.Contains(baseStr, "2A") {
		corsicaReplaced := strings.Replace(baseStr, "2A", "19", 1)
		n, err := strconv.ParseInt(corsicaReplaced, 10, 64)
		if err != nil {
			return false
		}
		num = n
	} else if strings.Contains(baseStr, "2B") {
		corsicaReplaced := strings.Replace(baseStr, "2B", "18", 1)
		n, err := strconv.ParseInt(corsicaReplaced, 10, 64)
		if err != nil {
			return false
		}
		num = n
	} else {
		n, err := strconv.ParseInt(baseStr, 10, 64)
		if err != nil {
			return false
		}
		num = n
	}

	calculatedKey := 97 - (num % 97)
	return calculatedKey == expectedKey
}

