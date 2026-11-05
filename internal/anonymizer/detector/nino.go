package detector

import (
	"regexp"
	"unicode"
)

// ninoRegex matches candidates for the UK National Insurance Number.
var ninoRegex = regexp.MustCompile(`(?i)\b[A-Z]{2}(?:[ -]?[0-9]{2}){3}(?:[ -]?[A-D])?\b`)

var ninoCompetingPassportKeywords = []string{
	"passaporte", "passport", "embarque", "viagem", "policia federal", "dpf", "imigracao", "imigração",
}

// invalidNINOPrefixes lists 2-letter prefix combinations explicitly prohibited by HMRC.
var invalidNINOPrefixes = map[string]bool{
	"BG": true, "GB": true, "KN": true, "NK": true, "NT": true, "TN": true, "ZZ": true,
}

// disallowedNINOChars1 contains characters disallowed as the 1st letter of a NINO: D, F, I, Q, U, V.
var disallowedNINOChars1 = map[rune]bool{
	'D': true, 'F': true, 'I': true, 'Q': true, 'U': true, 'V': true,
}

// disallowedNINOChars2 contains characters disallowed as the 2nd letter of a NINO: D, F, I, O, Q, U, V.
var disallowedNINOChars2 = map[rune]bool{
	'D': true, 'F': true, 'I': true, 'O': true, 'Q': true, 'U': true, 'V': true,
}

// NINODetector detects UK National Insurance Numbers (NINO).
type NINODetector struct{}

// NewNINODetector creates a new NINODetector.
func NewNINODetector() *NINODetector {
	return &NINODetector{}
}

// Type returns EntityNINO.
func (d *NINODetector) Type() EntityType {
	return EntityNINO
}

// ContextWords returns keywords that provide contextual evidence for UK NINO.
func (d *NINODetector) ContextWords() []string {
	return []string{
		"nino", "national insurance", "ni number", "ni no", "hmrc", "dwp", "social security",
	}
}

// Detect scans input text for valid UK National Insurance Numbers.
func (d *NINODetector) Detect(text string) []Match {
	if text == "" {
		return nil
	}

	indices := ninoRegex.FindAllStringIndex(text, -1)
	if len(indices) == 0 {
		return nil
	}

	var matches []Match
	for _, idx := range indices {
		start, end := idx[0], idx[1]
		candidate := text[start:end]

		if ValidateNINO(candidate) {
			ctxText := ExtractSurroundingContext(text, start, end, 5)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())

			// If competing passport context is present and no NINO context, yield to PassportDetector
			if len(FindContextKeywords(ctxText, ninoCompetingPassportKeywords)) > 0 && len(ctxFound) == 0 {
				continue
			}

			// Clean candidate length (8 chars without suffix, 9 chars with suffix A-D)
			cleanLen := 0
			for _, r := range candidate {
				if unicode.IsLetter(r) || unicode.IsDigit(r) {
					cleanLen++
				}
			}

			// An 8-character candidate without suffix (2 letters + 6 digits) strictly requires NINO context
			// to avoid conflicting with passports or foreign identifiers.
			if cleanLen == 8 && len(ctxFound) == 0 {
				continue
			}

			score := 0.96
			if len(ctxFound) > 0 {
				score = 1.0
			}

			matches = append(matches, Match{
				Type:            EntityNINO,
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

// ValidateNINO verifies candidate string against official HMRC rules.
func ValidateNINO(raw string) bool {
	// Clean candidate string of spaces and hyphens
	var clean []rune
	for _, r := range raw {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			clean = append(clean, unicode.ToUpper(r))
		}
	}

	// Length must be 8 (without suffix) or 9 (with suffix A-D)
	if len(clean) != 8 && len(clean) != 9 {
		return false
	}

	// First two characters must be letters
	if !unicode.IsLetter(clean[0]) || !unicode.IsLetter(clean[1]) {
		return false
	}

	// First letter check
	if disallowedNINOChars1[clean[0]] {
		return false
	}

	// Second letter check
	if disallowedNINOChars2[clean[1]] {
		return false
	}

	// Disallowed combinations
	prefix := string(clean[:2])
	if invalidNINOPrefixes[prefix] {
		return false
	}

	// Next 6 characters must be digits
	for i := 2; i < 8; i++ {
		if !unicode.IsDigit(clean[i]) {
			return false
		}
	}

	// 9th character (suffix), if present, must be A, B, C, or D
	if len(clean) == 9 {
		suffix := clean[8]
		if suffix != 'A' && suffix != 'B' && suffix != 'C' && suffix != 'D' {
			return false
		}
	}

	return true
}
