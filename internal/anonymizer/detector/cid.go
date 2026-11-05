package detector

import (
	"regexp"
)

// cidWithPrefixRegex matches CID-10 and CID-11 codes preceded by explicit CID markers.
var cidWithPrefixRegex = regexp.MustCompile(`(?i)\b(?:CID(?:[- ]?1[01])?)\s*[:#\-]?\s*([A-TV-Z][0-9]{2}(?:\.[0-9]{1,2})?|[0-9A-Z]{4})\b`)

// cidCodeOnlyRegex matches standalone CID-10 formatted codes (e.g. J45.0, F32.1, E11.9, I10).
var cidCodeOnlyRegex = regexp.MustCompile(`\b[A-TV-Z][0-9]{2}\.[0-9]{1,2}\b`)

// CIDDetector detects International Classification of Diseases (CID-10 / CID-11) diagnostic codes.
type CIDDetector struct{}

// NewCIDDetector creates a new CIDDetector.
func NewCIDDetector() *CIDDetector {
	return &CIDDetector{}
}

// Type returns EntityCID.
func (d *CIDDetector) Type() EntityType {
	return EntityCID
}

// ContextWords returns healthcare diagnostic context keywords.
func (d *CIDDetector) ContextWords() []string {
	return []string{
		"cid", "cid-10", "cid-11", "cid10", "cid11", "diagnostico", "diagnóstico", "patologia",
		"doenca", "doença", "atestado", "laudo", "medico", "médico", "clinico", "clínico",
		"quadro clinico", "quadro clínico", "hipotese diagnostica", "hipótese diagnóstica",
	}
}

// Detect scans input text for CID diagnostic codes.
func (d *CIDDetector) Detect(text string) []Match {
	if text == "" {
		return nil
	}

	var matches []Match
	claimed := make(map[int]bool)

	// 1. Scan for prefixed CID mentions (e.g. "CID-10 J45.0" or "CID: F32.1")
	submatches := cidWithPrefixRegex.FindAllStringSubmatchIndex(text, -1)
	for _, idx := range submatches {
		if len(idx) < 4 {
			continue
		}
		valStart, valEnd := idx[2], idx[3]
		candidate := text[valStart:valEnd]

		ctxText := ExtractSurroundingContext(text, valStart, valEnd, 5)
		ctxFound := FindContextKeywords(ctxText, d.ContextWords())

		matches = append(matches, Match{
			Type:            EntityCID,
			Value:           candidate,
			Start:           valStart,
			End:             valEnd,
			Score:           0.98,
			ContextFound:    ctxFound,
			ValidatorPassed: true,
		})
		for k := valStart; k < valEnd; k++ {
			claimed[k] = true
		}
	}

	// 2. Scan for standalone dotted CID codes (e.g. "J45.0" with explicit diagnostic context)
	codeIndices := cidCodeOnlyRegex.FindAllStringIndex(text, -1)
	for _, idx := range codeIndices {
		start, end := idx[0], idx[1]
		if claimed[start] || claimed[end-1] {
			continue
		}

		ctxText := ExtractSurroundingContext(text, start, end, 5)
		ctxFound := FindContextKeywords(ctxText, d.ContextWords())

		// Strictly require diagnostic context for dotted codes without CID prefix
		if len(ctxFound) == 0 {
			continue
		}

		candidate := text[start:end]
		matches = append(matches, Match{
			Type:            EntityCID,
			Value:           candidate,
			Start:           start,
			End:             end,
			Score:           0.92,
			ContextFound:    ctxFound,
			ValidatorPassed: true,
		})
		for k := start; k < end; k++ {
			claimed[k] = true
		}
	}

	return matches
}
