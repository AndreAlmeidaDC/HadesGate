package anonymizer

import (
	"testing"

	"github.com/hadesgate/hadesgate/internal/anonymizer/detector"
)

func TestFamilyForEntity(t *testing.T) {
	tests := []struct {
		entity detector.EntityType
		want   Family
	}{
		{detector.EntityCPF, FamilyPersonal},
		{detector.EntityCreditCard, FamilyPersonal},
		{detector.EntityCNPJ, FamilyCorporate},
		{detector.EntityEIN, FamilyCorporate},
		{detector.EntityMedicalID, FamilyHealth},
		{detector.EntityNHS, FamilyHealth},
		{detector.EntitySSN, FamilyIntl},
		{detector.EntityDNI, FamilyIntl},
		{detector.EntityAPIKey, FamilySession},
		{detector.EntityIPAddress, FamilyDevice},
		{detector.EntityType("UNKNOWN_ENTITY"), FamilyPersonal},
	}

	for _, tt := range tests {
		t.Run(string(tt.entity), func(t *testing.T) {
			got := FamilyForEntity(tt.entity)
			if got != tt.want {
				t.Errorf("FamilyForEntity(%s) = %s, want %s", tt.entity, got, tt.want)
			}
		})
	}
}

func TestResolveConfidenceThreshold(t *testing.T) {
	globalMin := 0.70
	famThresholds := map[string]float64{
		"HEALTH_PHI": 0.60,
		"PERSONAL":   0.75,
	}
	entThresholds := map[string]float64{
		"PERSON": 0.88,
	}

	// 1. Entity-specific override beats family and global
	threshPerson := ResolveConfidenceThreshold(detector.EntityName, globalMin, famThresholds, entThresholds)
	if threshPerson != 0.88 {
		t.Errorf("expected PERSON threshold 0.88, got %f", threshPerson)
	}

	// 2. Family-specific override beats global
	threshMed := ResolveConfidenceThreshold(detector.EntityMedicalID, globalMin, famThresholds, entThresholds)
	if threshMed != 0.60 {
		t.Errorf("expected MedicalID threshold 0.60 from HEALTH_PHI family, got %f", threshMed)
	}

	threshCPF := ResolveConfidenceThreshold(detector.EntityCPF, globalMin, famThresholds, entThresholds)
	if threshCPF != 0.75 {
		t.Errorf("expected CPF threshold 0.75 from PERSONAL family, got %f", threshCPF)
	}

	// 3. Fallback to global when neither entity nor family has override
	threshSSN := ResolveConfidenceThreshold(detector.EntitySSN, globalMin, famThresholds, entThresholds)
	if threshSSN != 0.70 {
		t.Errorf("expected SSN threshold 0.70 from global default, got %f", threshSSN)
	}

	// 4. Default fallback when global is 0
	threshDefault := ResolveConfidenceThreshold(detector.EntitySSN, 0, nil, nil)
	if threshDefault != 0.70 {
		t.Errorf("expected default 0.70 when global is 0, got %f", threshDefault)
	}
}

