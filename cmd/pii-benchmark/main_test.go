package main

import "testing"

func TestValidateCorpusRejectsWeakOrAmbiguousFixtures(t *testing.T) {
	valid := []corpusCase{
		{ID: "positive", Text: "CPF 529.982.247-25", Expected: []expectedEntity{{Type: "CPF", Text: "529.982.247-25"}}},
		{ID: "negative", Text: "pedido 123", Expected: nil},
	}
	if err := validateCorpus(valid, 2, 1, 1); err != nil {
		t.Fatalf("valid corpus rejected: %v", err)
	}

	tests := []struct {
		name   string
		corpus []corpusCase
	}{
		{"duplicate id", append(valid, valid[0])},
		{"missing expected text", []corpusCase{{ID: "bad", Text: "sem CPF", Expected: []expectedEntity{{Type: "CPF", Text: "529.982.247-25"}}}}},
		{"duplicate expected", []corpusCase{{ID: "bad", Text: "CPF 529.982.247-25", Expected: []expectedEntity{{Type: "CPF", Text: "529.982.247-25"}, {Type: "CPF", Text: "529.982.247-25"}}}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateCorpus(test.corpus, 1, 0, 0); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestTypesBelowThresholdPreventsGlobalMetricsFromHidingWeakTypes(t *testing.T) {
	metrics := map[string]*typeMetrics{
		"CPF":        {Precision: 1, Recall: 1},
		"MEDICATION": {Precision: 0.75, Recall: 1},
		"PIX":        {Precision: 1, Recall: 0.5},
	}

	failed := typesBelowThreshold(metrics, 0.90, 0.90)
	if len(failed) != 2 || failed[0] != "MEDICATION" || failed[1] != "PIX" {
		t.Fatalf("unexpected failed types: %v", failed)
	}
}
