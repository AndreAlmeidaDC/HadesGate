package detector

import (
	"testing"
)

func TestAhoCorasick_Search(t *testing.T) {
	patterns := []string{
		"dipirona",
		"amoxicilina",
		"novalgina",
		"clonazepam",
		"paracetamol",
		"sol", // ambiguous short word
	}

	ac := NewAhoCorasick(patterns)

	text := "O médico receitou Novalgina e Dipirona 500mg para a dor. O sol está forte."
	matches := ac.Search(text)

	// Should match Novalgina, Dipirona, sol (word boundary respected)
	found := make(map[string]bool)
	for _, m := range matches {
		found[m.Pattern] = true
		extracted := text[m.Start:m.End]
		if normalizeMedicationText(extracted) != m.Pattern {
			t.Errorf("extracted text %q does not match pattern %q", extracted, m.Pattern)
		}
	}

	if !found["novalgina"] {
		t.Errorf("expected 'novalgina' match")
	}
	if !found["dipirona"] {
		t.Errorf("expected 'dipirona' match")
	}
	if !found["sol"] {
		t.Errorf("expected 'sol' match")
	}

	// Boundary test: "absoluto" should NOT match "sol"
	negText := "Isso é um valor absoluto sem parar."
	negMatches := ac.Search(negText)
	if len(negMatches) > 0 {
		t.Fatalf("expected 0 matches for 'absoluto'/'parar', got %+v", negMatches)
	}
}

func TestAhoCorasick_AccentsAndCase(t *testing.T) {
	patterns := []string{
		"ácido acetilsalicílico",
		"cloridrato de fluoxetina",
	}

	ac := NewAhoCorasick(patterns)

	text := "Tomar ACIDO ACETILSALICILICO após o almoço e Cloridrato de Fluoxetina pela manhã."
	matches := ac.Search(text)

	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d: %+v", len(matches), matches)
	}

	if text[matches[0].Start:matches[0].End] != "ACIDO ACETILSALICILICO" {
		t.Errorf("match 0 = %q, expected 'ACIDO ACETILSALICILICO'", text[matches[0].Start:matches[0].End])
	}
	if text[matches[1].Start:matches[1].End] != "Cloridrato de Fluoxetina" {
		t.Errorf("match 1 = %q, expected 'Cloridrato de Fluoxetina'", text[matches[1].Start:matches[1].End])
	}
}

