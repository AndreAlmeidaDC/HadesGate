package detector

import (
	"regexp"
	"strings"
	"unicode"
)

// rgRegex matches standard formatted or raw Brazilian RG (7 to 9 digits, e.g. 12.345.678-9, 3.109.380, 3109380, MG-12.345.678).
var rgRegex = regexp.MustCompile(`\b(?:[a-zA-Z]{2}[- ]?)?\d{1,2}\.?\d{3}\.?\d{3}(?:-?[0-9xX])?\b`)

// RGDetector identifies Brazilian General Identity (RG) numbers.
type RGDetector struct{}

// NewRGDetector creates a new RGDetector.
func NewRGDetector() *RGDetector {
	return &RGDetector{}
}

// Type returns EntityRG.
func (d *RGDetector) Type() EntityType {
	return EntityRG
}

// ContextWords returns keywords providing contextual evidence for RG numbers.
func (d *RGDetector) ContextWords() []string {
	return []string{
		"rg", "registro geral", "identidade", "carteira de identidade",
		"cedula de identidade", "ssp", "orgao emissor", "expedicao", "segunda via",
	}
}

// Detect scans text for Brazilian RG documents.
func (d *RGDetector) Detect(text string) []Match {
	indices := rgRegex.FindAllStringIndex(text, -1)
	if len(indices) == 0 {
		return nil
	}

	matches := make([]Match, 0, len(indices))
	for _, idx := range indices {
		candidate := text[idx[0]:idx[1]]

		// Count clean characters (digits or 'X')
		cleanLen := 0
		for _, r := range candidate {
			if unicode.IsDigit(r) || r == 'x' || r == 'X' {
				cleanLen++
			}
		}

		if cleanLen < 7 || cleanLen > 9 {
			continue
		}

		// If candidate matched "RG " or "RG:" prefix, trim it so only the document number is masked
		startOffset := 0
		val := candidate
		lowerCand := strings.ToLower(candidate)
		if strings.HasPrefix(lowerCand, "rg") && len(candidate) > 2 && (candidate[2] == ' ' || candidate[2] == ':' || candidate[2] == '-') {
			startOffset = 2
			for startOffset < len(candidate) && (candidate[startOffset] == ' ' || candidate[startOffset] == ':' || candidate[startOffset] == '-') {
				startOffset++
			}
			val = candidate[startOffset:]
		}

		// Recalculate clean characters for the actual number value
		cleanLen = 0
		for _, r := range val {
			if unicode.IsDigit(r) || r == 'x' || r == 'X' {
				cleanLen++
			}
		}
		if cleanLen < 7 || cleanLen > 9 {
			continue
		}

		ctxText := ExtractSurroundingContext(text, idx[0], idx[1], 5)
		combinedCtx := strings.ToLower(candidate + " " + ctxText)
		ctxFound := FindContextKeywords(combinedCtx, d.ContextWords())

		// RG document numbers MUST require contextual evidence to avoid false positives with regular order/item numbers
		score := 0.30
		hasPunctuation := strings.Contains(val, ".") || strings.Contains(val, "-")
		if len(ctxFound) > 0 {
			if hasPunctuation {
				score = 0.95
			} else {
				score = 0.85
			}
		}

		matches = append(matches, Match{
			Type:            EntityRG,
			Value:           val,
			Start:           idx[0] + startOffset,
			End:             idx[1],
			Score:           score,
			ContextFound:    ctxFound,
			ValidatorPassed: len(ctxFound) > 0,
		})
	}

	return matches
}
