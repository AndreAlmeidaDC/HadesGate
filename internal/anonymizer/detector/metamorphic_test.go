package detector

import (
	"fmt"
	"testing"
)

func TestDetectors_PreserveMatchesAcrossContextAndBoundaryTransformations(t *testing.T) {
	tests := []struct {
		name     string
		detector Detector
		value    string
	}{
		{name: "cpf", detector: NewCPFDetector(), value: "529.982.247-25"},
		{name: "cnpj", detector: NewCNPJDetector(), value: "04.252.011/0001-10"},
		{name: "email", detector: NewEmailDetector(), value: "maria.silva@example.com"},
		{name: "credit-card", detector: NewCreditCardDetector(), value: "4111 1111 1111 1111"},
		{name: "ip", detector: NewIPDetector(), value: "192.168.10.45"},
		{name: "iban", detector: NewIBANDetector(), value: "DE89 3704 0044 0532 0130 00"},
		{name: "api-key", detector: NewAPIKeyDetector(), value: "ghp_1234567890abcdefghijklmnopqrstuvwxyz"},
	}

	transformations := []struct {
		name  string
		apply func(string) string
	}{
		{name: "standalone", apply: func(value string) string { return value }},
		{name: "unicode-prefix", apply: func(value string) string { return "Dados válidos: «" + value + "»." }},
		{name: "newlines", apply: func(value string) string { return "início\n" + value + "\nfim" }},
		{name: "json-string", apply: func(value string) string { return fmt.Sprintf(`{"value":"%s"}`, value) }},
		{name: "markdown", apply: func(value string) string { return "**valor**: `" + value + "`" }},
	}

	for _, test := range tests {
		for _, transformation := range transformations {
			t.Run(test.name+"/"+transformation.name, func(t *testing.T) {
				text := transformation.apply(test.value)
				matches := test.detector.Detect(text)
				for _, match := range matches {
					if match.Value == test.value {
						if match.Start < 0 || match.End > len(text) || text[match.Start:match.End] != test.value {
							t.Fatalf("invalid byte span for %+v in %q", match, text)
						}
						return
					}
				}
				t.Fatalf("%s was not detected after %s transformation: %+v", test.value, transformation.name, matches)
			})
		}
	}
}
