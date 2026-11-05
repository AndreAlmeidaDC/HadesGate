package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"sync"
	"time"
)

type encryptedMapping struct {
	Token      string
	Ciphertext []byte
	Nonce      []byte
	Type       string
	CreatedAt  time.Time
	ExpiresAt  time.Time
}

type sessionData struct {
	tokens    map[string]encryptedMapping
	expiresAt time.Time
}

// MemoryVault provides an in-memory, thread-safe implementation of Vault with automatic TTL cleanup
// and volatile AES-256-GCM RAM encryption for all surrogate tokens held on the heap.
type MemoryVault struct {
	mu        sync.RWMutex
	sessions  map[string]*sessionData
	stopChan  chan struct{}
	cipherKey []byte
	aead      cipher.AEAD
}

// NewMemoryVault creates and starts an in-memory token vault with volatile AES-256-GCM RAM encryption.
func NewMemoryVault(cleanupInterval time.Duration) *MemoryVault {
	key := make([]byte, 32) // AES-256
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		panic(fmt.Sprintf("failed to generate random vault encryption key: %v", err))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(fmt.Sprintf("failed to create AES cipher for vault: %v", err))
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		panic(fmt.Sprintf("failed to create GCM AEAD for vault: %v", err))
	}

	v := &MemoryVault{
		sessions:  make(map[string]*sessionData),
		stopChan:  make(chan struct{}),
		cipherKey: key,
		aead:      aead,
	}
	go v.startJanitor(cleanupInterval)
	return v
}

// Close stops the janitor goroutine and zeroes out the volatile encryption key from RAM.
func (v *MemoryVault) Close() {
	v.mu.Lock()
	defer v.mu.Unlock()

	select {
	case <-v.stopChan:
	default:
		close(v.stopChan)
	}

	for i := range v.cipherKey {
		v.cipherKey[i] = 0
	}
}

func (v *MemoryVault) startJanitor(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			v.purgeExpired()
		case <-v.stopChan:
			return
		}
	}
}

func (v *MemoryVault) purgeExpired() {
	now := time.Now()
	v.mu.Lock()
	defer v.mu.Unlock()

	for id, sess := range v.sessions {
		if now.After(sess.expiresAt) {
			for k, enc := range sess.tokens {
				for i := range enc.Ciphertext {
					enc.Ciphertext[i] = 0
				}
				for i := range enc.Nonce {
					enc.Nonce[i] = 0
				}
				delete(sess.tokens, k)
			}
			delete(v.sessions, id)
		}
	}
}

// Put stores an encrypted mapping into the vault for a given session using AES-256-GCM.
func (v *MemoryVault) Put(sessionID, token, plainText, entityType string, ttl time.Duration) {
	now := time.Now()
	exp := now.Add(ttl)

	nonce := make([]byte, v.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return
	}

	// Bind authenticated data (AAD) to sessionID + token to prevent token transplantation
	aad := []byte(sessionID + ":" + token)
	ciphertext := v.aead.Seal(nil, nonce, []byte(plainText), aad)

	v.mu.Lock()
	defer v.mu.Unlock()

	sess, exists := v.sessions[sessionID]
	if !exists {
		sess = &sessionData{
			tokens:    make(map[string]encryptedMapping),
			expiresAt: exp,
		}
		v.sessions[sessionID] = sess
	}

	if exp.After(sess.expiresAt) {
		sess.expiresAt = exp
	}

	sess.tokens[token] = encryptedMapping{
		Token:      token,
		Ciphertext: ciphertext,
		Nonce:      nonce,
		Type:       entityType,
		CreatedAt:  now,
		ExpiresAt:  exp,
	}
}

// GetPlainText retrieves and decrypts the value for a token using AES-256-GCM.
func (v *MemoryVault) GetPlainText(sessionID, token string) (string, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	sess, exists := v.sessions[sessionID]
	if !exists {
		return "", false
	}

	enc, found := sess.tokens[token]
	if !found {
		return "", false
	}

	aad := []byte(sessionID + ":" + token)
	plaintext, err := v.aead.Open(nil, enc.Nonce, enc.Ciphertext, aad)
	if err != nil {
		return "", false
	}

	return string(plaintext), true
}

// GetAllSessionMappings retrieves and decrypts a copy of all mappings for a session.
func (v *MemoryVault) GetAllSessionMappings(sessionID string) map[string]string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	sess, exists := v.sessions[sessionID]
	if !exists {
		return nil
	}

	res := make(map[string]string, len(sess.tokens))
	for token, enc := range sess.tokens {
		aad := []byte(sessionID + ":" + token)
		if plaintext, err := v.aead.Open(nil, enc.Nonce, enc.Ciphertext, aad); err == nil {
			res[token] = string(plaintext)
		}
	}
	return res
}

// DeleteSession purges a session immediately and zeroes memory buffers.
func (v *MemoryVault) DeleteSession(sessionID string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if sess, exists := v.sessions[sessionID]; exists {
		for k, enc := range sess.tokens {
			for i := range enc.Ciphertext {
				enc.Ciphertext[i] = 0
			}
			for i := range enc.Nonce {
				enc.Nonce[i] = 0
			}
			delete(sess.tokens, k)
		}
		delete(v.sessions, sessionID)
	}
}

// ActiveSessionsCount returns the current number of sessions in memory.
func (v *MemoryVault) ActiveSessionsCount() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.sessions)
}
