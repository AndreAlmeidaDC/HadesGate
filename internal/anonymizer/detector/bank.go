package detector

import (
	"regexp"
	"strings"
)

// contextualBankRegex matches explicit Bank Account and Agency references.
// e.g.: "agência 1234", "conta corrente 12345-6", "ag: 0001 cc: 123456-7", "conta: 98765-4"
var contextualBankRegex = regexp.MustCompile(`(?i)\b(?:ag[êe]ncia|ag\.?|conta(?:\s+corrente|\s+poupan[çc]a)?|c\/c|cc)\s*(?::|=|\s)?\s*([0-9]{3,12}-?[0-9a-zA-Z])\b`)

// BankAccountDetector identifies Brazilian bank account and agency numbers.
type BankAccountDetector struct{}

// NewBankAccountDetector creates a new BankAccountDetector.
func NewBankAccountDetector() *BankAccountDetector {
	return &BankAccountDetector{}
}

// Type returns EntityBankAccount.
func (d *BankAccountDetector) Type() EntityType {
	return EntityBankAccount
}

// ContextWords returns keywords and major Brazilian bank names that provide banking context.
func (d *BankAccountDetector) ContextWords() []string {
	return []string{
		"agencia", "agência", "conta", "conta corrente", "poupanca", "poupança",
		"c/c", "banco", "dados bancarios", "dados bancários", "deposito", "depósito", "ted", "doc",
		"itau", "itaú", "bradesco", "santander", "caixa", "caixa economica", "caixa econômica",
		"banco do brasil", "bb", "nubank", "inter", "banco inter", "c6", "c6 bank",
		"pagbank", "sicredi", "sicoob", "btg", "btg pactual", "safra", "banrisul",
	}
}

// Detect scans text for bank account and agency numbers.
func (d *BankAccountDetector) Detect(text string) []Match {
	submatches := contextualBankRegex.FindAllStringSubmatchIndex(text, -1)
	if len(submatches) == 0 {
		return nil
	}

	matches := make([]Match, 0, len(submatches))
	for _, sub := range submatches {
		if len(sub) >= 4 && sub[2] != -1 && sub[3] != -1 {
			val := text[sub[2]:sub[3]]
			ctxText := ExtractSurroundingContext(text, sub[2], sub[3], 5)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())

			score := 0.40
			if len(ctxFound) > 0 {
				score = 0.95
			}

			// If it contains a hyphen (e.g. 12345-6), confidence is inherently stronger
			if strings.Contains(val, "-") && len(ctxFound) > 0 {
				score = 0.98
			}

			matches = append(matches, Match{
				Type:            EntityBankAccount,
				Value:           val,
				Start:           sub[2],
				End:             sub[3],
				Score:           score,
				ContextFound:    ctxFound,
				ValidatorPassed: len(ctxFound) > 0,
			})
		}
	}

	return matches
}
