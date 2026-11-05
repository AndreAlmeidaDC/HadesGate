package detector

import (
	"regexp"
	"unicode"
)

// voterIDRegex matches Brazilian Título de Eleitor (12 digits, optional grouping).
var voterIDRegex = regexp.MustCompile(`\b\d{4}[ -]?\d{4}[ -]?\d{4}\b`)

// pisRegex matches 11-digit PIS/PASEP/NIT (e.g. 120.45678.90-1 or continuous).
var pisRegex = regexp.MustCompile(`\b\d{3}\.?\d{5}\.?\d{2}-?\d\b`)

// GovernmentIDDetector detects Brazilian public document identifiers: Título de Eleitor and PIS/PASEP/NIT.
type GovernmentIDDetector struct{}

// NewGovernmentIDDetector creates a new GovernmentIDDetector.
func NewGovernmentIDDetector() *GovernmentIDDetector {
	return &GovernmentIDDetector{}
}

// Type returns EntityGovernmentID.
func (d *GovernmentIDDetector) Type() EntityType {
	return EntityGovernmentID
}

// ContextWords returns Brazilian governmental and civic context keywords.
func (d *GovernmentIDDetector) ContextWords() []string {
	return []string{
		"titulo de eleitor", "título de eleitor", "eleitoral", "zona eleitoral", "secao eleitoral",
		"seção eleitoral", "tse", "tre", "eleicoes", "eleições",
		"pis", "pasep", "nit", "nis", "carteira de trabalho", "ctps", "clt", "inss", "beneficio", "benefício",
	}
}

// Detect scans input text for Título de Eleitor and PIS/PASEP identifiers.
func (d *GovernmentIDDetector) Detect(text string) []Match {
	if text == "" {
		return nil
	}

	var matches []Match
	claimed := make(map[int]bool)

	// 1. Scan for Título de Eleitor (12 digits with TSE dual-digit checksum)
	voterIndices := voterIDRegex.FindAllStringIndex(text, -1)
	for _, idx := range voterIndices {
		start, end := idx[0], idx[1]
		candidate := text[start:end]
		if ValidateVoterID(candidate) {
			ctxText := ExtractSurroundingContext(text, start, end, 5)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())

			// Strictly require context for Título de Eleitor to avoid false positives with 12-digit numbers
			if len(ctxFound) == 0 {
				continue
			}

			matches = append(matches, Match{
				Type:            EntityGovernmentID,
				SubType:         "VOTER_ID",
				Value:           candidate,
				Start:           start,
				End:             end,
				Score:           0.96,
				ContextFound:    ctxFound,
				ValidatorPassed: true,
			})
			for k := start; k < end; k++ {
				claimed[k] = true
			}
		}
	}

	// 2. Scan for PIS / PASEP / NIT (11 digits with official Modulo 11)
	pisIndices := pisRegex.FindAllStringIndex(text, -1)
	for _, idx := range pisIndices {
		start, end := idx[0], idx[1]
		if claimed[start] || claimed[end-1] {
			continue
		}

		candidate := text[start:end]
		if ValidatePIS(candidate) {
			ctxText := ExtractSurroundingContext(text, start, end, 5)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())

			// Strictly require labor/PIS context to prevent collision with CPF or 11-digit numbers
			if len(ctxFound) == 0 {
				continue
			}

			matches = append(matches, Match{
				Type:            EntityGovernmentID,
				SubType:         "PIS_PASEP",
				Value:           candidate,
				Start:           start,
				End:             end,
				Score:           0.95,
				ContextFound:    ctxFound,
				ValidatorPassed: true,
			})
			for k := start; k < end; k++ {
				claimed[k] = true
			}
		}
	}

	return matches
}

// ValidateVoterID validates a 12-digit Brazilian Título de Eleitor per official TSE Modulo 11 specification.
func ValidateVoterID(raw string) bool {
	digits := make([]int, 0, 12)
	for _, r := range raw {
		if unicode.IsDigit(r) {
			digits = append(digits, int(r-'0'))
		}
	}

	if len(digits) != 12 {
		return false
	}

	uf := digits[8]*10 + digits[9]
	if uf < 1 || uf > 28 {
		return false
	}

	// 1. First check digit (digits[10])
	sum1 := 0
	for i := 0; i < 8; i++ {
		sum1 += digits[i] * (2 + i)
	}
	r1 := sum1 % 11
	var d11 int
	if r1 == 10 {
		d11 = 0
	} else if r1 == 0 {
		if uf == 1 || uf == 2 { // SP or MG
			d11 = 1
		} else {
			d11 = 0
		}
	} else {
		d11 = r1
	}

	if digits[10] != d11 {
		return false
	}

	// 2. Second check digit (digits[11])
	sum2 := digits[8]*7 + digits[9]*8 + digits[10]*9
	r2 := sum2 % 11
	var d12 int
	if r2 == 10 {
		d12 = 0
	} else if r2 == 0 {
		if uf == 1 || uf == 2 {
			d12 = 1
		} else {
			d12 = 0
		}
	} else {
		d12 = r2
	}

	return digits[11] == d12
}

// ValidatePIS verifies an 11-digit PIS/PASEP/NIT using the official Modulo 11 checksum.
func ValidatePIS(raw string) bool {
	digits := make([]int, 0, 11)
	for _, r := range raw {
		if unicode.IsDigit(r) {
			digits = append(digits, int(r-'0'))
		}
	}

	if len(digits) != 11 {
		return false
	}

	// Reject identical digits
	allEqual := true
	for i := 1; i < 11; i++ {
		if digits[i] != digits[0] {
			allEqual = false
			break
		}
	}
	if allEqual {
		return false
	}

	multipliers := [10]int{3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	sum := 0
	for i := 0; i < 10; i++ {
		sum += digits[i] * multipliers[i]
	}

	remainder := sum % 11
	checkDigit := 11 - remainder
	if checkDigit == 10 || checkDigit == 11 {
		checkDigit = 0
	}

	return digits[10] == checkDigit
}

