package detector

import (
	"testing"
)

func TestValidateCPF(t *testing.T) {
	tests := []struct {
		name     string
		cpf      string
		expected bool
	}{
		// Valid CPFs (standard test vectors)
		{"valid formatted", "529.982.247-25", true},
		{"valid unformatted", "52998224725", true},
		{"valid test cpf 2", "111.444.777-35", true},
		// Invalid CPFs
		{"invalid check digits", "123.456.789-00", false},
		{"all identical digits", "111.111.111-11", false},
		{"all zeros", "000.000.000-00", false},
		{"too short", "1234567890", false},
		{"random string", "abcdefghijk", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateCPF(tt.cpf)
			if result != tt.expected {
				t.Errorf("ValidateCPF(%q) = %v, expected %v", tt.cpf, result, tt.expected)
			}
		})
	}
}

func TestValidateCNPJ(t *testing.T) {
	tests := []struct {
		name     string
		cnpj     string
		expected bool
	}{
		// Valid CNPJ test vectors
		{"valid formatted", "11.222.333/0001-81", true},
		{"valid unformatted", "11222333000181", true},
		// Invalid CNPJ
		{"invalid checksum", "11.222.333/0001-99", false},
		{"all equal digits", "11.111.111/1111-11", false},
		{"too short", "1122233300018", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateCNPJ(tt.cnpj)
			if result != tt.expected {
				t.Errorf("ValidateCNPJ(%q) = %v, expected %v", tt.cnpj, result, tt.expected)
			}
		})
	}
}

func TestValidateCreditCard(t *testing.T) {
	tests := []struct {
		name     string
		card     string
		expected bool
	}{
		// Valid test cards (Luhn compliant)
		{"valid visa test", "4532 0151 1283 0366", true},
		{"valid mastercard test", "5425 2334 3010 9903", true},
		// Invalid cards
		{"invalid luhn", "4532 0151 1283 0367", false},
		{"too short", "1234 5678", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateCreditCard(tt.card)
			if result != tt.expected {
				t.Errorf("ValidateCreditCard(%q) = %v, expected %v", tt.card, result, tt.expected)
			}
		})
	}
}

func TestValidatePhone(t *testing.T) {
	tests := []struct {
		name     string
		phone    string
		expected bool
	}{
		{"valid mobile with ddd", "(11) 98765-4321", true},
		{"valid mobile with country code", "+55 11 98765-4321", true},
		{"valid landline", "(11) 3456-7890", true},
		{"invalid ddd", "(01) 98765-4321", false},
		{"invalid mobile length", "9876-4321", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidatePhone(tt.phone)
			if result != tt.expected {
				t.Errorf("ValidatePhone(%q) = %v, expected %v", tt.phone, result, tt.expected)
			}
		})
	}
}

func TestValidateCNH(t *testing.T) {
	tests := []struct {
		name     string
		cnh      string
		expected bool
	}{
		{"valid cnh test vector 1", "12345678900", true},
		{"all zeros invalid", "00000000000", false},
		{"all identical invalid", "11111111111", false},
		{"too short", "1234567890", false},
		{"invalid check digits", "12345678912", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateCNH(tt.cnh)
			if result != tt.expected {
				t.Errorf("ValidateCNH(%q) = %v, expected %v", tt.cnh, result, tt.expected)
			}
		})
	}
}

