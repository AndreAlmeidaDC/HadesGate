package detector

import (
	"regexp"
	"strings"
)

var (
	// pemPrivateKeyRegex matches PEM-encoded private keys (RSA, DSA, EC, OpenSSH, PGP, etc.)
	pemPrivateKeyRegex = regexp.MustCompile(`-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z0-9 ]*PRIVATE KEY-----`)

	// sshKeyRegex matches inline OpenSSH public/authorized keys
	sshKeyRegex = regexp.MustCompile(`\b(?:ssh-(?:rsa|ed25519|dss)|ecdsa-sha2-nistp(?:256|384|521))\s+[A-Za-z0-9+/]{32,}={0,3}\b`)

	// dbConnStringRegex matches database, broker, and HTTP Basic Auth URIs containing credentials
	dbConnStringRegex = regexp.MustCompile(`(?i)\b(?:postgres(?:ql)?|mysql|mariadb|mongodb(?:\+srv)?|redis(?:s)?|amqp(?:s)?|mssql|sqlserver|clickhouse|cockroach(?:db)?|https?)://[^:\s@/]+:[^@\s/]+@[^\s/'"` + "`" + `]+(?:/[^\s'"` + "`" + `]*)?`)

	// jwtRegex matches standard JSON Web Tokens (JWT) with header, payload and signature
	jwtRegex = regexp.MustCompile(`\beyJ[a-zA-Z0-9_-]{10,}\.eyJ[a-zA-Z0-9_-]{10,}\.[a-zA-Z0-9_-]{10,}\b`)

	// passwordAssignmentRegex matches explicit password assignments in configs, env vars, or prompts
	passwordAssignmentRegex = regexp.MustCompile(`(?i)\b(?:password|passwd|pwd|senha|db_pass(?:word)?|database_pass(?:word)?|redis_pass(?:word)?|admin_pass(?:word)?)\s*[:=]\s*['"]?([^\s'"` + "`" + `,;]{6,64})['"]?`)

	ignoredPasswordPlaceholders = map[string]bool{
		"null": true, "none": true, "true": true, "false": true, "undefined": true,
		"password": true, "senha": true, "changeme": true, "secret": true, "<password>": true,
		"your_password": true, "sua_senha": true, "example": true, "exemplo": true,
	}
)

// SecretDetector detects cryptographic private keys, database connection strings with credentials, and JWTs.
type SecretDetector struct{}

// NewSecretDetector creates a new SecretDetector.
func NewSecretDetector() *SecretDetector {
	return &SecretDetector{}
}

// Type returns EntitySecret.
func (d *SecretDetector) Type() EntityType {
	return EntitySecret
}

// ContextWords returns keywords associated with infrastructure secrets and credentials.
func (d *SecretDetector) ContextWords() []string {
	return []string{
		"secret", "private_key", "pem", "rsa", "jwt", "database", "connection",
		"postgres", "mysql", "mongodb", "redis", "senha", "password",
	}
}

// Detect scans text for infrastructure secrets, connection strings and private keys.
func (d *SecretDetector) Detect(text string) []Match {
	if text == "" {
		return nil
	}

	var matches []Match

	// 1. Detect PEM Private Keys
	pemIndices := pemPrivateKeyRegex.FindAllStringIndex(text, -1)
	for _, idx := range pemIndices {
		val := text[idx[0]:idx[1]]
		ctxText := ExtractSurroundingContext(text, idx[0], idx[1], 5)
		combinedCtx := strings.ToLower(val + " " + ctxText)
		ctxFound := FindContextKeywords(combinedCtx, d.ContextWords())

		matches = append(matches, Match{
			Type:            EntitySecret,
			SubType:         "PRIVATE_KEY",
			Value:           val,
			Start:           idx[0],
			End:             idx[1],
			Score:           1.0,
			ContextFound:    ctxFound,
			ValidatorPassed: true,
		})
	}

	// 2. Detect Database Connection Strings with Passwords
	dbIndices := dbConnStringRegex.FindAllStringIndex(text, -1)
	for _, idx := range dbIndices {
		val := text[idx[0]:idx[1]]
		ctxText := ExtractSurroundingContext(text, idx[0], idx[1], 5)
		combinedCtx := strings.ToLower(val + " " + ctxText)
		ctxFound := FindContextKeywords(combinedCtx, d.ContextWords())

		matches = append(matches, Match{
			Type:            EntitySecret,
			SubType:         "DB_URI",
			Value:           val,
			Start:           idx[0],
			End:             idx[1],
			Score:           1.0,
			ContextFound:    ctxFound,
			ValidatorPassed: true,
		})
	}

	// 3. Detect JSON Web Tokens (JWT)
	jwtIndices := jwtRegex.FindAllStringIndex(text, -1)
	for _, idx := range jwtIndices {
		val := text[idx[0]:idx[1]]
		ctxText := ExtractSurroundingContext(text, idx[0], idx[1], 5)
		combinedCtx := strings.ToLower(val + " " + ctxText)
		ctxFound := FindContextKeywords(combinedCtx, d.ContextWords())

		matches = append(matches, Match{
			Type:            EntitySecret,
			SubType:         "JWT",
			Value:           val,
			Start:           idx[0],
			End:             idx[1],
			Score:           0.95,
			ContextFound:    ctxFound,
			ValidatorPassed: true,
		})
	}

	// 4. Detect Inline OpenSSH Keys
	sshIndices := sshKeyRegex.FindAllStringIndex(text, -1)
	for _, idx := range sshIndices {
		val := text[idx[0]:idx[1]]
		ctxText := ExtractSurroundingContext(text, idx[0], idx[1], 5)
		combinedCtx := strings.ToLower(val + " " + ctxText)
		ctxFound := FindContextKeywords(combinedCtx, d.ContextWords())

		matches = append(matches, Match{
			Type:            EntitySecret,
			SubType:         "SSH_KEY",
			Value:           val,
			Start:           idx[0],
			End:             idx[1],
			Score:           0.98,
			ContextFound:    ctxFound,
			ValidatorPassed: true,
		})
	}

	// 5. Detect Explicit Password Assignments (e.g. password="...", senha: ...)
	pwdSubs := passwordAssignmentRegex.FindAllStringSubmatchIndex(text, -1)
	for _, sub := range pwdSubs {
		if len(sub) >= 4 && sub[2] != -1 && sub[3] != -1 {
			val := text[sub[2]:sub[3]]
			lowerVal := strings.ToLower(val)
			if ignoredPasswordPlaceholders[lowerVal] || strings.HasPrefix(lowerVal, "[hg_") || strings.HasPrefix(lowerVal, "[pg_") {
				continue
			}
			// Reject all-equal masks like "********" or "aaaaaaaa"
			if CalculateShannonEntropy(val) < 2.0 {
				continue
			}

			ctxText := ExtractSurroundingContext(text, sub[2], sub[3], 5)
			combinedCtx := strings.ToLower(text[sub[0]:sub[1]] + " " + ctxText)
			ctxFound := FindContextKeywords(combinedCtx, d.ContextWords())

			matches = append(matches, Match{
				Type:            EntitySecret,
				SubType:         "PASSWORD",
				Value:           val,
				Start:           sub[2],
				End:             sub[3],
				Score:           0.92,
				ContextFound:    ctxFound,
				ValidatorPassed: true,
			})
		}
	}

	return matches
}
