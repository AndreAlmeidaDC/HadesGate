package vault

import "time"

// Mapping stores the token-to-plaintext relationship for a session.
type Mapping struct {
	Token     string    `json:"token"`
	PlainText string    `json:"plain_text"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Vault manages ephemeral token surrogate mappings.
type Vault interface {
	Put(sessionID, token, plainText, entityType string, ttl time.Duration)
	GetPlainText(sessionID, token string) (string, bool)
	GetAllSessionMappings(sessionID string) map[string]string
	DeleteSession(sessionID string)
	ActiveSessionsCount() int
}
