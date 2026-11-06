package anonymizer

import (
	"strings"
	"testing"
	"time"

	"github.com/hadesgate/hadesgate/internal/anonymizer/detector"
	"github.com/hadesgate/hadesgate/internal/anonymizer/vault"
	"github.com/hadesgate/hadesgate/internal/stream"
)

func TestAnonymizer_FullEndToEnd(t *testing.T) {
	v := vault.NewMemoryVault(5 * time.Minute)
	defer v.Close()

	cfg := DefaultConfig()
	engine := NewEngine(cfg, v)

	sessionID := "sess-e2e-1"
	rawPrompt := "Envie um e-mail para João da Silva, CPF 529.982.247-25, cobrando R$ 500. Contato: joao.silva@empresa.com.br ou fone (11) 98765-4321."

	result := engine.Anonymize(sessionID, rawPrompt)

	// Verify that raw sensitive data is gone from sanitized text
	if result.SanitizedText == rawPrompt {
		t.Fatalf("Sanitized text is identical to original prompt! No PII was masked.")
	}

	// Verify all entities were detected
	foundTypes := make(map[string]bool)
	for _, ent := range result.Entities {
		foundTypes[string(ent.Type)] = true
	}

	expectedTypes := []string{"PERSON", "CPF", "EMAIL", "PHONE"}
	for _, exp := range expectedTypes {
		if !foundTypes[exp] {
			t.Errorf("Expected entity type %s was not detected in entities: %+v", exp, result.Entities)
		}
	}

	// Verify Stream Detokenization can reconstitute the exact original string
	detok := stream.NewDetokenizer(sessionID, v)
	reconstituted := detok.Feed(result.SanitizedText) + detok.Flush()

	if reconstituted != rawPrompt {
		t.Fatalf("Reconstituted text does not match original!\nGot:      %q\nExpected: %q", reconstituted, rawPrompt)
	}
}

func TestAnonymizer_DisabledTypePassesThroughWithoutVaultOrMetrics(t *testing.T) {
	v := vault.NewMemoryVault(5 * time.Minute)
	defer v.Close()
	engine := NewEngine(DefaultConfig(), v)

	const sessionID = "sess-disabled-cpf"
	const rawCPF = "529.982.247-25"
	const rawEmail = "joao.silva@empresa.com.br"
	result := engine.AnonymizeWithOptions(sessionID, "CPF "+rawCPF+" email "+rawEmail, AnonymizeOptions{
		IsTypeDisabled: func(entityType detector.EntityType) bool {
			return entityType == detector.EntityType("CPF")
		},
	})

	if !strings.Contains(result.SanitizedText, rawCPF) {
		t.Fatalf("disabled CPF should pass through unchanged: %q", result.SanitizedText)
	}
	if strings.Contains(result.SanitizedText, rawEmail) {
		t.Fatalf("enabled email should still be tokenized: %q", result.SanitizedText)
	}
	for _, entity := range result.Entities {
		if entity.Type == detector.EntityType("CPF") {
			t.Fatalf("disabled CPF must not be reported as detected: %+v", result.Entities)
		}
	}
	if len(v.GetAllSessionMappings(sessionID)) != 1 {
		t.Fatalf("vault should contain only the enabled email token: %+v", v.GetAllSessionMappings(sessionID))
	}
}

func TestAnonymizer_NewBrazilianDetectors(t *testing.T) {
	v := vault.NewMemoryVault(5 * time.Minute)
	defer v.Close()

	cfg := DefaultConfig()
	engine := NewEngine(cfg, v)

	sessionID := "sess-br-1"
	rawPrompt := "Pagamento via chave pix: 123e4567-e89b-12d3-a456-426614174000. Documento RG 12.345.678-9 SSP e CNH 12345678900. Banco Itaú agência 1234 conta 56789-0. Passaporte CS123456 da Polícia Federal."

	result := engine.Anonymize(sessionID, rawPrompt)

	foundTypes := make(map[string]bool)
	for _, ent := range result.Entities {
		foundTypes[string(ent.Type)] = true
	}

	expected := []string{"PIX", "RG", "CNH", "BANK_ACCOUNT", "PASSPORT"}
	for _, exp := range expected {
		if !foundTypes[exp] {
			t.Errorf("Expected entity %s was not detected. Entities found: %+v", exp, result.Entities)
		}
	}

	// Verify detokenization restores original
	detok := stream.NewDetokenizer(sessionID, v)
	reconstituted := detok.Feed(result.SanitizedText) + detok.Flush()
	if reconstituted != rawPrompt {
		t.Fatalf("Reconstituted text does not match original!\nGot:      %q\nExpected: %q", reconstituted, rawPrompt)
	}
}

