package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

const encryptedSecretPrefix = "enc:v1:"

type secretCipher struct {
	aead cipher.AEAD
}

// ParseMasterKey decodes a 32-byte AES key from base64 (standard/raw) or hex.
func ParseMasterKey(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("HADESGATE_CONFIG_MASTER_KEY is required")
	}
	decoders := []func(string) ([]byte, error){
		base64.StdEncoding.DecodeString,
		base64.RawStdEncoding.DecodeString,
		base64.URLEncoding.DecodeString,
		base64.RawURLEncoding.DecodeString,
		hex.DecodeString,
	}
	for _, decode := range decoders {
		key, err := decode(raw)
		if err == nil && len(key) == 32 {
			return key, nil
		}
	}
	return nil, fmt.Errorf("HADESGATE_CONFIG_MASTER_KEY must encode exactly 32 bytes (base64 or hex)")
}

func newSecretCipher(key []byte) (*secretCipher, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("config master key must be exactly 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating config cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating config AEAD: %w", err)
	}
	return &secretCipher{aead: aead}, nil
}

func (c *secretCipher) encrypt(plaintext, aad string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generating config nonce: %w", err)
	}
	sealed := c.aead.Seal(nonce, nonce, []byte(plaintext), []byte(aad))
	return encryptedSecretPrefix + base64.RawURLEncoding.EncodeToString(sealed), nil
}

func (c *secretCipher) decrypt(envelope, aad string) (string, error) {
	if envelope == "" {
		return "", nil
	}
	if !strings.HasPrefix(envelope, encryptedSecretPrefix) {
		return "", fmt.Errorf("secret is not a supported encrypted envelope")
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(envelope, encryptedSecretPrefix))
	if err != nil {
		return "", fmt.Errorf("decoding encrypted secret: %w", err)
	}
	if len(payload) < c.aead.NonceSize()+c.aead.Overhead() {
		return "", fmt.Errorf("encrypted secret is truncated")
	}
	nonce := payload[:c.aead.NonceSize()]
	ciphertext := payload[c.aead.NonceSize():]
	plaintext, err := c.aead.Open(nil, nonce, ciphertext, []byte(aad))
	if err != nil {
		return "", fmt.Errorf("decrypting secret: authentication failed")
	}
	return string(plaintext), nil
}

func systemSecretAAD(key string) string {
	return "hadesgate/system_config/" + key
}

func tenantSecretAAD(id string) string {
	return "hadesgate/tenant/" + id + "/proxy_key"
}
