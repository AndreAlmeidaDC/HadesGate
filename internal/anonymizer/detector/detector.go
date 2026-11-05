package detector

import (
	"math"
	"strings"
	"unicode"
	"unicode/utf8"
)

// EntityType represents the category of sensitive PII data.
type EntityType string

const (
	EntityCPF         EntityType = "CPF"
	EntityCNPJ        EntityType = "CNPJ"
	EntityPhone       EntityType = "PHONE"
	EntityEmail       EntityType = "EMAIL"
	EntityCreditCard  EntityType = "CREDIT_CARD"
	EntityName        EntityType = "PERSON"
	EntityCEP         EntityType = "CEP"
	EntityPIX         EntityType = "PIX"
	EntityRG          EntityType = "RG"
	EntityCNH         EntityType = "CNH"
	EntityBankAccount EntityType = "BANK_ACCOUNT"
	EntityIBAN        EntityType = "IBAN"
	EntityPassport    EntityType = "PASSPORT"
	EntityIPAddress   EntityType = "IP_ADDRESS"
	EntityAPIKey      EntityType = "API_KEY"
	EntitySecret      EntityType = "SECRET"
	EntityCustom      EntityType = "CUSTOM"
	EntitySSN         EntityType = "SSN"
	EntityEIN         EntityType = "EIN"
	EntityNINO        EntityType = "NINO"
	EntityNHS         EntityType = "NHS"
	EntityDNI         EntityType = "DNI"
	EntityNIR         EntityType = "NIR"
	EntityCodiceFiscale EntityType = "CODICE_FISCALE"
	EntityMedicalID   EntityType = "MEDICAL_ID"
	EntitySUS         EntityType = "SUS_CARD"
	EntityVehicle     EntityType = "VEHICLE"
	EntityLegal       EntityType = "LEGAL"
	EntityDevice      EntityType = "DEVICE_ID"
	EntityGovernmentID EntityType = "GOVERNMENT_ID"
	EntityCID         EntityType = "CID"
	EntityMedication  EntityType = "MEDICATION"
	EntityCorporate   EntityType = "CORPORATE"
)

// Match represents a detected sensitive piece of information in a text.
type Match struct {
	Type            EntityType `json:"type"`
	SubType         string     `json:"sub_type,omitempty"`    // Optional granular category (e.g. PASSWORD, JWT, DB_URI)
	Value           string     `json:"value"`                 // original sensitive text
	Start           int        `json:"start"`                 // byte start index in original string
	End             int        `json:"end"`                   // byte end index in original string
	Score           float64    `json:"score"`                 // 0.0 to 1.0 confidence score
	ContextFound    []string   `json:"context_found"`         // Context words detected in the surrounding window
	ValidatorPassed bool       `json:"validator_passed"`      // Whether mathematical checksum/verification passed
	IsSemantic      bool       `json:"is_semantic,omitempty"` // True when produced by an external semantic NER sidecar
}

// Detector defines the interface that all PII detectors must implement.
type Detector interface {
	Type() EntityType
	Detect(text string) []Match
}

// ContextAwareDetector is an optional interface for detectors that define specific context keywords.
type ContextAwareDetector interface {
	Detector
	ContextWords() []string
}

// ExtractSurroundingContext returns the surrounding text window (up to maxWords before and after)
// in lowercase for fast substring/word checking. Safe for multi-byte UTF-8 strings.
func ExtractSurroundingContext(text string, start, end, maxWords int) string {
	if text == "" || start < 0 || end > len(text) || start > end {
		return ""
	}

	// 1. Find window before start (scanning backwards rune-by-rune for maxWords)
	wordsBefore := 0
	windowStart := start
	inWord := false
	for i := start; i > 0; {
		r, size := utf8.DecodeLastRuneInString(text[:i])
		nextIdx := i - size
		if !unicode.IsSpace(r) && !inWord {
			inWord = true
			wordsBefore++
			if wordsBefore > maxWords {
				windowStart = i
				break
			}
		} else if unicode.IsSpace(r) {
			inWord = false
		}
		i = nextIdx
		if i == 0 {
			windowStart = 0
		}
	}

	// 2. Find window after end (scanning forwards rune-by-rune for maxWords)
	wordsAfter := 0
	windowEnd := end
	inWord = false
	for i := end; i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		if !unicode.IsSpace(r) && !inWord {
			inWord = true
			wordsAfter++
			if wordsAfter > maxWords {
				windowEnd = i
				break
			}
		} else if unicode.IsSpace(r) {
			inWord = false
		}
		i += size
		if i >= len(text) {
			windowEnd = len(text)
		}
	}

	beforePart := strings.ToLower(text[windowStart:start])
	afterPart := strings.ToLower(text[end:windowEnd])
	return beforePart + " " + afterPart
}

// FindContextKeywords searches for any occurrences of keywords in the context text.
func FindContextKeywords(contextText string, keywords []string) []string {
	if contextText == "" || len(keywords) == 0 {
		return nil
	}
	var found []string
	for _, kw := range keywords {
		kwLower := strings.ToLower(kw)
		if strings.Contains(contextText, kwLower) {
			found = append(found, kw)
		}
	}
	return found
}

// CalculateShannonEntropy computes the Shannon entropy of a string in bits per symbol.
// Mathematical formula: H = -sum(P(c) * log2(P(c))).
// Natural human language/text typically has entropy between 1.5 and 3.0 bits/char.
// Cryptographic keys, secrets and random hashes typically have entropy >= 3.2 - 4.5.
func CalculateShannonEntropy(s string) float64 {
	if len(s) == 0 {
		return 0.0
	}

	var freq [256]int
	for i := 0; i < len(s); i++ {
		freq[s[i]]++
	}

	n := float64(len(s))
	var entropy float64
	for _, count := range freq {
		if count > 0 {
			p := float64(count) / n
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}