func TestAnonymizer_AllowlistAndDenylist(t *testing.T) {
	v := vault.NewMemoryVault(5 * time.Minute)
	defer v.Close()

	cfg := DefaultConfig()
	cfg.Allowlist = []string{"suporte@empresa.com.br"}
	cfg.Denylist = []string{"Projeto Manhatan", "super_secret_token"}
	engine := NewEngine(cfg, v)

	sessionID := "sess-allow-deny"
	prompt := "Contato permitido: suporte@empresa.com.br mas vazou joao@empresa.com.br e o Projeto Manhatan."

	result := engine.Anonymize(sessionID, prompt)

	// suporte@empresa.com.br must NOT be masked
	if !strings.Contains(result.SanitizedText, "suporte@empresa.com.br") {
		t.Errorf("suporte@empresa.com.br was masked but should have been allowlisted! Result: %s", result.SanitizedText)
	}

	// joao@empresa.com.br MUST be masked
	if strings.Contains(result.SanitizedText, "joao@empresa.com.br") {
		t.Errorf("joao@empresa.com.br was NOT masked! Result: %s", result.SanitizedText)
	}

	// Projeto Manhatan MUST be masked as CONFIDENTIAL
	if strings.Contains(result.SanitizedText, "Projeto Manhatan") {
		t.Errorf("Projeto Manhatan was NOT masked as Denylist! Result: %s", result.SanitizedText)
	}
	if !strings.Contains(result.SanitizedText, "_CONFIDENTIAL_1]") {
		t.Errorf("Expected a namespaced CONFIDENTIAL token in text, got: %s", result.SanitizedText)
	}
}

func TestAnonymizer_DoesNotReuseTokenForDifferentValuesInSession(t *testing.T) {
	v := vault.NewMemoryVault(5 * time.Minute)
	defer v.Close()

	engine := NewEngine(DefaultConfig(), v)
	const sessionID = "sess-multiple-messages"

	first := engine.Anonymize(sessionID, "CPF 529.982.247-25")
	second := engine.Anonymize(sessionID, "CPF 111.444.777-35")

	if len(first.Entities) != 1 || len(second.Entities) != 1 {
		t.Fatalf("expected one entity in each message, got first=%+v second=%+v", first.Entities, second.Entities)
	}
	if first.Entities[0].Token == second.Entities[0].Token {
		t.Fatalf("different values reused token %q", first.Entities[0].Token)
	}

	detok := stream.NewDetokenizer(sessionID, v)
	restored := detok.Feed(first.SanitizedText+" / "+second.SanitizedText) + detok.Flush()
	if restored != "CPF 529.982.247-25 / CPF 111.444.777-35" {
		t.Fatalf("restored text mismatch: %q", restored)
	}
}

func TestAnonymizer_MinConfidenceFilter(t *testing.T) {
	v := vault.NewMemoryVault(5 * time.Minute)
	defer v.Close()

	cfg := DefaultConfig()
	cfg.MinConfidence = 0.80
	engine := NewEngine(cfg, v)

	sessionID := "sess-conf-filter"
	// 12345678 is unformatted CEP with score 0.50 (without context)
	// It should be skipped by MinConfidence = 0.80
	prompt := "Código do lote 12345678 sem dados."
	result := engine.Anonymize(sessionID, prompt)

	if len(result.Entities) > 0 {
		t.Errorf("Expected 0 entities due to confidence score < 0.80, got: %+v", result.Entities)
	}
	if result.SanitizedText != prompt {
		t.Errorf("Prompt was modified when confidence was too low. Got: %s", result.SanitizedText)
	}
}

