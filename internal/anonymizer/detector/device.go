package detector

import (
	"regexp"
)

// imeiRegex matches 15-digit IMEI candidate numbers.
var imeiRegex = regexp.MustCompile(`\b[0-9]{15}\b`)

// macRegex matches standard IEEE 802 MAC addresses (e.g. 00:1A:2B:3C:4D:5E or 00-1A-2B-3C-4D-5E).
var macRegex = regexp.MustCompile(`(?i)\b(?:[0-9A-F]{2}[:-]){5}[0-9A-F]{2}\b`)

// DeviceDetector detects hardware and network device identifiers: IMEI and MAC addresses.
type DeviceDetector struct{}

// NewDeviceDetector creates a new DeviceDetector.
func NewDeviceDetector() *DeviceDetector {
	return &DeviceDetector{}
}

// Type returns EntityDevice.
func (d *DeviceDetector) Type() EntityType {
	return EntityDevice
}

// ContextWords returns telecommunications and network hardware context keywords.
func (d *DeviceDetector) ContextWords() []string {
	return []string{
		"imei", "dispositivo", "celular", "smartphone", "aparelho", "terminal", "modem",
		"mac", "mac address", "endereco mac", "endereço mac", "ethernet", "wi-fi", "wifi", "rede",
	}
}

// Detect scans input text for IMEI and MAC addresses.
func (d *DeviceDetector) Detect(text string) []Match {
	if text == "" {
		return nil
	}

	var matches []Match
	claimed := make(map[int]bool)

	// 1. Scan for MAC Addresses
	macIndices := macRegex.FindAllStringIndex(text, -1)
	for _, idx := range macIndices {
		start, end := idx[0], idx[1]
		candidate := text[start:end]

		ctxText := ExtractSurroundingContext(text, start, end, 5)
		ctxFound := FindContextKeywords(ctxText, d.ContextWords())

		score := 0.95
		if len(ctxFound) > 0 {
			score = 1.0
		}

		matches = append(matches, Match{
			Type:            EntityDevice,
			SubType:         "MAC_ADDRESS",
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

	// 2. Scan for IMEI (15 digits with Luhn checksum)
	imeiIndices := imeiRegex.FindAllStringIndex(text, -1)
	for _, idx := range imeiIndices {
		start, end := idx[0], idx[1]
		if claimed[start] || claimed[end-1] {
			continue
		}

		candidate := text[start:end]
		if luhnCheck(candidate) {
			ctxText := ExtractSurroundingContext(text, start, end, 5)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())

			// Strictly require context for IMEI to avoid clashing with 15-digit CNS/French NIR or generic numbers
			if len(ctxFound) == 0 {
				continue
			}

			matches = append(matches, Match{
				Type:            EntityDevice,
				SubType:         "IMEI",
				Value:           candidate,
				Start:           start,
				End:             end,
				Score:           1.0,
				ContextFound:    ctxFound,
				ValidatorPassed: true,
			})
		}
	}

	return matches
}
