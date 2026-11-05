package detector

import (
	"testing"
)

func TestNameCatalog_LoadedAndLookups(t *testing.T) {
	ac := GetNameAhoCorasick()
	if ac == nil || len(ac.entries) == 0 {
		t.Fatalf("expected non-empty name catalog, got %v", ac)
	}

	tests := []struct {
		name      string
		wantGiven bool
		wantSurn  bool
		minFreq   uint32
	}{
		{"carlos", true, false, 1000000},
		{"silva", false, true, 20000000},
		{"maria", true, false, 10000000},
		{"santos", false, true, 10000000},
		{"rodolfo", true, false, 5000},
		{"bandeira", false, true, 20000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			entry, ok := ac.Lookup(tc.name)
			if !ok {
				t.Fatalf("Lookup(%q) not found", tc.name)
			}
			if tc.wantGiven && (entry.Flags&NameTypeGiven == 0) {
				t.Errorf("%q: want Given flag, got %d", tc.name, entry.Flags)
			}
			if tc.wantSurn && (entry.Flags&NameTypeSurname == 0) {
				t.Errorf("%q: want Surname flag, got %d", tc.name, entry.Flags)
			}
			maxFreq := entry.FreqGiven
			if entry.FreqSurname > maxFreq {
				maxFreq = entry.FreqSurname
			}
			if maxFreq < tc.minFreq {
				t.Errorf("%q: freq %d < %d", tc.name, maxFreq, tc.minFreq)
			}
		})
	}
}

func TestNameAhoCorasick_SearchCandidateTokens(t *testing.T) {
	ac := GetNameAhoCorasick()
	text := "Olá Carlos Eduardo da Silva, confirmamos o contato de Mariana Costa."

	candidates := ac.SearchCandidateTokens(text)
	if len(candidates) == 0 {
		t.Fatalf("expected candidate tokens in text %q", text)
	}

	foundWords := make(map[string]bool)
	for _, c := range candidates {
		foundWords[c.Text] = true
	}

	for _, w := range []string{"Carlos", "Eduardo", "Silva", "Mariana", "Costa"} {
		if !foundWords[w] {
			t.Errorf("expected candidate %q to be found by AC search", w)
		}
	}
}