func TestAnonymizer_UserSampleText(t *testing.T) {
	v := vault.NewMemoryVault(5 * time.Minute)
	defer v.Close()

	cfg := DefaultConfig()
	engine := NewEngine(cfg, v)

	sessionID := "sess-user-sample"
	rawPrompt := "Olá Rodolfo! Este é seu CPF: 098.976.844-92, e tbm temos o numero do seu RG 3109380. Você pode tbm pegar o fianl do seu cartão que é 8558."

	result := engine.Anonymize(sessionID, rawPrompt)

	if len(result.Entities) != 3 {
		t.Fatalf("Expected exactly 3 entities (PERSON, CPF, RG), got %d: %+v", len(result.Entities), result.Entities)
	}
	if strings.Contains(result.SanitizedText, "098.976.844-92") {
		t.Errorf("CPF was not masked")
	}
	if strings.Contains(result.SanitizedText, "3109380") {
		t.Errorf("RG was not masked")
	}
	if strings.Contains(result.SanitizedText, "Rodolfo") {
		t.Errorf("Name was not masked")
	}
	// Per PCI-DSS standards, the last 4 digits of a card are non-sensitive and should remain visible
	if !strings.Contains(result.SanitizedText, "8558") {
		t.Errorf("Card final 4 digits (8558) should NOT be masked per PCI-DSS, but was missing: %s", result.SanitizedText)
	}

	// Verify detokenization restores original exactly
	detok := stream.NewDetokenizer(sessionID, v)
	reconstituted := detok.Feed(result.SanitizedText) + detok.Flush()
	if reconstituted != rawPrompt {
		t.Fatalf("Reconstituted text mismatch!\nGot:      %q\nExpected: %q", reconstituted, rawPrompt)
	}
}

func TestAnonymizer_IBANAndIPAndIntlPhone(t *testing.T) {
	v := vault.NewMemoryVault(5 * time.Minute)
	defer v.Close()

	engine := NewEngine(DefaultConfig(), v)
	const sessionID = "sess-intl-1"

	rawPrompt := "Transferir para IBAN DE89 3704 0044 0532 0130 00 a partir do IP 192.168.15.20. Contato internacional: +1 (415) 555-2671."
	res := engine.Anonymize(sessionID, rawPrompt)

	foundTypes := make(map[string]bool)
	for _, ent := range res.Entities {
		foundTypes[string(ent.Type)] = true
	}
	for _, exp := range []string{"IBAN", "IP_ADDRESS", "PHONE"} {
		if !foundTypes[exp] {
			t.Errorf("expected entity type %s, got entities: %+v", exp, res.Entities)
		}
	}

	detok := stream.NewDetokenizer(sessionID, v)
	restored := detok.Feed(res.SanitizedText) + detok.Flush()
	if restored != rawPrompt {
		t.Fatalf("restored text mismatch: got %q, want %q", restored, rawPrompt)
	}
}

func TestAnonymizer_APIKeysAndSecrets(t *testing.T) {
	v := vault.NewMemoryVault(5 * time.Minute)
	defer v.Close()

	cfg := DefaultConfig()
	engine := NewEngine(cfg, v)

	sessionID := "sess-secrets-e2e"
	rawPrompt := "OpenAI key: sk-proj-1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890 e banco postgres://user:secretPwd123@db.prod.internal:5432/main_db"

	result := engine.Anonymize(sessionID, rawPrompt)

	if strings.Contains(result.SanitizedText, "sk-proj-") {
		t.Errorf("OpenAI key was not masked: %s", result.SanitizedText)
	}
	if strings.Contains(result.SanitizedText, "secretPwd123") {
		t.Errorf("Database secret was not masked: %s", result.SanitizedText)
	}

	foundAPIKey := false
	foundSecret := false
	for _, ent := range result.Entities {
		if ent.Type == "API_KEY" {
			foundAPIKey = true
		}
		if ent.Type == "SECRET" {
			foundSecret = true
		}
	}
	if !foundAPIKey {
		t.Errorf("Expected API_KEY entity in: %+v", result.Entities)
	}
	if !foundSecret {
		t.Errorf("Expected SECRET entity in: %+v", result.Entities)
	}

	// Verify detokenization restores original exactly
	detok := stream.NewDetokenizer(sessionID, v)
	reconstituted := detok.Feed(result.SanitizedText) + detok.Flush()
	if reconstituted != rawPrompt {
		t.Fatalf("Reconstituted text mismatch!\nGot:      %q\nExpected: %q", reconstituted, rawPrompt)
	}
}

