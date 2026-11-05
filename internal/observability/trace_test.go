package observability

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hadesgate/hadesgate/internal/anonymizer"
	"github.com/hadesgate/hadesgate/internal/anonymizer/detector"
)

func TestSQLiteTraceLogger_PersistenceAndSummary(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "hadesgate_test.db")

	logger, err := NewSQLiteTraceLogger(dbPath, 3)
	if err != nil {
		t.Fatalf("NewSQLiteTraceLogger failed: %v", err)
	}

	now := time.Now()
	logger.Record(Trace{
		ID:              "req_1",
		Timestamp:       now.Add(-2 * time.Hour), // older than 1h
		Provider:        "openai",
		Model:           "gpt-4o",
		Language:        "PT",
		StatusCode:      200,
		DurationMs:      12.5,
		AnonDurationUs:  120,
		EntitiesCount:   2,
		EntityTypes:     []string{"PERSON", "CPF"},
		PIITypes:        []string{"PERSON", "CPF"},
		SanitizedPrompt: "Cobrar fatura de [PG_abc123_PERSON_1] no CPF [PG_abc123_CPF_1].",
	})
	logger.Record(Trace{
		ID:              "req_2",
		Timestamp:       now.Add(-10 * time.Minute),
		Provider:        "anthropic",
		Model:           "claude-3-7-sonnet",
		Language:        "EN",
		StatusCode:      200,
		DurationMs:      25.5,
		AnonDurationUs:  180,
		EntitiesCount:   2,
		EntityTypes:     []string{"EMAIL", "API_KEY"},
		PIITypes:        []string{"EMAIL"},
		SecretTypes:     []string{"API_KEY"},
		SanitizedPrompt: "Ticket from [PG_def456_EMAIL_1] with [PG_def456_API_KEY_1].",
	})
	logger.Record(Trace{
		ID:              "req_3",
		Timestamp:       now.Add(-2 * time.Minute),
		Provider:        "ollama",
		Model:           "ollama/llama3",
		Language:        "PT",
		StatusCode:      200,
		DurationMs:      8.0,
		AnonDurationUs:  60,
		EntitiesCount:   0,
		SanitizedPrompt: "Olá, tudo bem?",
	})

	if err := logger.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Re-open the SQLite file to verify persistence across restarts
	reopened, err := NewSQLiteTraceLogger(dbPath, 3)
	if err != nil {
		t.Fatalf("reopening SQLite logger failed: %v", err)
	}
	defer reopened.Close()

	traces := reopened.GetAll()
	if len(traces) != 3 {
		t.Fatalf("expected 3 persisted traces, got %d", len(traces))
	}
	if traces[0].ID != "req_3" || traces[2].ID != "req_1" {
		t.Fatalf("expected newest trace first, got IDs: %s, %s, %s", traces[0].ID, traces[1].ID, traces[2].ID)
	}

	summary := reopened.GetSummary()
	if summary.TotalRequests != 3 {
		t.Errorf("expected TotalRequests=3, got %d", summary.TotalRequests)
	}
	if summary.MaskedRequests != 2 {
		t.Errorf("expected MaskedRequests=2, got %d", summary.MaskedRequests)
	}
	if summary.TotalEntities != 4 {
		t.Errorf("expected TotalEntities=4, got %d", summary.TotalEntities)
	}
	if summary.RequestsLastHour != 2 {
		t.Errorf("expected RequestsLastHour=2, got %d", summary.RequestsLastHour)
	}
	if summary.AvgScanUs != 120 {
		t.Errorf("expected AvgScanUs=120, got %d", summary.AvgScanUs)
	}
	if summary.ProviderDistribution["openai"] != 1 || summary.ProviderDistribution["anthropic"] != 1 || summary.ProviderDistribution["ollama"] != 1 {
		t.Errorf("unexpected ProviderDistribution: %+v", summary.ProviderDistribution)
	}

	// Verify maxRows pruning (maxRows = 3)
	reopened.Record(Trace{
		ID:              "req_4",
		Timestamp:       now,
		Provider:        "openai",
		Model:           "gpt-4o-mini",
		Language:        "ES",
		StatusCode:      200,
		AnonDurationUs:  100,
		EntitiesCount:   1,
		EntityTypes:     []string{"SECRET"},
		SecretTypes:     []string{"PASSWORD"},
		SanitizedPrompt: "Clave [PG_999999_PASSWORD_1]",
	})

	afterPrune := reopened.GetAll()
	if len(afterPrune) != 3 {
		t.Fatalf("expected 3 traces after pruning, got %d", len(afterPrune))
	}
	if afterPrune[0].ID != "req_4" || afterPrune[2].ID != "req_2" {
		t.Errorf("expected req_1 to be pruned, remaining IDs: %s, %s, %s", afterPrune[0].ID, afterPrune[1].ID, afterPrune[2].ID)
	}

	// Verify zero sensitive data exists on disk in raw SQLite bytes
	rawBytes, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("reading sqlite file: %v", err)
	}
	if strings.Contains(string(rawBytes), "token_map") {
		t.Fatalf("unexpected token_map column/data found in sqlite file")
	}
}

func TestClassifyEntitiesAndLanguageDetection(t *testing.T) {
	entities := []anonymizer.DetectedEntity{
		{Type: detector.EntityName, Token: "[PG_1_PERSON_1]"},
		{Type: detector.EntityEmail, Token: "[PG_1_EMAIL_1]"},
		{Type: detector.EntityAPIKey, Token: "[PG_1_API_KEY_1]"},
		{Type: detector.EntitySecret, SubType: "PASSWORD", Token: "[PG_1_PASSWORD_1]"},
		{Type: detector.EntitySecret, SubType: "JWT", Token: "[PG_1_JWT_1]"},
	}

	policyTypes, piiTypes, secretTypes := ClassifyEntities(entities)
	if len(policyTypes) != 5 {
		t.Fatalf("expected 5 policyTypes, got %v", policyTypes)
	}
	if strings.Join(piiTypes, ",") != "PERSON,EMAIL" {
		t.Errorf("expected PII types PERSON,EMAIL, got %v", piiTypes)
	}
	if strings.Join(secretTypes, ",") != "API_KEY,PASSWORD,JWT" {
		t.Errorf("expected Secret types API_KEY,PASSWORD,JWT, got %v", secretTypes)
	}

	langCases := []struct {
		text string
		want string
	}{
		{"Olá, preciso enviar uma cobrança para o cliente no CPF informado.", "PT"},
		{"Please help me respond to the support ticket regarding the error.", "EN"},
		{"Hola, necesito ayuda con mi cuenta y contraseña por favor.", "ES"},
		{"Bonjour, merci pour votre aide avec mon compte.", "FR"},
		{"Ciao, grazie per il tuo aiuto con questo utente.", "IT"},
		{"Bitte helfen Sie mir mit meinem Konto und Passwort.", "DE"},
	}
	for _, tc := range langCases {
		if got := DetectLanguage(tc.text); got != tc.want {
			t.Errorf("DetectLanguage(%q) = %q, want %q", tc.text, got, tc.want)
		}
	}
}
