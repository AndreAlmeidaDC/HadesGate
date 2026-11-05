package detector

import (
	"regexp"
	"strings"
)

// Known deterministic provider patterns (Score: 1.0)
var providerKeyPatterns = []struct {
	name  string
	regex *regexp.Regexp
}{
	{
		name:  "OpenAI Project / Service Account Key",
		regex: regexp.MustCompile(`\bsk-(?:proj|svcacct)-[a-zA-Z0-9_-]{20,}\b`),
	},
	{
		name:  "OpenAI Standard Key",
		regex: regexp.MustCompile(`\bsk-[a-zA-Z0-9]{20,50}\b`),
	},
	{
		name:  "Anthropic Key",
		regex: regexp.MustCompile(`\bsk-ant-[a-zA-Z0-9_-]{20,}\b`),
	},
	{
		name:  "Groq API Key",
		regex: regexp.MustCompile(`\bgsk_[a-zA-Z0-9]{20,}\b`),
	},
	{
		name:  "xAI API Key",
		regex: regexp.MustCompile(`\bxai-[a-zA-Z0-9]{20,}\b`),
	},
	{
		name:  "AWS Access Key ID",
		regex: regexp.MustCompile(`\b(?:AKIA|ABIA|ACCA|ASIA)[0-9A-Z]{16}\b`),
	},
	{
		name:  "GitHub Personal Access Token",
		regex: regexp.MustCompile(`\bgh[pousr]_[a-zA-Z0-9]{36}\b`),
	},
	{
		name:  "GitHub Fine-Grained Token",
		regex: regexp.MustCompile(`\bgithub_pat_[a-zA-Z0-9]{22}_[a-zA-Z0-9]{59}\b`),
	},
	{
		name:  "GitLab Personal Access Token",
		regex: regexp.MustCompile(`\bglpat-[a-zA-Z0-9_-]{20,}\b`),
	},
	{
		name:  "Google / Gemini API Key",
		regex: regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{35}\b`),
	},
	{
		name:  "Slack Token",
		regex: regexp.MustCompile(`\bxox[bpar]-[0-9]{10,13}-[0-9]{10,13}-[a-zA-Z0-9]{24,32}\b`),
	},
	{
		name:  "Stripe Secret Key",
		regex: regexp.MustCompile(`\b[rs]k_(?:live|test)_[0-9a-zA-Z]{24,34}\b`),
	},
	{
		name:  "HuggingFace Token",
		regex: regexp.MustCompile(`\bhf_[a-zA-Z0-9]{34,}\b`),
	},
	{
		name:  "NPM Access Token",
		regex: regexp.MustCompile(`\bnpm_[a-zA-Z0-9]{36}\b`),
	},
	{
		name:  "PyPI Upload Token",
		regex: regexp.MustCompile(`\bpypi-[a-zA-Z0-9_-]{32,}\b`),
	},
	{
		name:  "SendGrid API Key",
		regex: regexp.MustCompile(`\bSG\.[a-zA-Z0-9_-]{16,}\.[a-zA-Z0-9_-]{16,}\b`),
	},
}

// genericKeyAssignmentRegex matches variable assignments and HTTP Bearer/Basic tokens
var genericKeyAssignmentRegex = regexp.MustCompile(`(?i)\b(?:(?:api[_-]?key|access[_-]?key|secret[_-]?key|auth[_-]?token|access[_-]?token|client[_-]?secret|private[_-]?token)\s*[:=]\s*['"]?([a-zA-Z0-9_\-\.]{16,128})['"]?|(?:authorization\s*[:=]\s*)?bearer\s+([a-zA-Z0-9_\-\.=]{16,512}))`)

// APIKeyDetector identifies cloud, AI and service provider API keys and high-entropy tokens.
type APIKeyDetector struct {
	minEntropy float64
}

// NewAPIKeyDetector creates a new instance with standard entropy threshold (3.2 bits/char).
func NewAPIKeyDetector() *APIKeyDetector {
	return &APIKeyDetector{
		minEntropy: 3.2,
	}
}

// Type returns EntityAPIKey.
func (d *APIKeyDetector) Type() EntityType {
	return EntityAPIKey
}

// ContextWords returns keywords associated with API credentials.
func (d *APIKeyDetector) ContextWords() []string {
	return []string{"key", "api_key", "token", "bearer", "secret", "credencial", "authorization", "auth"}
}

// Detect scans text for well-known API keys and high-entropy generic tokens.
func (d *APIKeyDetector) Detect(text string) []Match {
	if text == "" {
		return nil
	}

	var matches []Match
	type byteSpan struct {
		start int
		end   int
	}
	claimed := make(map[byteSpan]bool)

	// 1. Scan deterministic provider patterns first (Score 1.0)
	for _, p := range providerKeyPatterns {
		indices := p.regex.FindAllStringIndex(text, -1)
		for _, idx := range indices {
			span := byteSpan{start: idx[0], end: idx[1]}
			if claimed[span] {
				continue
			}
			claimed[span] = true

			candidate := text[idx[0]:idx[1]]
			ctxText := ExtractSurroundingContext(text, idx[0], idx[1], 5)
			combinedCtx := strings.ToLower(candidate + " " + ctxText)
			ctxFound := FindContextKeywords(combinedCtx, d.ContextWords())

			matches = append(matches, Match{
				Type:            EntityAPIKey,
				Value:           candidate,
				Start:           idx[0],
				End:             idx[1],
				Score:           1.0,
				ContextFound:    ctxFound,
				ValidatorPassed: true,
			})
		}
	}

	// 2. Scan generic contextual assignments and Bearer tokens with Shannon entropy validation
	submatches := genericKeyAssignmentRegex.FindAllStringSubmatchIndex(text, -1)
	for _, sub := range submatches {
		valStart, valEnd := -1, -1
		if len(sub) >= 4 && sub[2] != -1 && sub[3] != -1 {
			valStart, valEnd = sub[2], sub[3]
		} else if len(sub) >= 6 && sub[4] != -1 && sub[5] != -1 {
			valStart, valEnd = sub[4], sub[5]
		}
		if valStart != -1 && valEnd != -1 {
			span := byteSpan{start: valStart, end: valEnd}
			if claimed[span] {
				continue
			}

			val := text[valStart:valEnd]
			// Validate Shannon entropy of the assigned value
			entropy := CalculateShannonEntropy(val)
			if entropy < d.minEntropy {
				continue // Skip low-entropy strings (normal words, repeated characters)
			}

			claimed[span] = true
			ctxText := ExtractSurroundingContext(text, valStart, valEnd, 5)
			combinedCtx := strings.ToLower(text[sub[0]:sub[1]] + " " + ctxText)
			ctxFound := FindContextKeywords(combinedCtx, d.ContextWords())

			matches = append(matches, Match{
				Type:            EntityAPIKey,
				Value:           val,
				Start:           valStart,
				End:             valEnd,
				Score:           0.95,
				ContextFound:    ctxFound,
				ValidatorPassed: true,
			})
		}
	}

	return matches
}