func TestAnonymizer_CanonicalEntityReuse(t *testing.T) {
	v := vault.NewMemoryVault(5 * time.Minute)
	defer v.Close()

	engine := NewEngine(DefaultConfig(), v)
	const sessionID = "sess-canonical-reuse"

	prompt := "O CPF de Maria Silva é 529.982.247-25. Confirme o cadastro de MARIA SILVA no CPF 52998224725."
	res := engine.Anonymize(sessionID, prompt)

	var cpfTokens []string
	var personTokens []string
	for _, ent := range res.Entities {
		if ent.Type == "CPF" {
			cpfTokens = append(cpfTokens, ent.Token)
		}
		if ent.Type == "PERSON" {
			personTokens = append(personTokens, ent.Token)
		}
	}
	if len(cpfTokens) != 2 || cpfTokens[0] != cpfTokens[1] {
		t.Fatalf("expected formatted and unformatted CPF to reuse the same token, got: %+v", cpfTokens)
	}
	if len(personTokens) != 2 || personTokens[0] != personTokens[1] {
		t.Fatalf("expected 'Maria Silva' and 'MARIA SILVA' to reuse the same token, got: %+v", personTokens)
	}
}

func TestAnonymizer_BillingPromptWithCPFApposition(t *testing.T) {
	v := vault.NewMemoryVault(5 * time.Minute)
	defer v.Close()

	engine := NewEngine(DefaultConfig(), v)
	sessionID := "sess-billing-cpf-apposition"

	rawPrompt := "Vamso criar uma cobrança para Rodolfo Bandeira de CPF: 09897684492."
	res := engine.Anonymize(sessionID, rawPrompt)

	if len(res.Entities) != 2 {
		t.Fatalf("expected exactly 2 entities (PERSON and CPF), got %d: %+v", len(res.Entities), res.Entities)
	}

	var personEntity, cpfEntity *DetectedEntity
	for i := range res.Entities {
		ent := &res.Entities[i]
		if ent.Type == detector.EntityName {
			personEntity = ent
		}
		if ent.Type == detector.EntityCPF {
			cpfEntity = ent
		}
	}

	if personEntity == nil || personEntity.Original != "Rodolfo Bandeira" {
		t.Fatalf("expected PERSON to be 'Rodolfo Bandeira', got: %+v", personEntity)
	}
	if cpfEntity == nil || cpfEntity.Original != "09897684492" {
		t.Fatalf("expected CPF to be '09897684492', got: %+v", cpfEntity)
	}

	// Ensure "de CPF:" was NOT swallowed into the person token and remains visible in sanitized text
	if !strings.Contains(res.SanitizedText, "de CPF:") {
		t.Errorf("expected 'de CPF:' to remain untouched in sanitized text, got %q", res.SanitizedText)
	}

	// Verify detokenization restores original exactly
	detok := stream.NewDetokenizer(sessionID, v)
	restored := detok.Feed(res.SanitizedText) + detok.Flush()
	if restored != rawPrompt {
		t.Fatalf("restored text mismatch: got %q, want %q", restored, rawPrompt)
	}
}

func TestEngine_ScanEgress(t *testing.T) {
	v := vault.NewMemoryVault(5 * time.Minute)
	defer v.Close()

	engine := NewEngine(DefaultConfig(), v)

	// Case 1: Text contains a user-known restored entity -> should NOT be masked
	known := map[string]bool{
		"529.982.247-25": true,
	}
	textWithKnown := "O cliente do CPF 529.982.247-25 foi consultado."
	res1 := engine.ScanEgress(known, textWithKnown)
	if len(res1.LeakedEntities) > 0 {
		t.Fatalf("expected 0 leaked entities for known user data, got %d", len(res1.LeakedEntities))
	}
	if res1.SanitizedText != textWithKnown {
		t.Fatalf("expected text to remain unchanged for known user data, got %q", res1.SanitizedText)
	}

	// Case 2: LLM generated a new unmapped credential/PII -> MUST be masked as [HG_EGRESS_...]
	textWithNewLeak := "Aqui está uma chave de API: sk-proj-1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ e cartão 4111 1111 1111 1111."
	res2 := engine.ScanEgress(known, textWithNewLeak)
	if len(res2.LeakedEntities) != 2 {
		t.Fatalf("expected 2 leaked entities, got %d: %+v", len(res2.LeakedEntities), res2.LeakedEntities)
	}
	if strings.Contains(res2.SanitizedText, "sk-proj-") {
		t.Errorf("expected API key to be masked in egress, got %q", res2.SanitizedText)
	}
	if !strings.Contains(res2.SanitizedText, "[HG_EGRESS_") {
		t.Errorf("expected [HG_EGRESS_...] placeholder in sanitized text, got %q", res2.SanitizedText)
	}
}

