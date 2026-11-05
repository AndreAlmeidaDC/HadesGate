package detector

import (
	"regexp"
	"strings"
	"unicode"
)

// ibanCountryLengths maps ISO 3166-1 alpha-2 country codes to their official IBAN total character length.
var ibanCountryLengths = map[string]int{
	"AD": 24, "AE": 23, "AL": 28, "AT": 20, "AZ": 28, "BA": 20, "BE": 16, "BG": 22,
	"BH": 22, "BR": 29, "BY": 28, "CH": 21, "CR": 22, "CY": 28, "CZ": 24, "DE": 22,
	"DK": 18, "DO": 28, "EE": 20, "EG": 29, "ES": 24, "FI": 18, "FO": 18, "FR": 27,
	"GB": 22, "GE": 22, "GI": 23, "GL": 18, "GR": 27, "GT": 28, "HR": 21, "HU": 28,
	"IE": 22, "IL": 23, "IQ": 23, "IS": 26, "IT": 27, "JO": 30, "KW": 30, "KZ": 20,
	"LB": 28, "LC": 32, "LI": 21, "LT": 20, "LU": 20, "LV": 21, "MC": 27, "MD": 24,
	"ME": 22, "MK": 19, "MR": 27, "MT": 31, "MU": 30, "NL": 18, "NO": 15, "PK": 24,
	"PL": 28, "PS": 29, "PT": 25, "QA": 29, "RO": 24, "RS": 22, "SA": 24, "SC": 31,
	"SE": 24, "SI": 19, "SK": 24, "SM": 27, "ST": 25, "SV": 28, "TL": 23, "TN": 24,
	"TR": 26, "UA": 29, "VA": 22, "VG": 24, "XK": 20,
}

// ibanCandidateRegex harvests wide candidate IBAN strings (with optional space/hyphen grouping).
var ibanCandidateRegex = regexp.MustCompile(`\b[A-Z]{2}[0-9]{2}(?:[ -]?[A-Z0-9]{1,4}){3,9}\b`)

// IBANDetector detects International Bank Account Numbers (IBAN) using ISO 13616 Mod-97 verification.
type IBANDetector struct{}

// NewIBANDetector creates a new IBANDetector.
func NewIBANDetector() *IBANDetector {
	return &IBANDetector{}
}

// Type returns EntityIBAN.
func (d *IBANDetector) Type() EntityType {
	return EntityIBAN
}

// ContextWords returns keywords associated with IBANs and international wire transfers.
func (d *IBANDetector) ContextWords() []string {
	return []string{
		"iban", "swift", "bic", "sepa", "wire", "transfer", "transferencia", "transferência",
		"conta internacional", "bank", "banco", "beneficiario", "beneficiário",
	}
}

// Detect scans text for valid IBANs, applying trailing-token peeling when prose words follow space-separated IBANs.
func (d *IBANDetector) Detect(text string) []Match {
	indices := ibanCandidateRegex.FindAllStringIndex(text, -1)
	if len(indices) == 0 {
		return nil
	}

	var matches []Match
	for _, idx := range indices {
		start, end := idx[0], idx[1]
		candidate := text[start:end]

		// Wide-input, strict-output: if trailing space-separated prose words were captured
		// (e.g. "DE89 3704 0044 0532 0130 00 NOW"), iteratively peel trailing tokens until valid.
		validCandidate, validEnd := peelValidIBAN(candidate, start, end)
		if validCandidate == "" {
			continue
		}

		ctxText := ExtractSurroundingContext(text, start, validEnd, 5)
		ctxFound := FindContextKeywords(ctxText, d.ContextWords())

		matches = append(matches, Match{
			Type:            EntityIBAN,
			Value:           validCandidate,
			Start:           start,
			End:             validEnd,
			Score:           1.0,
			ContextFound:    ctxFound,
			ValidatorPassed: true,
		})
	}

	return matches
}

func peelValidIBAN(candidate string, start, end int) (string, int) {
	curr := candidate
	currEnd := end
	for len(curr) >= 15 {
		if ValidateIBAN(curr) {
			return curr, currEnd
		}
		lastSep := strings.LastIndexAny(curr, " -")
		if lastSep == -1 {
			break
		}
		trimmed := strings.TrimRight(curr[:lastSep], " -")
		currEnd = start + len(trimmed)
		curr = trimmed
	}
	return "", 0
}

// ValidateIBAN checks country length and ISO 13616 Mod-97 checksum.
func ValidateIBAN(raw string) bool {
	var clean strings.Builder
	clean.Grow(len(raw))
	for _, r := range raw {
		if r == ' ' || r == '-' {
			continue
		}
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			clean.WriteRune(r)
		} else if r >= 'a' && r <= 'z' {
			clean.WriteRune(unicode.ToUpper(r))
		} else {
			return false
		}
	}

	s := clean.String()
	if len(s) < 15 || len(s) > 34 {
		return false
	}

	country := s[:2]
	expectedLen, ok := ibanCountryLengths[country]
	if !ok || len(s) != expectedLen {
		return false
	}

	// Check digits at positions 2 and 3 must be numeric
	if s[2] < '0' || s[2] > '9' || s[3] < '0' || s[3] > '9' {
		return false
	}

	// ISO 13616 Mod-97: move first 4 chars to end, expand A=10..Z=35, compute mod 97 == 1
	rearranged := s[4:] + s[:4]
	rem := 0
	for i := 0; i < len(rearranged); i++ {
		c := rearranged[i]
		if c >= '0' && c <= '9' {
			rem = (rem*10 + int(c-'0')) % 97
		} else if c >= 'A' && c <= 'Z' {
			val := int(c-'A') + 10
			rem = (rem*100 + val) % 97
		} else {
			return false
		}
	}

	return rem == 1
}
