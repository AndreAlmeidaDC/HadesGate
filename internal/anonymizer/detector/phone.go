package detector

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// phoneRegex matches Brazilian landline and mobile phone patterns with optional country code (+55).
var phoneRegex = regexp.MustCompile(`(?:\+55\s?)?(?:\(?([1-9][0-9])\)?\s?)?(?:(?:9\s?\d{4}[-\s]?\d{4})|(?:[2-8]\d{3}[-\s]?\d{4}))\b`)

// intlPhoneRegex matches international E.164 formatted phone numbers starting with '+' (e.g. +1 (415) 555-2671, +351 912 345 678).
var intlPhoneRegex = regexp.MustCompile(`\+[1-9]\d{0,2}[\s.-]?\(?\d{1,4}\)?[\s.-]?\d{3,4}[\s.-]?\d{3,4}\b`)

// validDDDs is a map of officially valid Brazilian area codes (DDDs).
var validDDDs = map[int]bool{
	11: true, 12: true, 13: true, 14: true, 15: true, 16: true, 17: true, 18: true, 19: true,
	21: true, 22: true, 24: true, 27: true, 28: true,
	31: true, 32: true, 33: true, 34: true, 35: true, 37: true, 38: true,
	41: true, 42: true, 43: true, 44: true, 45: true, 46: true, 47: true, 48: true, 49: true,
	51: true, 53: true, 54: true, 55: true,
	61: true, 62: true, 63: true, 64: true, 65: true, 66: true, 67: true, 68: true, 69: true,
	71: true, 73: true, 74: true, 75: true, 77: true, 79: true,
	81: true, 82: true, 83: true, 84: true, 85: true, 86: true, 87: true, 88: true, 89: true,
	91: true, 92: true, 93: true, 94: true, 95: true, 96: true, 97: true, 98: true, 99: true,
}

// PhoneDetector detects Brazilian phone numbers.
type PhoneDetector struct{}

// NewPhoneDetector creates a new instance of PhoneDetector.
func NewPhoneDetector() *PhoneDetector {
	return &PhoneDetector{}
}

// Type returns EntityPhone.
func (d *PhoneDetector) Type() EntityType {
	return EntityPhone
}

// ContextWords returns keywords that provide contextual evidence for phone numbers.
func (d *PhoneDetector) ContextWords() []string {
	return []string{"telefone", "tel", "celular", "cel", "contato", "whatsapp", "whats", "zap", "fone", "ligar"}
}

// Detect scans text for Brazilian and international E.164 phone numbers.
func (d *PhoneDetector) Detect(text string) []Match {
	var matches []Match
	claimed := make(map[int]bool)

	indices := phoneRegex.FindAllStringIndex(text, -1)
	for _, idx := range indices {
		candidate := text[idx[0]:idx[1]]
		if ValidatePhone(candidate) {
			for k := idx[0]; k < idx[1]; k++ {
				claimed[k] = true
			}
			ctxText := ExtractSurroundingContext(text, idx[0], idx[1], 5)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())
			score := 0.85
			if len(ctxFound) > 0 {
				score = 0.95
			}
			matches = append(matches, Match{
				Type:            EntityPhone,
				Value:           candidate,
				Start:           idx[0],
				End:             idx[1],
				Score:           score,
				ContextFound:    ctxFound,
				ValidatorPassed: true,
			})
		}
	}

	intlIndices := intlPhoneRegex.FindAllStringIndex(text, -1)
	for _, idx := range intlIndices {
		if claimed[idx[0]] {
			continue
		}
		candidate := text[idx[0]:idx[1]]
		if ValidateInternationalPhone(candidate) {
			ctxText := ExtractSurroundingContext(text, idx[0], idx[1], 5)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())
			score := 0.88
			if len(ctxFound) > 0 {
				score = 0.95
			}
			matches = append(matches, Match{
				Type:            EntityPhone,
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

// ValidateInternationalPhone validates non-Brazilian E.164 phone numbers starting with '+'.
func ValidateInternationalPhone(raw string) bool {
	if !strings.HasPrefix(strings.TrimSpace(raw), "+") {
		return false
	}
	var sb strings.Builder
	for _, r := range raw {
		if unicode.IsDigit(r) {
			sb.WriteRune(r)
		}
	}
	digits := sb.String()
	// E.164 requires 8 to 15 digits total (country code + subscriber number)
	if len(digits) < 9 || len(digits) > 15 {
		return false
	}
	// If it starts with 55 (Brazil), defer to strict Brazilian DDD validation
	if strings.HasPrefix(digits, "55") {
		return ValidatePhone(raw)
	}
	return true
}

// ValidatePhone validates if a string represents a coherent Brazilian phone number.
func ValidatePhone(raw string) bool {
	// Count digits and strip other chars
	var sb strings.Builder
	for _, r := range raw {
		if unicode.IsDigit(r) {
			sb.WriteRune(r)
		}
	}
	digits := sb.String()

	// Handle country code +55
	if strings.HasPrefix(digits, "55") && (len(digits) == 12 || len(digits) == 13) {
		digits = digits[2:]
	}

	// 10 digits = DDD + 8 digit landline/old mobile
	// 11 digits = DDD + 9 digit mobile (starting with 9)
	if len(digits) == 11 {
		ddd, err := strconv.Atoi(digits[:2])
		if err != nil || !validDDDs[ddd] {
			return false
		}
		// 9th digit must be 9
		return digits[2] == '9'
	}

	if len(digits) == 10 {
		ddd, err := strconv.Atoi(digits[:2])
		if err != nil || !validDDDs[ddd] {
			return false
		}
		// Landlines usually start with 2, 3, 4, 5
		first := digits[2]
		return first >= '2' && first <= '5'
	}

	return false
}
