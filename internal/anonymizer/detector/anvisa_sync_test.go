package detector

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAnvisaSynchronizer_AlreadyUpToDate(t *testing.T) {
	headCalls := 0
	getCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			headCalls++
			w.Header().Set("ETag", DefaultAnvisaETag)
			w.Header().Set("Last-Modified", DefaultAnvisaLastModified)
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodGet {
			getCalls++
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	med := NewMedicationDetector()
	sync := NewAnvisaSynchronizer(med)
	sync.SetEndpoint(server.URL)

	res, err := sync.CheckAndSync(context.Background(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.UpToDate {
		t.Errorf("expected UpToDate to be true, got false")
	}
	if res.Updated {
		t.Errorf("expected Updated to be false, got true")
	}
	if headCalls != 1 {
		t.Errorf("expected 1 HEAD call, got %d", headCalls)
	}
	if getCalls != 0 {
		t.Errorf("expected 0 GET calls when up to date, got %d", getCalls)
	}
}

func TestAnvisaSynchronizer_StreamUpdate(t *testing.T) {
	csvData := `TIPO_PRODUTO;NOME_PRODUTO;DATA_FINALIZACAO_PROCESSO;CATEGORIA_REGULATORIA;NUMERO_REGISTRO_PRODUTO;DATA_VENCIMENTO_REGISTRO;NUMERO_PROCESSO;CLASSE_TERAPEUTICA;EMPRESA_DETENTORA_REGISTRO;SITUACAO_REGISTRO;PRINCIPIO_ATIVO
"MEDICAMENTO";"NOVO FARMACO TESTE";"22/12/1999";"Novo";113000267;"122009";"250000285639942";"ANTIDIABETICOS";"02685377000157";"Ativo";"substancia x, substancia y"
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("ETag", "new_etag_12345")
			w.Header().Set("Last-Modified", "Wed, 04 Nov 2026 22:00:00 GMT")
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "text/csv")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(csvData))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	med := NewMedicationDetector()
	sync := NewAnvisaSynchronizer(med)
	sync.SetEndpoint(server.URL)

	res, err := sync.CheckAndSync(context.Background(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.UpToDate {
		t.Errorf("expected UpToDate to be false")
	}
	if !res.Updated {
		t.Errorf("expected Updated to be true")
	}
	if res.ActiveETag != "new_etag_12345" {
		t.Errorf("expected ActiveETag 'new_etag_12345', got %q", res.ActiveETag)
	}

	// Verify detection of newly synced terms
	matches := med.Detect("Paciente iniciou tratamento com Novo Farmaco Teste 10mg.")
	if len(matches) == 0 {
		t.Fatalf("expected detection of newly synced medication, got none")
	}
}

func TestMedicationDetector_FullCatalogDetect(t *testing.T) {
	med := NewMedicationDetector()

	status := med.Status()
	if status.TermsCount < 20000 {
		t.Errorf("expected at least 20,000 terms from ANVISA catalog, got %d", status.TermsCount)
	}

	// Test 1: Commercial and generic medications with and without dosages
	prompt := "Receita médica: tomar Novalgina 500mg se tiver febre e Pregabalina 75mg à noite."
	matches := med.Detect(prompt)

	if len(matches) < 2 {
		t.Fatalf("expected at least 2 matches, got %d: %+v", len(matches), matches)
	}

	matchedValues := make([]string, 0, len(matches))
	for _, m := range matches {
		matchedValues = append(matchedValues, m.Value)
	}

	foundNovalgina := false
	foundPregabalina := false
	for _, v := range matchedValues {
		if strings.HasPrefix(strings.ToLower(v), "novalgina") {
			foundNovalgina = true
		}
		if strings.HasPrefix(strings.ToLower(v), "pregabalina") {
			foundPregabalina = true
		}
	}

	if !foundNovalgina {
		t.Errorf("expected 'Novalgina' in matches, got: %v", matchedValues)
	}
	if !foundPregabalina {
		t.Errorf("expected 'Pregabalina' in matches, got: %v", matchedValues)
	}

	// Test 2: Verify ambiguous terms like "sono" or "pleno" require context
	negPrompt := "Ele teve um sono tranquilo hoje."
	negMatches := med.Detect(negPrompt)
	if len(negMatches) > 0 {
		t.Errorf("expected 0 matches for everyday words without clinical context, got: %+v", negMatches)
	}

	// Test 3: Same word WITH clinical context (e.g. dosage or prescription)
	posPrompt := "O médico receitou Sono 10mg para insônia."
	posMatches := med.Detect(posPrompt)
	// If "sono" is in catalog, it will match with dosage
	if len(posMatches) > 0 && !strings.Contains(strings.ToLower(posMatches[0].Value), "sono") {
		t.Errorf("expected 'sono' match with dosage, got %+v", posMatches)
	}
}