func TestAnonymizer_Phase14(t *testing.T) {
	v := vault.NewMemoryVault(5 * time.Minute)
	defer v.Close()

	engine := NewEngine(DefaultConfig(), v)

	// Test Passport disambiguation over NINO
	resPass := engine.Anonymize("bench-pass", "Apresentou o passaporte CS123456 no embarque.")
	if len(resPass.Entities) != 1 || resPass.Entities[0].Type != detector.EntityPassport {
		t.Fatalf("expected 1 PASSPORT entity, got: %+v", resPass.Entities)
	}

	// Test HIPAA Medical identifiers
	resMed := engine.Anonymize("bench-med", "Receita assinada pelo médico responsável com DEA AB1234563 e NPI 1679580450 no Prontuário REC-987452.")
	if len(resMed.Entities) != 3 {
		t.Fatalf("expected 3 medical entities, got %d: %+v", len(resMed.Entities), resMed.Entities)
	}
	for _, ent := range resMed.Entities {
		if ent.Type != detector.EntityMedicalID {
			t.Errorf("expected MEDICAL_ID type, got %s", ent.Type)
		}
	}

	// Test International Detectors
	resIntl := engine.Anonymize("bench-intl", "SSN: 219-45-6780, EIN: 12-3456789, NINO: RN 12 34 56 A, NHS: 943 476 5919, DNI: 12345678Z, NIR: 1 85 12 75 108 042 79, CF: RSSMRA85M01H501Q.")
	typesFound := make(map[detector.EntityType]bool)
	for _, ent := range resIntl.Entities {
		typesFound[ent.Type] = true
	}

	expectedTypes := []detector.EntityType{
		detector.EntitySSN, detector.EntityEIN, detector.EntityNINO,
		detector.EntityNHS, detector.EntityDNI, detector.EntityNIR,
		detector.EntityCodiceFiscale,
	}
	for _, exp := range expectedTypes {
		if !typesFound[exp] {
			t.Errorf("expected entity %s to be detected, found: %+v", exp, typesFound)
		}
	}
}

func TestAnonymizer_DynamicThresholds(t *testing.T) {
	v := vault.NewMemoryVault(5 * time.Minute)
	defer v.Close()

	cfg := DefaultConfig()
	engine := NewEngine(cfg, v)

	text := "Olá Carlos Silva, portador do CPF 529.982.247-25."

	// Default: PERSON and CPF are detected
	res1 := engine.Anonymize("s1", text)
	hasPerson := false
	for _, e := range res1.Entities {
		if e.Type == detector.EntityName {
			hasPerson = true
		}
	}
	if !hasPerson {
		t.Fatalf("expected PERSON to be detected by default")
	}

	// Now set PERSON threshold to 0.99 (higher than name score ~0.90)
	engine.SetThresholds(nil, map[string]float64{
		"PERSON": 0.99,
	})

	res2 := engine.Anonymize("s2", text)
	for _, e := range res2.Entities {
		if e.Type == detector.EntityName {
			t.Fatalf("PERSON should have been filtered out by 0.99 threshold, but was detected")
		}
	}

	// Now reset PERSON to 0.70 via family PERSONAL
	engine.SetThresholds(map[string]float64{
		"PERSONAL": 0.70,
	}, nil)

	res3 := engine.Anonymize("s3", text)
	hasPersonAgain := false
	for _, e := range res3.Entities {
		if e.Type == detector.EntityName {
			hasPersonAgain = true
		}
	}
	if !hasPersonAgain {
		t.Fatalf("expected PERSON to be detected again after resetting PERSONAL family threshold")
	}
}

func TestAnonymizer_RequestScopedMatchFilterRunsBeforeVaultTokenization(t *testing.T) {
	v := vault.NewMemoryVault(5 * time.Minute)
	defer v.Close()
	engine := NewEngine(DefaultConfig(), v)
	text := "Receita de Amoxicilina 500mg."
	result := engine.AnonymizeWithOptions("contextual-filter", text, AnonymizeOptions{
		FilterMatches: func(_ string, matches []detector.Match) []detector.Match {
			filtered := matches[:0]
			for _, match := range matches {
				if match.Type != detector.EntityMedication {
					filtered = append(filtered, match)
				}
			}
			return filtered
		},
	})
	if result.SanitizedText != text || len(result.Entities) != 0 || len(result.TokenMap) != 0 {
		t.Fatalf("filtered match reached tokenization or vault: %+v", result)
	}
}
