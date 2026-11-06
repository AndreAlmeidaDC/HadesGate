package policy

import (
	"strings"
	"testing"

	"github.com/hadesgate/hadesgate/internal/anonymizer/detector"
)

func TestDecisionPriority(t *testing.T) {
	engine := New(map[string]string{"CPF": "route_local", "SECRET": "block"})
	decision := engine.Decide([]string{"PERSON", "CPF", "SECRET"})
	if decision.Action != Block {
		t.Fatalf("got %q, want block", decision.Action)
	}
	if len(decision.Types) != 2 {
		t.Fatalf("unexpected policy types: %v", decision.Types)
	}
}

func TestContextualHealthDefaultsAndExplicitOverrides(t *testing.T) {
	defaults := New(nil)
	if defaults.ActionFor("MEDICATION") != Contextual || defaults.ActionFor("CID") != Contextual {
		t.Fatalf("health attributes must be contextual by default")
	}
	if defaults.ActionFor("MEDICAL_ID") != Mask || defaults.ActionFor("SUS_CARD") != Mask {
		t.Fatalf("direct health identifiers must remain masked")
	}
	if New(map[string]string{"MEDICATION": "mask"}).ActionFor("MEDICATION") != Mask {
		t.Fatal("explicit entity mask did not override contextual default")
	}
	if New(map[string]string{"CPF": "contextual"}).ActionFor("CPF") != Mask {
		t.Fatal("unsupported contextual action must fail closed to mask")
	}
	unsupported := New(map[string]string{"CPF": "contextual", "PERSONAL": "block"}).WithFamilyResolver(func(string) string { return "PERSONAL" })
	if unsupported.ActionFor("CPF") != Block {
		t.Fatal("unsupported contextual entity action must not weaken a stricter family policy")
	}
	engine := New(map[string]string{"HEALTH_PHI": "block"}).WithFamilyResolver(func(entityType string) string { return "HEALTH_PHI" })
	if engine.ActionFor("MEDICATION") != Block {
		t.Fatal("explicit family action did not override contextual default")
	}
}

func TestContextualHealthRequiresIdentityInSameSegment(t *testing.T) {
	engine := New(nil)
	medication := func(text string) detector.Match {
		start := strings.Index(text, "Amoxicilina")
		return detector.Match{Type: detector.EntityMedication, Value: "Amoxicilina", Start: start, End: start + len("Amoxicilina")}
	}
	person := func(text string) detector.Match {
		start := strings.Index(text, "João Silva")
		return detector.Match{Type: detector.EntityName, Value: "João Silva", Start: start, End: start + len("João Silva")}
	}

	standalone := "Crie uma receita de Amoxicilina 500mg."
	if filtered := engine.FilterContextualMatches(standalone, []detector.Match{medication(standalone)}); len(filtered) != 0 {
		t.Fatalf("standalone medication should pass through: %+v", filtered)
	}
	linked := "Paciente João Silva deve tomar Amoxicilina 500mg."
	if filtered := engine.FilterContextualMatches(linked, []detector.Match{person(linked), medication(linked)}); len(filtered) != 2 {
		t.Fatalf("linked medication must remain protected: %+v", filtered)
	}
	unrelated := "João Silva confirmou o cadastro. Explique Amoxicilina 500mg."
	if filtered := engine.FilterContextualMatches(unrelated, []detector.Match{person(unrelated), medication(unrelated)}); len(filtered) != 1 || filtered[0].Type != detector.EntityName {
		t.Fatalf("different sentence must not create a health link: %+v", filtered)
	}

	cid := func(text string) detector.Match {
		start := strings.Index(text, "J45.0")
		return detector.Match{Type: detector.EntityCID, Value: "J45.0", Start: start, End: start + len("J45.0")}
	}
	standaloneCID := "Explique o diagnóstico CID J45.0."
	if filtered := engine.FilterContextualMatches(standaloneCID, []detector.Match{cid(standaloneCID)}); len(filtered) != 0 {
		t.Fatalf("standalone CID should pass through: %+v", filtered)
	}
	linkedCID := "O diagnóstico de João Silva é CID J45.0."
	if filtered := engine.FilterContextualMatches(linkedCID, []detector.Match{person(linkedCID), cid(linkedCID)}); len(filtered) != 2 {
		t.Fatalf("linked CID must remain protected: %+v", filtered)
	}
}

func TestDecision_FamilyResolverFallback(t *testing.T) {
	// Configure policy action on the macro-family HEALTH_PHI: route_local
	engine := New(map[string]string{"HEALTH_PHI": "route_local"}).WithFamilyResolver(func(entityType string) string {
		if entityType == "SUS_CARD" || entityType == "MEDICAL_ID" {
			return "HEALTH_PHI"
		}
		return "PERSONAL"
	})

	decision := engine.Decide([]string{"SUS_CARD"})
	if decision.Action != RouteLocal {
		t.Fatalf("expected RouteLocal via family fallback, got %s", decision.Action)
	}
	if len(decision.Types) != 1 || decision.Types[0] != "SUS_CARD" {
		t.Fatalf("expected SUS_CARD in decision types, got %v", decision.Types)
	}
}

func TestDisabledActionUsesEntityAndFamilyResolution(t *testing.T) {
	engine := New(map[string]string{
		"CPF":      "disabled",
		"PERSONAL": "disabled",
		"EMAIL":    "mask",
	}).WithFamilyResolver(func(entityType string) string {
		if entityType == "PERSON" || entityType == "PHONE" {
			return "PERSONAL"
		}
		return ""
	})

	if !engine.IsDisabled("cpf") {
		t.Fatal("expected entity-level CPF disable to be effective")
	}
	if !engine.IsDisabled("PERSON") {
		t.Fatal("expected PERSONAL family disable to apply to PERSON")
	}
	if engine.IsDisabled("EMAIL") {
		t.Fatal("explicit EMAIL mask must override a family/default disable")
	}
	if !engine.HasDisabledActions() {
		t.Fatal("expected policy to report configured disabled actions")
	}

	decision := engine.Decide([]string{"CPF", "PERSON", "EMAIL"})
	if decision.Action != Mask || len(decision.Types) != 0 {
		t.Fatalf("disabled entities must not participate in policy decisions: %+v", decision)
	}
}
