package detector

import (
	"testing"
)

func TestValidateSUS(t *testing.T) {
	tests := []struct {
		name     string
		sus      string
		expected bool
	}{
		{"valid definitive card (starts with 1)", "123456789010000", true},
		{"valid provisional card (starts with 7)", "700000000000005", true},
		{"invalid checksum", "123456789010009", false},
		{"invalid starting digit 3", "323456789010000", false},
		{"invalid length", "1234567890", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateSUS(tt.sus); got != tt.expected {
				t.Errorf("ValidateSUS(%q) = %v, expected %v", tt.sus, got, tt.expected)
			}
		})
	}
}

func TestSUSDetector_Detect(t *testing.T) {
	d := NewSUSDetector()
	text := "Paciente cadastrado no Cartão SUS número 123456789010000 na UBS."
	matches := d.Detect(text)
	if len(matches) != 1 || matches[0].Value != "123456789010000" {
		t.Fatalf("expected 1 SUS match, got %+v", matches)
	}
}

func TestValidateRENAVAM(t *testing.T) {
	tests := []struct {
		name     string
		renavam  string
		expected bool
	}{
		{"valid renavam test vector", "00123456789", true},
		{"invalid renavam checksum", "00123456781", false},
		{"all identical digits", "00000000000", false},
		{"invalid length", "12345678", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateRENAVAM(tt.renavam); got != tt.expected {
				t.Errorf("ValidateRENAVAM(%q) = %v, expected %v", tt.renavam, got, tt.expected)
			}
		})
	}
}

func TestVehicleDetector_Detect(t *testing.T) {
	d := NewVehicleDetector()

	// Mercosul Plate with vehicle context
	textMercosul := "O veículo possui a placa BRA2E19 registrada no Detran."
	matchesM := d.Detect(textMercosul)
	if len(matchesM) != 1 || matchesM[0].SubType != "LICENSE_PLATE" || matchesM[0].Value != "BRA2E19" {
		t.Fatalf("expected 1 Mercosul plate match, got %+v", matchesM)
	}

	// Antiga Plate with vehicle context
	textAntiga := "Carro autuado por multa da placa ABC-1234 na rodovia."
	matchesA := d.Detect(textAntiga)
	if len(matchesA) != 1 || matchesA[0].SubType != "LICENSE_PLATE" || matchesA[0].Value != "ABC-1234" {
		t.Fatalf("expected 1 Antiga plate match, got %+v", matchesA)
	}

	// RENAVAM with context
	textRenavam := "RENAVAM do veículo: 00123456789 para transferência."
	matchesR := d.Detect(textRenavam)
	if len(matchesR) != 1 || matchesR[0].SubType != "RENAVAM" || matchesR[0].Value != "00123456789" {
		t.Fatalf("expected 1 RENAVAM match, got %+v", matchesR)
	}
}

func TestValidateCNJProcess(t *testing.T) {
	tests := []struct {
		name     string
		process  string
		expected bool
	}{
		{"valid CNJ process", "0000001-39.2024.8.26.0100", true},
		{"invalid CNJ check digits", "0000001-00.2024.8.26.0100", false},
		{"invalid length", "0000001-39.2024.8.26", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateCNJProcess(tt.process); got != tt.expected {
				t.Errorf("ValidateCNJProcess(%q) = %v, expected %v", tt.process, got, tt.expected)
			}
		})
	}
}

func TestLegalDetector_Detect(t *testing.T) {
	d := NewLegalDetector()
	text := "Autos do processo judicial 0000001-39.2024.8.26.0100 em segredo de justiça."
	matches := d.Detect(text)
	if len(matches) != 1 || matches[0].Value != "0000001-39.2024.8.26.0100" {
		t.Fatalf("expected 1 CNJ legal match, got %+v", matches)
	}
}

func TestDeviceDetector_Detect(t *testing.T) {
	d := NewDeviceDetector()

	// MAC Address
	textMAC := "Endereço MAC da interface de rede: 00:1A:2B:3C:4D:5E conectado ao Wi-Fi."
	matchesMAC := d.Detect(textMAC)
	if len(matchesMAC) != 1 || matchesMAC[0].SubType != "MAC_ADDRESS" || matchesMAC[0].Value != "00:1A:2B:3C:4D:5E" {
		t.Fatalf("expected 1 MAC address match, got %+v", matchesMAC)
	}

	// IMEI (490154203237518 has valid Luhn checksum)
	textIMEI := "Smartphone roubado, IMEI do aparelho: 490154203237518 registrado."
	matchesIMEI := d.Detect(textIMEI)
	if len(matchesIMEI) != 1 || matchesIMEI[0].SubType != "IMEI" || matchesIMEI[0].Value != "490154203237518" {
		t.Fatalf("expected 1 IMEI match, got %+v", matchesIMEI)
	}
}

func TestCIDDetector_Detect(t *testing.T) {
	d := NewCIDDetector()

	// Prefixed CID
	textPrefixed := "Atestado médico com diagnóstico CID-10 J45.0 confirmado."
	matchesP := d.Detect(textPrefixed)
	if len(matchesP) != 1 || matchesP[0].Value != "J45.0" {
		t.Fatalf("expected 1 CID match, got %+v", matchesP)
	}
}

func TestMedicationDetector_Detect(t *testing.T) {
	d := NewMedicationDetector()

	// Medication with dosage
	textMed := "Receita médica: tomar Amoxicilina 500mg de 8 em 8 horas e Dipirona 1g se dor."
	matchesM := d.Detect(textMed)
	if len(matchesM) != 2 {
		t.Fatalf("expected 2 medication matches, got %d: %+v", len(matchesM), matchesM)
	}
}

func TestMedicationDetector_RejectsCatalogHomonymsAndMedicalIdentifiers(t *testing.T) {
	d := NewMedicationDetector()
	for _, text := range []string{
		"Banco Central do Brasil e Receita Federal publicaram a norma.",
		"Paciente no ambulatório, Prontuário REC-987452 registrado.",
	} {
		if matches := d.Detect(text); len(matches) != 0 {
			t.Fatalf("non-medication text %q produced matches: %+v", text, matches)
		}
	}
}
