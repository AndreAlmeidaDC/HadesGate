package detector

import "testing"

func TestPhoneDetector_RejectsNumbersEmbeddedInIdentifiersAndRepeatedSentinels(t *testing.T) {
	detector := NewPhoneDetector()
	for _, text := range []string{
		"trace_id 550e8400-e29b-41d4-a716-446655440000 acompanhou a requisição.",
		"CNH 99999999999 é um valor fictício reservado para testes.",
	} {
		if matches := detector.Detect(text); len(matches) != 0 {
			t.Fatalf("non-phone text %q produced matches: %+v", text, matches)
		}
	}
}

func TestNameDetector_RejectsCompoundLocationsWithoutPersonContext(t *testing.T) {
	detector := NewNameDetector()
	for _, text := range []string{
		"O congresso acontecerá em João Pessoa durante o mês de maio.",
		"A rodovia entre Vitória da Conquista e Salvador está liberada.",
	} {
		if matches := detector.Detect(text); len(matches) != 0 {
			t.Fatalf("location text %q produced person matches: %+v", text, matches)
		}
	}

	text := "Cliente João Pessoa, CPF 529.982.247-25, confirmou o cadastro."
	matches := detector.Detect(text)
	if len(matches) != 1 || matches[0].Value != "João Pessoa" {
		t.Fatalf("explicit person context must preserve a location-shaped name: %+v", matches)
	}
}

func TestMedicationDetector_RejectsOrdinaryTestingTerm(t *testing.T) {
	detector := NewMedicationDetector()
	text := "CNH 99999999999 é um valor fictício reservado para testes."
	if matches := detector.Detect(text); len(matches) != 0 {
		t.Fatalf("ordinary testing text produced medication matches: %+v", matches)
	}
}
