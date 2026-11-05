package detector

import (
	"fmt"
	"regexp"
	"strconv"
)

// cnjProcessRegex matches standard formatted Brazilian legal process numbers (NNNNNNN-DD.AAAA.J.TR.OOOO).
var cnjProcessRegex = regexp.MustCompile(`\b\d{7}-\d{2}\.\d{4}\.\d\.\d{2}\.\d{4}\b`)

// LegalDetector detects Brazilian judicial lawsuit numbers (CNJ) using official Modulo 97 verification.
type LegalDetector struct{}

// NewLegalDetector creates a new LegalDetector.
func NewLegalDetector() *LegalDetector {
	return &LegalDetector{}
}

// Type returns EntityLegal.
func (d *LegalDetector) Type() EntityType {
	return EntityLegal
}

// ContextWords returns judicial and legal context keywords.
func (d *LegalDetector) ContextWords() []string {
	return []string{
		"processo", "processo judicial", "autos", "vara", "tribunal", "cnj", "justiça", "justica",
		"segredo de justica", "segredo de justiça", "inquerito", "inquérito", "comarca", "forum",
		"fórum", "acao judicial", "ação judicial", "peticao", "petição",
	}
}

// Detect scans input text for Brazilian CNJ legal processes.
func (d *LegalDetector) Detect(text string) []Match {
	if text == "" {
		return nil
	}

	indices := cnjProcessRegex.FindAllStringIndex(text, -1)
	if len(indices) == 0 {
		return nil
	}

	var matches []Match
	for _, idx := range indices {
		start, end := idx[0], idx[1]
		candidate := text[start:end]

		if ValidateCNJProcess(candidate) {
			ctxText := ExtractSurroundingContext(text, start, end, 5)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())

			score := 0.98
			if len(ctxFound) > 0 {
				score = 1.0
			}

			matches = append(matches, Match{
				Type:            EntityLegal,
				SubType:         "CNJ_PROCESS",
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

// ValidateCNJProcess verifies a 20-digit CNJ process number using the official CNJ Resolução 65 Modulo 97 standard.
func ValidateCNJProcess(candidate string) bool {
	// Pattern: NNNNNNN-DD.AAAA.J.TR.OOOO (length 25 characters with separators)
	if len(candidate) != 25 {
		return false
	}

	// Extract components
	numPart := candidate[0:7]     // NNNNNNN
	checkPart := candidate[8:10]   // DD
	yearPart := candidate[11:15]   // AAAA
	judPart := candidate[16:17]    // J
	tribPart := candidate[18:20]   // TR
	origPart := candidate[21:25]   // OOOO

	expectedCheck, err := strconv.Atoi(checkPart)
	if err != nil {
		return false
	}

	n, err := strconv.ParseInt(numPart, 10, 64)
	if err != nil {
		return false
	}

	r1 := n % 97

	// Block 2: (R1 * 10^9) + (AAAA * 10^5) + (J * 10^4) + (TR * 10^2) + OOOO
	middleStr := fmt.Sprintf("%d%s%s%s%s", r1, yearPart, judPart, tribPart, origPart)
	middleVal, err := strconv.ParseInt(middleStr, 10, 64)
	if err != nil {
		return false
	}

	r2 := middleVal % 97
	r3 := (r2 * 100) % 97

	calculatedCheck := 98 - r3
	if calculatedCheck == 0 {
		calculatedCheck = 97
	}

	return int(calculatedCheck) == expectedCheck
}

