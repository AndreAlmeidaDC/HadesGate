package detector

import (
	"regexp"
	"strings"
	"unicode"
)

// placaMercosulRegex matches Brazilian Mercosul plates (AAA1B23 or AAA-1B23).
var placaMercosulRegex = regexp.MustCompile(`(?i)\b[A-Z]{3}[ -]?[0-9][A-Z][0-9]{2}\b`)

// placaAntigaRegex matches Brazilian standard old plates (AAA-1234 or AAA 1234).
var placaAntigaRegex = regexp.MustCompile(`(?i)\b[A-Z]{3}[ -][0-9]{4}\b`)

// renavamRegex matches 11-digit RENAVAM candidates.
var renavamRegex = regexp.MustCompile(`\b[0-9]{11}\b`)

// vinRegex matches 17-character VIN / Chassi format.
var vinRegex = regexp.MustCompile(`\b[A-HJ-NPR-Z0-9]{17}\b`)

// vinWeights defines ISO 3779 checksum calculation weights.
var vinWeights = [17]int{8, 7, 6, 5, 4, 3, 2, 10, 0, 9, 8, 7, 6, 5, 4, 3, 2}

// vinLetterValues maps valid VIN letters to their ISO 3779 values.
var vinLetterValues = map[rune]int{
	'A': 1, 'B': 2, 'C': 3, 'D': 4, 'E': 5, 'F': 6, 'G': 7, 'H': 8,
	'J': 1, 'K': 2, 'L': 3, 'M': 4, 'N': 5, 'P': 7, 'R': 9,
	'S': 2, 'T': 3, 'U': 4, 'V': 5, 'W': 6, 'X': 7, 'Y': 8, 'Z': 9,
}

// VehicleDetector detects vehicle identifiers: License Plates, RENAVAM, and VIN/Chassi.
type VehicleDetector struct{}

// NewVehicleDetector creates a new VehicleDetector.
func NewVehicleDetector() *VehicleDetector {
	return &VehicleDetector{}
}

// Type returns EntityVehicle.
func (d *VehicleDetector) Type() EntityType {
	return EntityVehicle
}

// ContextWords returns vehicle-related keywords.
func (d *VehicleDetector) ContextWords() []string {
	return []string{
		"placa", "veiculo", "veículo", "carro", "automovel", "automóvel", "caminhao", "caminhão",
		"moto", "frota", "multa", "detran", "denatran", "renavam", "chassi", "vin", "ipva", "licenciamento",
	}
}

// Detect scans input text for vehicle-related sensitive identifiers.
func (d *VehicleDetector) Detect(text string) []Match {
	if text == "" {
		return nil
	}

	var matches []Match
	claimed := make(map[int]bool)

	// 1. Scan for VIN / Chassi (17 characters with ISO 3779 check digit)
	vinIndices := vinRegex.FindAllStringIndex(text, -1)
	for _, idx := range vinIndices {
		start, end := idx[0], idx[1]
		candidate := text[start:end]
		if ValidateVIN(candidate) {
			ctxText := ExtractSurroundingContext(text, start, end, 5)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())

			score := 0.96
			if len(ctxFound) > 0 {
				score = 1.0
			}

			matches = append(matches, Match{
				Type:            EntityVehicle,
				SubType:         "VIN",
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

	// 2. Scan for RENAVAM (11 digits with Modulo 11 check digit)
	renavamIndices := renavamRegex.FindAllStringIndex(text, -1)
	for _, idx := range renavamIndices {
		start, end := idx[0], idx[1]
		if claimed[start] || claimed[end-1] {
			continue
		}

		candidate := text[start:end]
		if ValidateRENAVAM(candidate) {
			ctxText := ExtractSurroundingContext(text, start, end, 5)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())

			// Strictly require vehicle context for RENAVAM to prevent collisions with 11-digit numbers
			if len(ctxFound) == 0 {
				continue
			}

			matches = append(matches, Match{
				Type:            EntityVehicle,
				SubType:         "RENAVAM",
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

	// 3. Scan for License Plates (Mercosul and Antiga)
	allPlacaIndices := append(placaMercosulRegex.FindAllStringIndex(text, -1), placaAntigaRegex.FindAllStringIndex(text, -1)...)
	for _, idx := range allPlacaIndices {
		start, end := idx[0], idx[1]
		if claimed[start] || claimed[end-1] {
			continue
		}

		candidate := text[start:end]
		ctxText := ExtractSurroundingContext(text, start, end, 5)
		ctxFound := FindContextKeywords(ctxText, d.ContextWords())

		// Placas formatadas com espaço/hífen exigem contexto de trânsito/veículo para evitar siglas genéricas
		if len(ctxFound) == 0 {
			continue
		}

		matches = append(matches, Match{
			Type:            EntityVehicle,
			SubType:         "LICENSE_PLATE",
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

	return matches
}

// ValidateRENAVAM verifies whether candidate is a valid 11-digit RENAVAM code using DENATRAN Modulo 11.
func ValidateRENAVAM(candidate string) bool {
	if len(candidate) != 11 {
		return false
	}

	// Reject all-zeros or all-identical numbers
	allEqual := true
	for i := 1; i < 11; i++ {
		if candidate[i] != candidate[0] {
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
		r := candidate[i]
		if !unicode.IsDigit(rune(r)) {
			return false
		}
		sum += int(r-'0') * multipliers[i]
	}

	mod := (sum * 10) % 11
	if mod == 10 {
		mod = 0
	}

	return mod == int(candidate[10]-'0')
}

// ValidateVIN verifies whether candidate complies with the ISO 3779 17-character VIN standard.
func ValidateVIN(candidate string) bool {
	upper := strings.ToUpper(strings.TrimSpace(candidate))
	if len(upper) != 17 {
		return false
	}

	sum := 0
	for i := 0; i < 17; i++ {
		r := rune(upper[i])
		var val int
		if unicode.IsDigit(r) {
			val = int(r - '0')
		} else if v, ok := vinLetterValues[r]; ok {
			val = v
		} else {
			return false // Invalid character in VIN (e.g. I, O, Q)
		}

		sum += val * vinWeights[i]
	}

	remainder := sum % 11
	var expectedCheckChar rune
	if remainder == 10 {
		expectedCheckChar = 'X'
	} else {
		expectedCheckChar = rune('0' + remainder)
	}

	return rune(upper[8]) == expectedCheckChar
}

