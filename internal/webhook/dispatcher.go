package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Event represents an immutable, Zero-PII security alert sent to remote endpoints.
type Event struct {
	Event           string    `json:"event"`
	Timestamp       time.Time `json:"timestamp"`
	RequestID       string    `json:"request_id,omitempty"`
	TenantID        string    `json:"tenant_id,omitempty"`
	Action          string    `json:"action,omitempty"`
	BlockedEntities []string  `json:"blocked_entities,omitempty"`
	Model           string    `json:"model,omitempty"`
	ClientIP        string    `json:"client_ip,omitempty"`
	Details         string    `json:"details,omitempty"`

	// Compatibility fields for Slack, Microsoft Teams, and Discord
	Content string `json:"content,omitempty"` // Discord message body
	Text    string `json:"text,omitempty"`    // Slack / Teams message body
}

// Config specifies webhook dispatch parameters.
type Config struct {
	URL     string
	Secret  string
	Events  map[string]bool
	Timeout time.Duration
}

// Dispatcher coordinates non-blocking asynchronous event delivery to external webhooks.
type Dispatcher struct {
	mu         sync.RWMutex
	cfg        Config
	client     *http.Client
	eventQueue chan Event
	wg         sync.WaitGroup
	stopChan   chan struct{}
	stopOnce   sync.Once
	workerOnce sync.Once
}

// NewDispatcher creates and starts an asynchronous webhook worker.
func NewDispatcher(cfg Config) *Dispatcher {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	d := &Dispatcher{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		eventQueue: make(chan Event, 1024),
		stopChan:   make(chan struct{}),
	}

	if cfg.URL != "" {
		d.startWorker()
	}

	return d
}

func (d *Dispatcher) startWorker() {
	d.workerOnce.Do(func() {
		d.wg.Add(1)
		go d.worker()
	})
}

// UpdateConfig atomically updates the webhook configuration and ensures the worker is running if a URL is provided.
func (d *Dispatcher) UpdateConfig(cfg Config) {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	d.cfg = cfg
	d.client.Timeout = cfg.Timeout

	if cfg.URL != "" {
		d.startWorker()
	}
}

// Enabled returns true if a destination URL is configured.
func (d *Dispatcher) Enabled() bool {
	if d == nil {
		return false
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.cfg.URL != ""
}

// ShouldDispatch checks whether a given event type is active.
func (d *Dispatcher) ShouldDispatch(eventType string) bool {
	if d == nil {
		return false
	}
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.cfg.URL == "" {
		return false
	}
	if len(d.cfg.Events) == 0 {
		return true
	}
	return d.cfg.Events[eventType] || d.cfg.Events["*"]
}

// TestDispatch sends a synchronous test alert to verify connectivity.
func (d *Dispatcher) TestDispatch(evt Event) error {
	if d == nil {
		return fmt.Errorf("webhook dispatcher is nil")
	}
	d.mu.RLock()
	url := d.cfg.URL
	secret := d.cfg.Secret
	d.mu.RUnlock()

	if url == "" {
		return fmt.Errorf("no webhook URL configured")
	}

	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}
	if evt.Content == "" {
		evt.Content = "🔔 [HADESGATE TEST] Ping de teste de webhook recebido com sucesso!"
	}
	if evt.Text == "" {
		evt.Text = "🔔 [HADESGATE TEST] Ping de teste de webhook recebido com sucesso!"
	}

	payloadBytes, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshaling test payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("creating HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "HadesGate-Security-Webhook/2.0")

	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(payloadBytes)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-HadesGate-Signature", "sha256="+sig)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("connecting to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("remote server returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// Dispatch non-blockingly queues an alert event for delivery.
func (d *Dispatcher) Dispatch(evt Event) {
	if !d.ShouldDispatch(evt.Event) {
		return
	}
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}

	summary := fmt.Sprintf("⚠️ [HADESGATE ALERT] %s", strings.ToUpper(evt.Event))
	if len(evt.BlockedEntities) > 0 {
		summary += fmt.Sprintf(" | Violations: %s", strings.Join(evt.BlockedEntities, ", "))
	}
	if evt.Model != "" {
		summary += fmt.Sprintf(" | Model: %s", evt.Model)
	}
	if evt.TenantID != "" {
		summary += fmt.Sprintf(" | Tenant: %s", evt.TenantID)
	}
	if evt.Details != "" {
		summary += fmt.Sprintf(" | %s", evt.Details)
	}

	evt.Content = summary
	evt.Text = summary

	select {
	case d.eventQueue <- evt:
	default:
		log.Printf("⚠️  [WEBHOOK] Event queue full, dropping alert for event=%s req=%s\n", evt.Event, evt.RequestID)
	}
}

func (d *Dispatcher) worker() {
	defer d.wg.Done()
	for {
		select {
		case evt, ok := <-d.eventQueue:
			if !ok {
				return
			}
			d.send(evt)
		case <-d.stopChan:
			for {
				select {
				case evt := <-d.eventQueue:
					d.send(evt)
				default:
					return
				}
			}
		}
	}
}

func (d *Dispatcher) send(evt Event) {
	payloadBytes, err := json.Marshal(evt)
	if err != nil {
		log.Printf("⚠️  [WEBHOOK] Failed to marshal event: %v\n", err)
		return
	}

	d.mu.RLock()
	url := d.cfg.URL
	secret := d.cfg.Secret
	d.mu.RUnlock()

	if url == "" {
		return
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payloadBytes))
	if err != nil {
		log.Printf("⚠️  [WEBHOOK] Failed to build request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "HadesGate-Security-Webhook/2.0")

	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(payloadBytes)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-HadesGate-Signature", "sha256="+sig)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		log.Printf("⚠️  [WEBHOOK] Failed to deliver webhook to %s: %v\n", url, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("⚠️  [WEBHOOK] Non-2xx response status %d for event=%s\n", resp.StatusCode, evt.Event)
	}
}

// Close gracefully flushes pending events and stops the background worker.
func (d *Dispatcher) Close() {
	if d == nil {
		return
	}
	d.stopOnce.Do(func() {
		close(d.stopChan)
		d.wg.Wait()
	})
}

