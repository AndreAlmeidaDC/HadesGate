package detector

import (
	"net"
	"regexp"
	"strings"
)

var (
	// ipv4CandidateRegex matches dotted quad candidates.
	ipv4CandidateRegex = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)

	// ipv6CandidateRegex matches standard and compressed IPv6 addresses.
	ipv6CandidateRegex = regexp.MustCompile(`(?i)\b(?:[0-9a-f]{1,4}:){2,7}[0-9a-f]{1,4}\b|\b(?:[0-9a-f]{1,4}:){1,7}:(?:[0-9a-f]{1,4})?\b`)

	// versionPrefixRegex detects version strings immediately preceding a dotted quad (e.g. "v1.2.3.4", "version 1.2.3.4").
	versionPrefixRegex = regexp.MustCompile(`(?i)(?:^|\s)(?:v|ver|version|vers[aã]o|release)\s*$`)
)

// IPDetector identifies IPv4 and IPv6 network addresses.
type IPDetector struct{}

// NewIPDetector creates a new IPDetector.
func NewIPDetector() *IPDetector {
	return &IPDetector{}
}

// Type returns EntityIPAddress.
func (d *IPDetector) Type() EntityType {
	return EntityIPAddress
}

// ContextWords returns keywords providing contextual evidence for IP addresses.
func (d *IPDetector) ContextWords() []string {
	return []string{
		"ip", "ipv4", "ipv6", "host", "servidor", "server", "endereco ip", "endereço ip",
		"client_ip", "remote_addr", "x-forwarded-for", "ssh", "firewall", "gateway", "rede", "subnet",
	}
}

// Detect scans text for valid IPv4 and IPv6 addresses.
func (d *IPDetector) Detect(text string) []Match {
	if text == "" {
		return nil
	}

	var matches []Match

	// 1. Scan IPv4 candidates
	v4Indices := ipv4CandidateRegex.FindAllStringIndex(text, -1)
	for _, idx := range v4Indices {
		// Ignore if part of a 5+ octet number or preceded by a version prefix (e.g. "v1.2.3.4" or "1.2.3.4.5")
		if idx[0] > 0 && (text[idx[0]-1] == '.' || text[idx[0]-1] == 'v' || text[idx[0]-1] == 'V') {
			continue
		}
		if idx[1]+1 < len(text) && text[idx[1]] == '.' && text[idx[1]+1] >= '0' && text[idx[1]+1] <= '9' {
			continue
		}
		prefixStart := idx[0] - 16
		if prefixStart < 0 {
			prefixStart = 0
		}
		if versionPrefixRegex.MatchString(text[prefixStart:idx[0]]) {
			continue
		}

		candidate := text[idx[0]:idx[1]]
		if !ValidateIPv4(candidate) {
			continue
		}

		ctxText := ExtractSurroundingContext(text, idx[0], idx[1], 5)
		ctxFound := FindContextKeywords(ctxText, d.ContextWords())

		score := 0.90
		if len(ctxFound) > 0 {
			score = 0.98
		}

		matches = append(matches, Match{
			Type:            EntityIPAddress,
			Value:           candidate,
			Start:           idx[0],
			End:             idx[1],
			Score:           score,
			ContextFound:    ctxFound,
			ValidatorPassed: true,
		})
	}

	// 2. Scan IPv6 candidates
	v6Indices := ipv6CandidateRegex.FindAllStringIndex(text, -1)
	for _, idx := range v6Indices {
		candidate := text[idx[0]:idx[1]]
		if !ValidateIPv6(candidate) {
			continue
		}

		ctxText := ExtractSurroundingContext(text, idx[0], idx[1], 5)
		ctxFound := FindContextKeywords(ctxText, d.ContextWords())

		score := 0.95
		if len(ctxFound) > 0 {
			score = 0.99
		}

		matches = append(matches, Match{
			Type:            EntityIPAddress,
			Value:           candidate,
			Start:           idx[0],
			End:             idx[1],
			Score:           score,
			ContextFound:    ctxFound,
			ValidatorPassed: true,
		})
	}

	return matches
}

// ValidateIPv4 checks whether raw is a syntactically valid IPv4 address without leading-zero octets.
func ValidateIPv4(raw string) bool {
	parts := strings.Split(raw, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		if len(p) == 0 || (len(p) > 1 && p[0] == '0') {
			return false
		}
	}
	ip := net.ParseIP(raw)
	if ip == nil || ip.To4() == nil {
		return false
	}
	// Reject 0.0.0.0 and 255.255.255.255 broadcast/unspecified placeholders
	if raw == "0.0.0.0" || raw == "255.255.255.255" {
		return false
	}
	return true
}

// ValidateIPv6 checks whether raw is a valid IPv6 address.
func ValidateIPv6(raw string) bool {
	if !strings.Contains(raw, ":") {
		return false
	}
	ip := net.ParseIP(raw)
	return ip != nil && ip.To4() == nil
}
