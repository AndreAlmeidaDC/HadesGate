package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestDispatcher_DeliveryAndHMAC(t *testing.T) {
	secret := "test-webhook-secret-999"
	var receivedEvent Event
	var receivedSig string
	var mu sync.Mutex
	done := make(chan struct{}, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		receivedSig = r.Header.Get("X-HadesGate-Signature")
		body, _ := io.ReadAll(r.Body)

		// Verify signature against body
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		expectedSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

		if receivedSig != expectedSig {
			t.Errorf("HMAC signature mismatch: got %s, want %s", receivedSig, expectedSig)
		}

		if err := json.Unmarshal(body, &receivedEvent); err != nil {
			t.Errorf("failed to unmarshal webhook payload: %v", err)
		}

		w.WriteHeader(http.StatusOK)
		select {
		case done <- struct{}{}:
		default:
		}
	}))
	defer server.Close()

	d := NewDispatcher(Config{
		URL:     server.URL,
		Secret:  secret,
		Timeout: 2 * time.Second,
	})
	defer d.Close()

	d.Dispatch(Event{
		Event:           "block",
		RequestID:       "req-test-123",
		TenantID:        "tenant-acme",
		Action:          "block",
		BlockedEntities: []string{"API_KEY", "SECRET"},
		Model:           "gpt-4o",
		ClientIP:        "127.0.0.1",
		Details:         "Violated DLP policy",
	})

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for webhook delivery")
	}

	mu.Lock()
	defer mu.Unlock()
	if receivedEvent.Event != "block" {
		t.Errorf("expected event 'block', got '%s'", receivedEvent.Event)
	}
	if receivedEvent.Content == "" || receivedEvent.Text == "" {
		t.Errorf("expected platform-friendly Content and Text fields to be populated")
	}
	if len(receivedEvent.BlockedEntities) != 2 {
		t.Errorf("expected 2 blocked entities, got %d", len(receivedEvent.BlockedEntities))
	}
}

func TestDispatcher_EventFiltering(t *testing.T) {
	d := NewDispatcher(Config{
		URL: "http://example.com/webhook",
		Events: map[string]bool{
			"block": true,
		},
	})
	defer d.Close()

	if !d.ShouldDispatch("block") {
		t.Errorf("expected 'block' to be enabled")
	}
	if d.ShouldDispatch("rate_limit") {
		t.Errorf("expected 'rate_limit' to be filtered out")
	}
}

func TestDispatcher_TestDispatchTarget_Validation(t *testing.T) {
	d := NewDispatcher(Config{})
	defer d.Close()

	// 1. Invalid URL scheme (such as an autofilled email)
	err := d.TestDispatchTarget("trainer@fitflow.com", "", Event{Event: "test_ping"})
	if err == nil {
		t.Fatal("expected error for non-http URL, got nil")
	}

	// 2. Empty URL
	err = d.TestDispatchTarget("", "", Event{Event: "test_ping"})
	if err == nil {
		t.Fatal("expected error for empty URL, got nil")
	}

	// 3. Valid target
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err = d.TestDispatchTarget(server.URL, "secret123", Event{Event: "test_ping"})
	if err != nil {
		t.Fatalf("expected successful dispatch, got %v", err)
	}
}

// TestDispatcher_UpdateConfigWhileDelivering exercises the hot-reload path
// against in-flight deliveries: UpdateConfig rewrites the shared HTTP client
// timeout while the worker goroutine and synchronous test dispatches use it.
func TestDispatcher_UpdateConfigWhileDelivering(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	d := NewDispatcher(Config{URL: server.URL, Secret: "s", Events: map[string]bool{"*": true}})
	defer d.Close()

	stop := make(chan struct{})
	var wg sync.WaitGroup

	// Concurrent config reloads, the shape used by PUT /api/config and the GUI.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			timeout := time.Duration(2+i%3) * time.Second
			d.UpdateConfig(Config{URL: server.URL, Secret: "s", Timeout: timeout})
		}
	}()

	// Concurrent deliveries through the shared client.
	for worker := 0; worker < 4; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				if err := d.TestDispatchTarget(server.URL, "", Event{Event: "test_ping"}); err != nil {
					t.Errorf("dispatch failed during reload: %v", err)
					return
				}
			}
		}()
	}

	time.Sleep(500 * time.Millisecond)
	close(stop)
	wg.Wait()
}
