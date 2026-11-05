package detector

import (
	"testing"
)

func TestValidateSSN(t *testing.T) {
	tests := []struct {
		name     string
		ssn      string
		expected bool
	}{
		{"valid formatted", "219-45-6780", true},
		{"valid spaced", "219 45 6780", true},
		{"valid unformatted", "219456780", true},
		{"invalid area 000", "000-45-6780", false},
		{"invalid area 666", "666-45-6780", false},
		{"invalid area 900+", "920-45-6780", false},
		{"invalid group 00", "219-00-6780", false},
		{"invalid serial 0000", "219-45-0000", false},
		{"invalid dummy SSA ad", "078-05-1120", false},
		{"invalid all equal", "111-11-1111", false},
		{"invalid sequential", "123-45-6789", false},
		{"invalid too short", "123-45-678", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateSSN(tt.ssn); got != tt.expected {
				t.Errorf("ValidateSSN(%q) = %v, expected %v", tt.ssn, got, tt.expected)
			}
		})
	}
}

func TestSSNDetector_Detect(t *testing.T) {
	d := NewSSNDetector()

	// Formatted detects without mandatory context
	matches := d.Detect("Employee social security number is 219-45-6780 on file.")
	if len(matches) != 1 || matches[0].Value != "219-45-6780" {
		t.Fatalf("expected 1 match for formatted SSN, got %+v", matches)
	}

	// Unformatted requires context
	matchesUnformattedWithCtx := d.Detect("SSN: 219456780")
	if len(matchesUnformattedWithCtx) != 1 || matchesUnformattedWithCtx[0].Value != "219456780" {
		t.Fatalf("expected 1 match for unformatted SSN with context, got %+v", matchesUnformattedWithCtx)
	}

	// Unformatted without context should be ignored
	matchesUnformattedNoCtx := d.Detect("Order number 219456780 was processed today.")
	if len(matchesUnformattedNoCtx) != 0 {
		t.Fatalf("expected 0 matches for unformatted SSN without context, got %+v", matchesUnformattedNoCtx)
	}
}

func TestValidateEIN(t *testing.T) {
	tests := []struct {
		name     string
		ein      string
		expected bool
	}{
		{"valid formatted", "12-3456789", true},
		{"valid prefix 20", "20-1234567", true},
		{"valid prefix 84", "84-9876543", true},
		{"invalid prefix 00", "00-1234567", false},
		{"invalid prefix 07", "07-1234567", false},
		{"invalid prefix 78", "78-1234567", false},
		{"invalid serial zeros", "12-0000000", false},
		{"invalid length", "12-34567", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateEIN(tt.ein); got != tt.expected {
				t.Errorf("ValidateEIN(%q) = %v, expected %v", tt.ein, got, tt.expected)
			}
		})
	}
}

func TestEINDetector_Detect(t *testing.T) {
	d := NewEINDetector()

	matches := d.Detect("Company EIN is 12-3456789 for tax filing.")
	if len(matches) != 1 || matches[0].Value != "12-3456789" {
		t.Fatalf("expected 1 match for EIN, got %+v", matches)
	}
}

func TestValidateNINO(t *testing.T) {
	tests := []struct {
		name     string
		nino     string
		expected bool
	}{
		{"valid with spaces and suffix", "RN 12 34 56 A", true},
		{"valid no spaces with suffix", "JH012345B", true},
		{"valid no suffix", "AB123456", true},
		{"invalid prefix BG", "BG123456A", false},
		{"invalid prefix GB", "GB123456A", false},
		{"invalid prefix ZZ", "ZZ123456A", false},
		{"disallowed first char D", "DA123456A", false},
		{"disallowed second char O", "AO123456A", false},
		{"disallowed suffix E", "AB123456E", false},
		{"invalid length", "AB12345", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateNINO(tt.nino); got != tt.expected {
				t.Errorf("ValidateNINO(%q) = %v, expected %v", tt.nino, got, tt.expected)
			}
		})
	}
}

func TestValidateNHS(t *testing.T) {
	tests := []struct {
		name     string
		nhs      string
		expected bool
	}{
		{"valid 943 476 5919", "943 476 5919", true},
		{"valid formatted hyphens", "943-476-5919", true},
		{"valid 485 777 3457", "485 777 3457", true},
		{"invalid check digit", "943 476 5918", false},
		{"invalid length", "943 476 591", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateNHS(tt.nhs); got != tt.expected {
				t.Errorf("ValidateNHS(%q) = %v, expected %v", tt.nhs, got, tt.expected)
			}
		})
	}
}

func TestValidateDNI_And_NIE(t *testing.T) {
	// DNI
	if !ValidateDNI("12345678Z") {
		t.Errorf("ValidateDNI(12345678Z) should be true")
	}
	if ValidateDNI("12345678A") {
		t.Errorf("ValidateDNI(12345678A) should be false")
	}

	// NIE
	if !ValidateNIE("X1234567L") {
		t.Errorf("ValidateNIE(X1234567L) should be true")
	}
	if ValidateNIE("X1234567A") {
		t.Errorf("ValidateNIE(X1234567A) should be false")
	}
}

func TestDNIDetector_Detect(t *testing.T) {
	d := NewDNIDetector()
	text := "El ciudadano español presentó DNI 12345678Z y el extranjero NIE X1234567L."
	matches := d.Detect(text)
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d (%+v)", len(matches), matches)
	}
}

func TestValidateNIR(t *testing.T) {
	tests := []struct {
		name     string
		nir      string
		expected bool
	}{
		{"valid male NIR", "1 85 12 75 108 042 79", true},
		{"valid female NIR", "2 90 05 92 012 345 44", true},
		{"invalid key", "1 85 12 75 108 042 99", false},
		{"invalid gender", "4 85 12 75 108 042 79", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateNIR(tt.nir); got != tt.expected {
				t.Errorf("ValidateNIR(%q) = %v, expected %v", tt.nir, got, tt.expected)
			}
		})
	}
}

func TestValidateCodiceFiscale(t *testing.T) {
	// Standard Italian tax code test vector: RSSMRA85M01H501Q
	if !ValidateCodiceFiscale("RSSMRA85M01H501Q") {
		t.Errorf("ValidateCodiceFiscale(RSSMRA85M01H501Q) should be true")
	}
	// Wrong check digit
	if ValidateCodiceFiscale("RSSMRA85M01H501A") {
		t.Errorf("ValidateCodiceFiscale(RSSMRA85M01H501A) should be false")
	}
}

func TestMedicalDetector_Detect(t *testing.T) {
	d := NewMedicalDetector()

	// DEA test
	deaText := "Dr. Smith DEA number is AB1234563 for prescription."
	deaMatches := d.Detect(deaText)
	if len(deaMatches) != 1 || deaMatches[0].SubType != "DEA" {
		t.Fatalf("expected 1 DEA match, got %+v", deaMatches)
	}

	// MRN test
	mrnText := "Paciente atendido no ambulatório, Prontuário: REC-987452 registrado."
	mrnMatches := d.Detect(mrnText)
	if len(mrnMatches) != 1 || mrnMatches[0].SubType != "MRN" || mrnMatches[0].Value != "REC-987452" {
		t.Fatalf("expected 1 MRN match, got %+v", mrnMatches)
	}

	// NPI test (1679580450 with 80840 Luhn)
	npiText := "Physician NPI: 1679580450 active."
	npiMatches := d.Detect(npiText)
	if len(npiMatches) != 1 || npiMatches[0].SubType != "NPI" || npiMatches[0].Value != "1679580450" {
		t.Fatalf("expected 1 NPI match, got %+v", npiMatches)
	}
}
