package vault

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestMemoryVault_EncryptedStorage(t *testing.T) {
	v := NewMemoryVault(50 * time.Millisecond)
	defer v.Close()

	sessionID := "test-session-enc"
	token := "[HG_abc123_CPF_1]"
	plaintext := "529.982.247-25"

	v.Put(sessionID, token, plaintext, "CPF", time.Minute)

	// 1. Verify retrieval works
	retrieved, ok := v.GetPlainText(sessionID, token)
	if !ok {
		t.Fatalf("expected token to be found in vault")
	}
	if retrieved != plaintext {
		t.Fatalf("got %s, want %s", retrieved, plaintext)
	}

	// 2. Verify that in internal memory heap, plaintext is NOT stored raw
	v.mu.RLock()
	sess, exists := v.sessions[sessionID]
	if !exists {
		v.mu.RUnlock()
		t.Fatalf("session not found internally")
	}
	encMapping, hasToken := sess.tokens[token]
	v.mu.RUnlock()

	if !hasToken {
		t.Fatalf("token mapping missing internally")
	}

	if bytes.Contains(encMapping.Ciphertext, []byte(plaintext)) {
		t.Fatalf("CRITICAL SECURITY INVARIANT VIOLATION: plaintext %s found in ciphertext bytes!", plaintext)
	}

	// 3. Verify GetAllSessionMappings
	all := v.GetAllSessionMappings(sessionID)
	if len(all) != 1 || all[token] != plaintext {
		t.Fatalf("GetAllSessionMappings failed: got %+v", all)
	}

	// 4. Verify ActiveSessionsCount
	if v.ActiveSessionsCount() != 1 {
		t.Fatalf("expected 1 active session, got %d", v.ActiveSessionsCount())
	}

	// 5. Verify DeleteSession
	v.DeleteSession(sessionID)
	if v.ActiveSessionsCount() != 0 {
		t.Fatalf("expected 0 active sessions after delete, got %d", v.ActiveSessionsCount())
	}
	_, okAfterDelete := v.GetPlainText(sessionID, token)
	if okAfterDelete {
		t.Fatalf("expected token to be absent after session deletion")
	}
}

func TestMemoryVault_AADIntegrity(t *testing.T) {
	v := NewMemoryVault(time.Minute)
	defer v.Close()

	sessionID1 := "sess-1"
	sessionID2 := "sess-2"
	token := "[HG_001_SECRET_1]"
	plaintext := "sk-super-secret-key-12345"

	v.Put(sessionID1, token, plaintext, "SECRET", time.Minute)

	// Trying to read using sessionID2 must fail even if token matches
	_, ok := v.GetPlainText(sessionID2, token)
	if ok {
		t.Fatalf("expected isolation between sessions")
	}
}

func TestMemoryVault_Concurrency(t *testing.T) {
	v := NewMemoryVault(100 * time.Millisecond)
	defer v.Close()

	var wg sync.WaitGroup
	numGoroutines := 30
	numOps := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			sessID := fmt.Sprintf("sess-worker-%d", workerID)
			for j := 0; j < numOps; j++ {
				token := fmt.Sprintf("[HG_tok_%d_%d]", workerID, j)
				val := fmt.Sprintf("sensitive-data-%d-%d", workerID, j)

				v.Put(sessID, token, val, "DATA", 500*time.Millisecond)

				got, ok := v.GetPlainText(sessID, token)
				if !ok || got != val {
					t.Errorf("worker %d op %d: got %s (ok=%v), want %s", workerID, j, got, ok, val)
				}
			}
			v.DeleteSession(sessID)
		}(i)
	}

	wg.Wait()
}

