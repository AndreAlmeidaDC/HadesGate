package detector

import (
	"regexp"
	"strings"
	"unicode"
)

// cardRegex matches potential credit card number sequences (13 to 19 digits).
var cardRegex = regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`)

// CreditCardDetector detects valid credit and debit cards using the Luhn algorithm.
type CreditCardDetector struct{}

// NewCreditCardDetector creates a new instance of CreditCardDetector.
func NewCreditCardDetector() *CreditCardDetector {
	return &CreditCardDetector{}
}

// Type returns EntityCreditCard.
func (d *CreditCardDetector) Type() EntityType {
	return EntityCreditCard
}

// ContextWords returns keywords that provide contextual evidence for credit cards.
func (d *CreditCardDetector) ContextWords() []string {
	return []string{"cartao", "cartão", "final", "credito", "crédito", "debito", "débito", "card", "cvv", "validade", "bandeira", "visa", "mastercard", "elo", "amex", "hipercard"}
}

// Detect scans text for valid credit card numbers (13-19 digits with Luhn algorithm).
// Note: Per PCI-DSS standards, the last 4 digits of a card are considered non-sensitive reference data.
func (d *CreditCardDetector) Detect(text string) []Match {
	indices := cardRegex.FindAllStringIndex(text, -1)
	if len(indices) == 0 {
		return nil
	}

	matches := make([]Match, 0, len(indices))
	for _, idx := range indices {
		candidate := text[idx[0]:idx[1]]
		if ValidateCreditCard(candidate) {
			ctxText := ExtractSurroundingContext(text, idx[0], idx[1], 5)
			ctxFound := FindContextKeywords(ctxText, d.ContextWords())

			// If surrounding context clearly indicates a telecommunications device / IMEI, yield to device detector
			deviceContext := FindContextKeywords(ctxText, []string{"imei", "smartphone", "celular", "modem"})
			if len(deviceContext) > 0 && len(ctxFound) == 0 {
				continue
			}

			score := 0.95
			if len(ctxFound) > 0 || strings.ContainsAny(candidate, " -") {
				score = 1.0
			}

			matches = append(matches, Match{
				Type:            EntityCreditCard,
				Value:           candidate,
				Start:           idx[0],
				End:             idx[1],
				Score:           score,
				ContextFound:    ctxFound,
				ValidatorPassed: true,
			})
		}
	}

	return matches
}

// ValidateCreditCard checks if a number passes the Luhn algorithm and has a valid length.
func ValidateCreditCard(raw string) bool {
	digits := make([]int, 0, 19)
	for _, r := range raw {
		if unicode.IsDigit(r) {
			digits = append(digits, int(r-'0'))
		}
	}

	n := len(digits)
	if n < 13 || n > 19 {
		return false
	}

	// Repeated sentinel values such as 0000... pass Luhn mathematically but
	// are not issued payment-card numbers and create noisy false positives.
	allEqual := true
	for _, digit := range digits[1:] {
		if digit != digits[0] {
			allEqual = false
			break
		}
	}
	if allEqual {
		return false
	}

	// Luhn check
	sum := 0
	alternate := false
	for i := n - 1; i >= 0; i-- {
		d := digits[i]
		if alternate {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		alternate = !alternate
	}

	return sum%10 == 0
}
