package detector

import (
	"testing"
)

func TestNameDetector_CompoundNamesFromCatalog(t *testing.T) {
	d := NewNameDetector()

	tests := []struct {
		name      string
		text      string
		wantNames []string
	}{
		{
			name:      "compound full name with two givens and one surname and particle",
			text:      "Olá Carlos Eduardo da Silva, seja bem-vindo ao portal.",
			wantNames: []string{"Carlos Eduardo da Silva"},
		},
		{
			name:      "compound name with given and surname and dos particle",
			text:      "O paciente João Pedro dos Santos realizou a consulta ontem.",
			wantNames: []string{"João Pedro dos Santos"},
		},
		{
			name:      "female compound name with de particle",
			text:      "Fatura enviada para Maria de Fátima Oliveira no início do mês.",
			wantNames: []string{"Maria de Fátima Oliveira"},
		},
		{
			name:      "compound name with das particle",
			text:      "Titular Francisca das Chagas Rodrigues confirmou a assinatura.",
			wantNames: []string{"Francisca das Chagas Rodrigues"},
		},
		{
			name:      "compound name with four tokens",
			text:      "Contrato em nome de Alexandre Magno Ferreira de Souza registrado em cartório.",
			wantNames: []string{"Alexandre Magno Ferreira de Souza"},
		},
		{
			name:      "name with de particle",
			text:      "Colaborador Glauber de Vasconcelos iniciou o projeto.",
			wantNames: []string{"Glauber de Vasconcelos"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			matches := d.Detect(tc.text)
			if len(matches) != len(tc.wantNames) {
				t.Fatalf("Detect(%q) returned %d matches %+v, want %d (%v)", tc.text, len(matches), matches, len(tc.wantNames), tc.wantNames)
			}
			for i, want := range tc.wantNames {
				if matches[i].Value != want {
					t.Errorf("match[%d].Value = %q, want %q", i, matches[i].Value, want)
				}
				if tc.text[matches[i].Start:matches[i].End] != want {
					t.Errorf("byte slice text[%d:%d] = %q, want %q", matches[i].Start, matches[i].End, tc.text[matches[i].Start:matches[i].End], want)
				}
				if matches[i].Score < 0.85 {
					t.Errorf("match score %v is unexpectedly low", matches[i].Score)
				}
			}
		})
	}
}

func TestNameDetector_SingleNamesWithTriggers(t *testing.T) {
	d := NewNameDetector()

	tests := []struct {
		name      string
		text      string
		wantNames []string
	}{
		{
			name:      "greeting trigger with single given name",
			text:      "Olá Marcos, como podemos te ajudar hoje?",
			wantNames: []string{"Marcos"},
		},
		{
			name:      "honorific doctor trigger with single given name",
			text:      "A Dra. Juliana receitou o novo tratamento ao paciente.",
			wantNames: []string{"Juliana"},
		},
		{
			name:      "client role trigger with single given name",
			text:      "Cliente Fernanda solicitou a emissão do comprovante.",
			wantNames: []string{"Fernanda"},
		},
		{
			name:      "formal salutation with single given name",
			text:      "Prezado Lucas, segue anexo o relatório solicitado.",
			wantNames: []string{"Lucas"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			matches := d.Detect(tc.text)
			if len(matches) != len(tc.wantNames) {
				t.Fatalf("Detect(%q) returned %d matches %+v, want %d (%v)", tc.text, len(matches), matches, len(tc.wantNames), tc.wantNames)
			}
			for i, want := range tc.wantNames {
				if matches[i].Value != want {
					t.Errorf("match[%d].Value = %q, want %q", i, matches[i].Value, want)
				}
				if tc.text[matches[i].Start:matches[i].End] != want {
					t.Errorf("byte slice text[%d:%d] = %q, want %q", matches[i].Start, matches[i].End, tc.text[matches[i].Start:matches[i].End], want)
				}
			}
		})
	}
}

func TestNameDetector_GeographicAndPlaceDisambiguation(t *testing.T) {
	d := NewNameDetector()

	tests := []struct {
		name string
		text string
	}{
		{
			name: "sao paulo city should not match person",
			text: "Viajamos para São Paulo no feriado de carnaval.",
		},
		{
			name: "rio de janeiro and belo horizonte should not match person",
			text: "O voo saiu do Rio de Janeiro com destino a Belo Horizonte nesta manhã.",
		},
		{
			name: "porto alegre and santa catarina should not match person",
			text: "Chegada em Porto Alegre via Santa Catarina pela rodovia federal.",
		},
		{
			name: "banco central and receita federal should not match person",
			text: "Banco Central do Brasil e Receita Federal emitiram nova portaria.",
		},
		{
			name: "street address with person-like words should not match person",
			text: "Entregar encomenda na Rua Silva Bueno número 1200.",
		},
		{
			name: "order and batch number prose should not match person",
			text: "O pedido 12345678 foi enviado para o lote 20260101.",
		},
		{
			name: "fiscal and financial terms should not match person",
			text: "Emitir Nota Fiscal de Serviço e pagar via Chave Pix ou Boleto Bancário.",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			matches := d.Detect(tc.text)
			if len(matches) > 0 {
				t.Errorf("Detect(%q) returned unexpected false positive matches: %+v", tc.text, matches)
			}
		})
	}
}

