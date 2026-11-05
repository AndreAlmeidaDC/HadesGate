package policy

import "testing"

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