func TestPIXDetector(t *testing.T) {
	d := NewPIXDetector()

	t.Run("evp with pix context", func(t *testing.T) {
		text := "Por favor faça o pagamento na minha chave pix: 123e4567-e89b-12d3-a456-426614174000 agora."
		matches := d.Detect(text)
		if len(matches) == 0 {
			t.Fatalf("expected matches, got 0")
		}
		found := false
		for _, m := range matches {
			if m.Type == EntityPIX && m.Score >= 0.90 {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected high-confidence PIX match, got: %+v", matches)
		}
	})

	t.Run("uuid without pix context", func(t *testing.T) {
		text := "Identificador da transação no banco de dados: 123e4567-e89b-12d3-a456-426614174000"
		matches := d.Detect(text)
		for _, m := range matches {
			if m.Value == "123e4567-e89b-12d3-a456-426614174000" && m.Score >= 0.70 {
				t.Errorf("expected low confidence score without pix context, got %v", m.Score)
			}
		}
	})
}

func TestRGDetector(t *testing.T) {
	d := NewRGDetector()

	t.Run("rg with context", func(t *testing.T) {
		text := "Portador do documento RG 12.345.678-9 SSP/SP"
		matches := d.Detect(text)
		if len(matches) == 0 {
			t.Fatalf("expected RG match, got 0")
		}
		if matches[0].Score < 0.80 {
			t.Errorf("expected high score for RG with context, got %v", matches[0].Score)
		}
	})

	t.Run("number without rg context", func(t *testing.T) {
		text := "O pedido de compra 12.345.678-9 foi aprovado"
		matches := d.Detect(text)
		if len(matches) > 0 && matches[0].Score >= 0.70 {
			t.Errorf("expected score below 0.70 for non-RG context, got %v", matches[0].Score)
		}
	})
}

func TestBankAccountDetector(t *testing.T) {
	d := NewBankAccountDetector()

	t.Run("bank account with agency and bank name", func(t *testing.T) {
		text := "Transferir no Banco Itaú, agência 1234 conta corrente 56789-0"
		matches := d.Detect(text)
		if len(matches) == 0 {
			t.Fatalf("expected bank match, got 0")
		}
		for _, m := range matches {
			if m.Type != EntityBankAccount || m.Score < 0.80 {
				t.Errorf("expected valid bank account match with high score, got %+v", m)
			}
		}
	})
}

func TestPassportDetector(t *testing.T) {
	d := NewPassportDetector()

	t.Run("passport with context", func(t *testing.T) {
		text := "Apresentar o passaporte CS123456 na imigração da Polícia Federal"
		matches := d.Detect(text)
		if len(matches) == 0 {
			t.Fatalf("expected passport match, got 0")
		}
		if matches[0].Score < 0.90 {
			t.Errorf("expected high score for passport with context, got %v", matches[0].Score)
		}
	})
}

func TestCustomDetector(t *testing.T) {
	rules := []CustomRule{
		{
			Name:          "MATRICULA_RH",
			Pattern:       `\bEMP-[0-9]{4}\b`,
			ContextWords:  []string{"matricula", "cracha", "colaborador"},
			BaseScore:     0.60,
			EnhancedScore: 0.95,
		},
	}
	d := NewCustomDetector(rules)

	t.Run("with context", func(t *testing.T) {
		text := "Crachá do colaborador: EMP-1234"
		matches := d.Detect(text)
		if len(matches) == 0 {
			t.Fatalf("expected custom match, got 0")
		}
		if matches[0].Score != 0.95 {
			t.Errorf("expected score 0.95, got %v", matches[0].Score)
		}
	})

	t.Run("without context", func(t *testing.T) {
		text := "Código técnico EMP-1234 do relatório"
		matches := d.Detect(text)
		if len(matches) == 0 {
			t.Fatalf("expected custom match, got 0")
		}
		if matches[0].Score != 0.60 {
			t.Errorf("expected score 0.60, got %v", matches[0].Score)
		}
	})
}

func TestCalculateShannonEntropy(t *testing.T) {
	if h := CalculateShannonEntropy(""); h != 0.0 {
		t.Errorf("expected 0.0 for empty string, got %v", h)
	}

	if h := CalculateShannonEntropy("aaaaaaaaaaaaaaaa"); h != 0.0 {
		t.Errorf("expected 0.0 for single repeated char, got %v", h)
	}

	// Repetitive words have low entropy (< 3.0)
	hRepetitive := CalculateShannonEntropy("passwordpassword")
	if hRepetitive > 3.1 {
		t.Errorf("expected repetitive word entropy < 3.1, got %v", hRepetitive)
	}

	// Random 32-char hex token should have high entropy (>= 3.5)
	hHex := CalculateShannonEntropy("4a8f9c1b2e3d7e8f0a1b2c3d4e5f6a7b")
	if hHex < 3.5 {
		t.Errorf("expected hex key entropy >= 3.5, got %v", hHex)
	}
}

func TestAPIKeyDetector(t *testing.T) {
	d := NewAPIKeyDetector()

	t.Run("openai project key", func(t *testing.T) {
		text := "Use a chave OpenAI: sk-proj-1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890 para testar."
		matches := d.Detect(text)
		if len(matches) == 0 {
			t.Fatalf("expected OpenAI key match, got 0")
		}
		if matches[0].Type != EntityAPIKey || matches[0].Score != 1.0 {
			t.Errorf("expected EntityAPIKey with score 1.0, got: %+v", matches[0])
		}
	})

	t.Run("aws access key", func(t *testing.T) {
		text := "Configurar AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE na AWS"
		matches := d.Detect(text)
		if len(matches) == 0 {
			t.Fatalf("expected AWS key match, got 0")
		}
		if matches[0].Value != "AKIAIOSFODNN7EXAMPLE" {
			t.Errorf("expected AKIAIOSFODNN7EXAMPLE, got %s", matches[0].Value)
		}
	})

	t.Run("github personal token", func(t *testing.T) {
		text := "Clone com token ghp_1234567890abcdefghijklmnopqrstuvwxyz"
		matches := d.Detect(text)
		if len(matches) == 0 {
			t.Fatalf("expected GitHub token match, got 0")
		}
		if matches[0].Type != EntityAPIKey {
			t.Errorf("expected EntityAPIKey, got %s", matches[0].Type)
		}
	})

	t.Run("generic high-entropy api key with context", func(t *testing.T) {
		text := `api_key = "a8f9c1b2e3d7e8f0a1b2c3d4e5f6a7b8"`
		matches := d.Detect(text)
		if len(matches) == 0 {
			t.Fatalf("expected generic high-entropy API key match, got 0")
		}
		if matches[0].Score < 0.90 {
			t.Errorf("expected score >= 0.90, got %v", matches[0].Score)
		}
	})

	t.Run("generic low-entropy string with key context should be ignored", func(t *testing.T) {
		text := `api_key = "aaaaaaaaaaaaaaaa"`
		matches := d.Detect(text)
		if len(matches) > 0 {
			t.Errorf("expected low entropy string to be rejected, but matched: %+v", matches)
		}
	})
}

func TestSecretDetector(t *testing.T) {
	d := NewSecretDetector()

	t.Run("pem rsa private key", func(t *testing.T) {
		text := "A chave privada do servidor é:\n-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEA0Y123456789...\n-----END RSA PRIVATE KEY-----\nFavor guardar."
		matches := d.Detect(text)
		if len(matches) == 0 {
			t.Fatalf("expected PEM private key match, got 0")
		}
		if matches[0].Type != EntitySecret || matches[0].Score != 1.0 {
			t.Errorf("expected EntitySecret with score 1.0, got: %+v", matches[0])
		}
	})

	t.Run("database connection string with password", func(t *testing.T) {
		text := "Conecte no banco: postgres://admin:SuperSecretPass123@db.empresa.com:5432/producao_db"
		matches := d.Detect(text)
		if len(matches) == 0 {
			t.Fatalf("expected database URI match, got 0")
		}
		if matches[0].Type != EntitySecret {
			t.Errorf("expected EntitySecret, got %s", matches[0].Type)
		}
	})

	t.Run("jwt token", func(t *testing.T) {
		text := "Authorization token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
		matches := d.Detect(text)
		if len(matches) == 0 {
			t.Fatalf("expected JWT match, got 0")
		}
		if matches[0].Type != EntitySecret {
			t.Errorf("expected EntitySecret, got %s", matches[0].Type)
		}
	})

	t.Run("password assignment and ssh key", func(t *testing.T) {
		text := `DB_PASSWORD="Empresa@2026!Prod" e chave ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl`
		matches := d.Detect(text)
		if len(matches) < 2 {
			t.Fatalf("expected at least 2 secret matches (password + ssh key), got %d: %+v", len(matches), matches)
		}
	})
}

func TestIBANDetector(t *testing.T) {
	d := NewIBANDetector()

	t.Run("valid German IBAN with trailing prose", func(t *testing.T) {
		text := "Envie para o IBAN DE89 3704 0044 0532 0130 00 AGORA por favor."
		matches := d.Detect(text)
		if len(matches) != 1 {
			t.Fatalf("expected 1 IBAN match, got %d: %+v", len(matches), matches)
		}
		if matches[0].Value != "DE89 3704 0044 0532 0130 00" {
			t.Errorf("expected peeled IBAN 'DE89 3704 0044 0532 0130 00', got %q", matches[0].Value)
		}
	})

	t.Run("valid Brazilian IBAN", func(t *testing.T) {
		text := "Conta internacional BR1800360305000010009795493C1"
		matches := d.Detect(text)
		if len(matches) != 1 {
			t.Fatalf("expected 1 Brazilian IBAN match, got %d", len(matches))
		}
	})

	t.Run("invalid IBAN checksum", func(t *testing.T) {
		text := "IBAN inválido DE00 3704 0044 0532 0130 00"
		matches := d.Detect(text)
		if len(matches) != 0 {
			t.Errorf("expected 0 matches for invalid IBAN checksum, got %+v", matches)
		}
	})
}

func TestIPDetector(t *testing.T) {
	d := NewIPDetector()

	t.Run("valid IPv4 and IPv6", func(t *testing.T) {
		text := "Servidor de produção em 10.24.16.88 e IPv6 2001:0db8:85a3:0000:0000:8a2e:0370:7334."
		matches := d.Detect(text)
		if len(matches) != 2 {
			t.Fatalf("expected 2 IP matches, got %d: %+v", len(matches), matches)
		}
	})

	t.Run("ignores semantic version strings", func(t *testing.T) {
		text := "Atualizado para a versão v1.2.3.4 e release 2.4.6.8."
		matches := d.Detect(text)
		if len(matches) != 0 {
			t.Errorf("expected 0 matches for version numbers, got %+v", matches)
		}
	})
}

func TestExtractSurroundingContext_UTF8Safe(t *testing.T) {
	text := "Atenção: coração, ação, canção, seção, proteção 529.982.247-25 confidencial, validação, exceção, solução, direção, operação."
	start := 56
	end := 70
	ctx := ExtractSurroundingContext(text, start, end, 3)
	if ctx == "" {
		t.Fatalf("expected non-empty surrounding context")
	}
}

func TestNameDetector_BoundaryAndDeterministicPatterns(t *testing.T) {
	d := NewNameDetector()

	tests := []struct {
		name      string
		text      string
		wantNames []string
	}{
		{
			name:      "user bug: name followed by de CPF without comma",
			text:      "Vamso criar uma cobrança para Rodolfo Bandeira de CPF: 09897684492.",
			wantNames: []string{"Rodolfo Bandeira"},
		},
		{
			name:      "name followed directly by CPF acronym without preposition",
			text:      "Cliente Rodolfo Bandeira CPF 09897684492 solicitou segunda via.",
			wantNames: []string{"Rodolfo Bandeira"},
		},
		{
			name:      "name followed by de RG and SSP",
			text:      "Enviar contrato para Mariana Costa de RG 12.345.678-9 SSP.",
			wantNames: []string{"Mariana Costa"},
		},
		{
			name:      "name with particle followed by CNH",
			text:      "Motorista João Carlos da Silva de CNH 12345678900.",
			wantNames: []string{"João Carlos da Silva"},
		},
		{
			name:      "unlisted given name and surname with strong relational and document context",
			text:      "Vamos gerar cobrança para Kalel Vancleef de CPF: 09897684492.",
			wantNames: []string{"Kalel Vancleef"},
		},
		{
			name:      "standalone name at start of sentence followed by CPF apposition",
			text:      "Rodolfo Bandeira de CPF: 09897684492 precisa de atendimento.",
			wantNames: []string{"Rodolfo Bandeira"},
		},
		{
			name:      "name followed by institution or city via preposition",
			text:      "Transferir para Rodolfo Bandeira no Banco Central em São Paulo.",
			wantNames: []string{"Rodolfo Bandeira"},
		},
		{
			name:      "all-caps name followed by CPF",
			text:      "Confirme o cadastro de RODOLFO BANDEIRA de CPF 09897684492.",
			wantNames: []string{"RODOLFO BANDEIRA"},
		},
		{
			name:      "negative: place and institution names must not match",
			text:      "Atualizamos para São Paulo na versão v1.2.3.4 com Banco Central e Receita Federal.",
			wantNames: nil,
		},
		{
			name:      "negative: document and payment labels must not match",
			text:      "Emitir Nota Fiscal de Serviço e pagar via Chave Pix ou Boleto Bancário.",
			wantNames: nil,
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
					t.Errorf("match[%d].Value = %q, want %q (full match: %+v)", i, matches[i].Value, want, matches[i])
				}
				if tc.text[matches[i].Start:matches[i].End] != want {
					t.Errorf("byte slice text[%d:%d] = %q, want %q", matches[i].Start, matches[i].End, tc.text[matches[i].Start:matches[i].End], want)
				}
			}
		})
	}
}
