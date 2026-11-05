package detector

import (
	"encoding/json"
	"os"
	"regexp"
)

// CustomRule defines a declarative PII recognition rule loaded from JSON/config.
type CustomRule struct {
	Name          string   `json:"name"`
	Pattern       string   `json:"pattern"`
	ContextWords  []string `json:"context_words"`
	BaseScore     float64  `json:"base_score"`
	EnhancedScore float64  `json:"enhanced_score"`

	compiled *regexp.Regexp
}

// CustomDetector executes dynamically loaded user-defined rules.
type CustomDetector struct {
	rules []CustomRule
}

// NewCustomDetector creates a detector with compiled custom rules.
func NewCustomDetector(rules []CustomRule) *CustomDetector {
	validRules := make([]CustomRule, 0, len(rules))
	for _, r := range rules {
		if r.Pattern == "" {
			continue
		}
		rx, err := regexp.Compile(r.Pattern)
		if err != nil {
			continue // Skip invalid regex gracefully
		}
		r.compiled = rx
		if r.BaseScore <= 0 {
			r.BaseScore = 0.50
		}
		if r.EnhancedScore <= 0 {
			r.EnhancedScore = 0.95
		}
		if r.Name == "" {
			r.Name = "CUSTOM"
		}
		validRules = append(validRules, r)
	}
	return &CustomDetector{rules: validRules}
}

// LoadCustomRulesFromFile parses custom recognizer rules from a JSON file.
func LoadCustomRulesFromFile(filePath string) ([]CustomRule, error) {
	if filePath == "" {
		return nil, nil
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var rules []CustomRule
	if err := json.Unmarshal(data, &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

// Type returns EntityCustom.
func (d *CustomDetector) Type() EntityType {
	return EntityCustom
}

// Detect scans text against all custom patterns and applies context scoring.
func (d *CustomDetector) Detect(text string) []Match {
	if len(d.rules) == 0 {
		return nil
	}

	var matches []Match
	for _, r := range d.rules {
		if r.compiled == nil {
			continue
		}

		indices := r.compiled.FindAllStringIndex(text, -1)
		for _, idx := range indices {
			candidate := text[idx[0]:idx[1]]
			ctxText := ExtractSurroundingContext(text, idx[0], idx[1], 5)
			ctxFound := FindContextKeywords(ctxText, r.ContextWords)

			score := r.BaseScore
			if len(ctxFound) > 0 {
				score = r.EnhancedScore
			}

			matches = append(matches, Match{
				Type:            EntityType(r.Name),
				Value:           candidate,
				Start:           idx[0],
				End:             idx[1],
				Score:           score,
				ContextFound:    ctxFound,
				ValidatorPassed: len(ctxFound) > 0,
			})
		}
	}

	return matches
}
