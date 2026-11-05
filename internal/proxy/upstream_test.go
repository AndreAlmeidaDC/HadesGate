package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hadesgate/hadesgate/internal/anonymizer"
	"github.com/hadesgate/hadesgate/internal/anonymizer/vault"
	privatestream "github.com/hadesgate/hadesgate/internal/stream"
)

func TestResolveTarget(t *testing.T) {
	tests := []struct {
		name     string
		cfg      UpstreamConfig
		headers  map[string]string
		model    string
		wantURL  string
		wantKey  string
		wantMock bool
		wantErr  bool
	}{
		{
			name: "default openai without key falls back to mock",
			cfg: UpstreamConfig{
				OpenAIBaseURL: "https://api.openai.com/v1",
			},
			model:    "gpt-4o",
			wantMock: true,
		},
		{
			name: "explicit mock model returns mock",
			cfg: UpstreamConfig{
				OpenAIBaseURL: "https://api.openai.com/v1",
				OpenAIKey:     "sk-test",
			},
			model:    "mock",
			wantMock: true,
		},
		{
			name: "custom router base url (OmniRoute) without key does not fall back to mock",
			cfg: UpstreamConfig{
				OpenAIBaseURL: "http://host.docker.internal:20128/v1",
			},
			model:    "claude-3-5-sonnet",
			wantURL:  "http://host.docker.internal:20128/v1/chat/completions",
			wantKey:  "",
			wantMock: false,
		},
		{
			name: "custom router with env api key uses configured key",
			cfg: UpstreamConfig{
				OpenAIBaseURL: "http://omniroute:20128/v1",
				OpenAIKey:     "omni-secret-key",
			},
			model:    "gpt-4o",
			wantURL:  "http://omniroute:20128/v1/chat/completions",
			wantKey:  "Bearer omni-secret-key",
			wantMock: false,
		},
		{
			name: "admin key in authorization header is not leaked upstream",
			cfg: UpstreamConfig{
				OpenAIBaseURL: "http://omniroute:20128/v1",
				AdminKey:      "privagate-admin-123",
				OpenAIKey:     "upstream-llm-key",
			},
			headers: map[string]string{
				"Authorization": "Bearer privagate-admin-123",
			},
			model:    "deepseek-chat",
			wantURL:  "http://omniroute:20128/v1/chat/completions",
			wantKey:  "Bearer upstream-llm-key",
			wantMock: false,
		},
		{
			name: "dynamic X-Upstream-Url and X-Upstream-Key override target",
			cfg: UpstreamConfig{
				OpenAIBaseURL:               "https://api.openai.com/v1",
				OpenAIKey:                   "default-key",
				AllowDynamicUpstream:        true,
				DynamicUpstreamAllowedHosts: []string{"localhost:9000"},
			},
			headers: map[string]string{
				"X-Upstream-Url": "http://localhost:9000/v1",
				"X-Upstream-Key": "my-router-key",
			},
			model:    "llama-3",
			wantURL:  "http://localhost:9000/v1/chat/completions",
			wantKey:  "Bearer my-router-key",
			wantMock: false,
		},
		{
			name:    "dynamic upstream is disabled by default",
			cfg:     UpstreamConfig{OpenAIKey: "default-key"},
			headers: map[string]string{"X-Upstream-Url": "http://localhost:9000/v1"},
			model:   "llama-3",
			wantErr: true,
		},
		{
			name: "dynamic upstream rejects host outside allowlist",
			cfg: UpstreamConfig{
				OpenAIKey:                   "default-key",
				AllowDynamicUpstream:        true,
				DynamicUpstreamAllowedHosts: []string{"router.internal:9000"},
			},
			headers: map[string]string{"X-Upstream-Url": "http://169.254.169.254/latest"},
			model:   "llama-3",
			wantErr: true,
		},
		{
			name: "dynamic upstream requires exact port match",
			cfg: UpstreamConfig{
				OpenAIKey:                   "default-key",
				AllowDynamicUpstream:        true,
				DynamicUpstreamAllowedHosts: []string{"localhost:9000"},
			},
			headers: map[string]string{"X-Upstream-Url": "http://localhost:9001/v1"},
			model:   "llama-3",
			wantErr: true,
		},
		{
			name: "ollama prefix routes to ollama base url",
			cfg: UpstreamConfig{
				OllamaBaseURL: "http://localhost:11434/v1",
			},
			model:    "ollama/llama3",
			wantURL:  "http://localhost:11434/v1/chat/completions",
			wantKey:  "",
			wantMock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "dynamic X-Upstream-Url and X-Upstream-Key override target" {
				setOptionalBoolField(t, &tt.cfg, "AllowPrivateDynamicUpstream", true)
			}
			disp := NewUpstreamDispatcher(tt.cfg)
			req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			targetURL, apiKey, isMock, err := disp.ResolveTarget(req, tt.model)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ResolveTarget() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			if isMock != tt.wantMock {
				t.Errorf("ResolveTarget() isMock = %v, want %v", isMock, tt.wantMock)
			}
			if !tt.wantMock {
				if targetURL != tt.wantURL {
					t.Errorf("ResolveTarget() targetURL = %q, want %q", targetURL, tt.wantURL)
				}
				if apiKey != tt.wantKey {
					t.Errorf("ResolveTarget() apiKey = %q, want %q", apiKey, tt.wantKey)
				}
			}
		})
	}
}

func TestUpstreamDispatcher_RuntimeConfigUpdatesAreAtomic(t *testing.T) {
	dispatcher := NewUpstreamDispatcher(UpstreamConfig{OllamaBaseURL: "http://localhost:11434/v1"})
	var wg sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for iteration := 0; iteration < 500; iteration++ {
				if id%2 == 0 {
					dispatcher.SetEnableEgressScan(iteration%2 == 0)
					dispatcher.SetOllamaBaseURL(fmt.Sprintf("http://127.0.0.1:%d/v1", 11434+iteration%2))
					continue
				}
				target := dispatcher.ResolveLocalTarget("/chat/completions")
				if !strings.HasSuffix(target, "/v1/chat/completions") {
					t.Errorf("observed partial runtime configuration: %q", target)
					return
				}
			}
		}(worker)
	}
	wg.Wait()
}

func TestResolveTargetRejectsAllowlistedPrivateAddressByDefault(t *testing.T) {
	dispatcher := NewUpstreamDispatcher(UpstreamConfig{
		OpenAIKey:                   "default-key",
		AllowDynamicUpstream:        true,
		DynamicUpstreamAllowedHosts: []string{"169.254.169.254"},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req.Header.Set("X-Upstream-Url", "http://169.254.169.254/latest")

	if _, _, _, err := dispatcher.ResolveTarget(req, "gpt-test"); err == nil {
		t.Fatal("allowlisted link-local metadata address bypassed SSRF protection")
	}
}

func TestResolveTargetRejectsAllowlistedLoopbackByDefault(t *testing.T) {
	dispatcher := NewUpstreamDispatcher(UpstreamConfig{
		OpenAIKey:                   "default-key",
		AllowDynamicUpstream:        true,
		DynamicUpstreamAllowedHosts: []string{"localhost:8443"},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req.Header.Set("X-Upstream-Url", "https://localhost:8443/v1")

	if _, _, _, err := dispatcher.ResolveTarget(req, "gpt-test"); err == nil {
		t.Fatal("allowlisted loopback host bypassed private-network opt-in")
	}
}

func setOptionalBoolField(t *testing.T, target interface{}, name string, value bool) {
	t.Helper()
	field := reflect.ValueOf(target).Elem().FieldByName(name)
	if field.IsValid() && field.CanSet() && field.Kind() == reflect.Bool {
		field.SetBool(value)
	}
}

func TestReadLimitedBody(t *testing.T) {
	body, err := readLimitedBody(strings.NewReader("12345678"), 8)
	if err != nil || string(body) != "12345678" {
		t.Fatalf("exact limit failed: body=%q err=%v", body, err)
	}

	if _, err := readLimitedBody(strings.NewReader("123456789"), 8); err == nil {
		t.Fatal("expected oversized response to be rejected")
	}
}

func TestStreamingRejectsOversizedEvent(t *testing.T) {
	dispatcher := NewUpstreamDispatcher(UpstreamConfig{MaxSSEEventBytes: 64})
	v := vault.NewMemoryVault(time.Minute)
	defer v.Close()
	detok := privatestream.NewDetokenizer("stream-limit", v)
	w := httptest.NewRecorder()

	err := dispatcher.HandleStreamingResponse(strings.NewReader(strings.Repeat("x", 65)), detok, w)
	if err == nil {
		t.Fatal("expected oversized SSE event to be rejected")
	}
}

func TestFetchModels(t *testing.T) {
	mockResponse := `{"object":"list","data":[{"id":"gpt-4o","object":"model"}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer srv.Close()

	disp := NewUpstreamDispatcher(UpstreamConfig{
		OpenAIBaseURL: srv.URL,
		Timeout:       2 * time.Second,
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	data, err := disp.FetchModels(context.Background(), req)
	if err != nil {
		t.Fatalf("FetchModels() unexpected error: %v", err)
	}

	if string(data) != mockResponse {
		t.Errorf("FetchModels() got %s, want %s", string(data), mockResponse)
	}
}

func TestHandleNonStreamingResponse_EgressScanning(t *testing.T) {
	v := vault.NewMemoryVault(time.Minute)
	defer v.Close()

	sessionID := "sess-egress-test"
	token := "[HG_user123_CPF_1]"
	v.Put(sessionID, token, "123.456.789-00", "CPF", time.Minute)

	eng := anonymizer.NewEngine(anonymizer.DefaultConfig(), v)
	disp := NewUpstreamDispatcher(UpstreamConfig{
		EnableEgressScan: true,
		Engine:           eng,
		MaxResponseBytes: 1024 * 1024,
	})

	detok := privatestream.NewDetokenizer(sessionID, v)

	// Upstream returns restored CPF (allowed because it was in vault),
	// but also a leaked CPF (not in vault) that LLM produced/hallucinated.
	upstreamJSON := `{"choices":[{"message":{"content":"User CPF is [HG_user123_CPF_1], but leaked CPF is 529.982.247-25"}}]}`

	rec := httptest.NewRecorder()
	err := disp.HandleNonStreamingResponse(strings.NewReader(upstreamJSON), detok, rec)
	if err != nil {
		t.Fatalf("HandleNonStreamingResponse failed: %v", err)
	}

	resBody := rec.Body.String()
	// Vault CPF should be restored to 123.456.789-00
	if !strings.Contains(resBody, "123.456.789-00") {
		t.Errorf("expected user CPF to be restored, got: %s", resBody)
	}

	// Leaked CPF (529.982.247-25) should NOT remain plaintext; it should be masked with [HG_EGRESS_CPF_1]
	if strings.Contains(resBody, "529.982.247-25") {
		t.Errorf("expected leaked CPF to be masked by egress scanner, but found plaintext in: %s", resBody)
	}
	if !strings.Contains(resBody, "[HG_EGRESS_CPF_1]") {
		t.Errorf("expected [HG_EGRESS_CPF_1] token in response, got: %s", resBody)
	}

	// Header X-HadesGate-Egress-Detected should be set
	if rec.Header().Get("X-HadesGate-Egress-Detected") != "true" {
		t.Errorf("expected X-HadesGate-Egress-Detected header to be 'true', got: '%s'", rec.Header().Get("X-HadesGate-Egress-Detected"))
	}
}

func TestHandleStreamingResponse_EgressScanningAcrossChunksAndToolArguments(t *testing.T) {
	v := vault.NewMemoryVault(time.Minute)
	defer v.Close()

	eng := anonymizer.NewEngine(anonymizer.DefaultConfig(), v)
	disp := NewUpstreamDispatcher(UpstreamConfig{
		EnableEgressScan: true,
		Engine:           eng,
		MaxSSEEventBytes: 1024 * 1024,
	})
	detok := privatestream.NewDetokenizer("sess-egress-stream", v)

	upstreamSSE := strings.Join([]string{
		`data: {"id":"chunk-1","model":"gpt-test","choices":[{"index":0,"delta":{"content":"CPF vazado 529.982."}}]}`,
		`data: {"id":"chunk-1","model":"gpt-test","choices":[{"index":0,"delta":{"content":"247-25"}}]}`,
		`data: {"id":"chunk-1","model":"gpt-test","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"cpf\":\"529.982."}}]}}]}`,
		`data: {"id":"chunk-1","model":"gpt-test","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"247-25\"}"}}]}}]}`,
		`data: [DONE]`,
		"",
	}, "\n")

	rec := httptest.NewRecorder()
	if err := disp.HandleStreamingResponse(strings.NewReader(upstreamSSE), detok, rec); err != nil {
		t.Fatalf("HandleStreamingResponse failed: %v", err)
	}

	body := rec.Body.String()
	if strings.Contains(body, "529.982.247-25") {
		t.Fatalf("streaming egress DLP leaked a fragmented CPF: %s", body)
	}
	if count := strings.Count(body, "[HG_EGRESS_CPF_1]"); count != 2 {
		t.Fatalf("expected content and tool arguments to be masked, got %d masks in: %s", count, body)
	}
	if rec.Header().Get("X-HadesGate-Egress-Detected") != "true" {
		t.Fatalf("expected streaming egress detection trailer/header, got %q", rec.Header().Get("X-HadesGate-Egress-Detected"))
	}
}

func TestHandleGenericStreamingResponse_EgressScanningAcrossAnthropicEvents(t *testing.T) {
	v := vault.NewMemoryVault(time.Minute)
	defer v.Close()

	eng := anonymizer.NewEngine(anonymizer.DefaultConfig(), v)
	disp := NewUpstreamDispatcher(UpstreamConfig{
		EnableEgressScan: true,
		Engine:           eng,
		MaxSSEEventBytes: 1024 * 1024,
	})
	detok := privatestream.NewDetokenizer("sess-anthropic-egress", v)
	upstreamSSE := strings.Join([]string{
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"CPF 529.982."}}`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"247-25"}}`,
		`data: [DONE]`,
		"",
	}, "\n")

	rec := httptest.NewRecorder()
	if err := disp.HandleGenericStreamingResponse(strings.NewReader(upstreamSSE), detok, rec, "anthropic"); err != nil {
		t.Fatalf("HandleGenericStreamingResponse failed: %v", err)
	}
	if body := rec.Body.String(); strings.Contains(body, "529.982.247-25") || !strings.Contains(body, "[HG_EGRESS_CPF_1]") {
		t.Fatalf("Anthropic stream was not protected: %s", body)
	}
}

func TestHandleStreamingResponse_FlushesEgressWhenDoneSentinelIsMissing(t *testing.T) {
	v := vault.NewMemoryVault(time.Minute)
	defer v.Close()
	eng := anonymizer.NewEngine(anonymizer.DefaultConfig(), v)
	disp := NewUpstreamDispatcher(UpstreamConfig{EnableEgressScan: true, Engine: eng})
	detok := privatestream.NewDetokenizer("sess-eof-egress", v)
	upstreamSSE := "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"CPF 529.982.247-25\"}}]}\n\n"

	rec := httptest.NewRecorder()
	if err := disp.HandleStreamingResponse(strings.NewReader(upstreamSSE), detok, rec); err != nil {
		t.Fatalf("HandleStreamingResponse failed: %v", err)
	}
	if body := rec.Body.String(); strings.Contains(body, "529.982.247-25") || !strings.Contains(body, "[HG_EGRESS_CPF_1]") {
		t.Fatalf("EOF flush did not protect buffered stream content: %s", body)
	}
}

func TestHandleStreamingResponse_EgressAllowsPlaintextRestoredFromVault(t *testing.T) {
	v := vault.NewMemoryVault(time.Minute)
	defer v.Close()
	const sessionID = "sess-known-egress"
	v.Put(sessionID, "[HG_known_CPF_1]", "123.456.789-00", "CPF", time.Minute)

	eng := anonymizer.NewEngine(anonymizer.DefaultConfig(), v)
	disp := NewUpstreamDispatcher(UpstreamConfig{EnableEgressScan: true, Engine: eng})
	detok := privatestream.NewDetokenizer(sessionID, v)
	upstreamSSE := strings.Join([]string{
		`data: {"choices":[{"index":0,"delta":{"content":"CPF original [HG_known_"}}]}`,
		`data: {"choices":[{"index":0,"delta":{"content":"CPF_1]"}}]}`,
		`data: [DONE]`,
		"",
	}, "\n")

	rec := httptest.NewRecorder()
	if err := disp.HandleStreamingResponse(strings.NewReader(upstreamSSE), detok, rec); err != nil {
		t.Fatalf("HandleStreamingResponse failed: %v", err)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "123.456.789-00") || strings.Contains(body, "HG_EGRESS_CPF") {
		t.Fatalf("egress DLP masked plaintext legitimately restored from the request vault: %s", body)
	}
}

func TestUpstreamDispatcherBYOKNativeProviderResolution(t *testing.T) {
	dispatcher := NewUpstreamDispatcher(UpstreamConfig{
		AnthropicURL: "https://api.anthropic.com",
		GeminiURL:    "https://generativelanguage.googleapis.com",
		AdminKey:     "hadesgate-proxy-key",
	})

	anthropicURL, anthropicKey, err := dispatcher.ResolveAnthropicTargetForKey("sk-ant-client-key")
	if err != nil {
		t.Fatalf("resolving Anthropic BYOK target: %v", err)
	}
	if anthropicURL != "https://api.anthropic.com/v1/messages" || anthropicKey != "sk-ant-client-key" {
		t.Fatalf("unexpected Anthropic BYOK target: url=%q key=%q", anthropicURL, anthropicKey)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-flash:generateContent", nil)
	req.Header.Set("X-Upstream-Key", "gemini-client-key")
	geminiURL, geminiKey, err := dispatcher.ResolveGeminiTarget(req, "gemini-2.5-flash", "generateContent", false)
	if err != nil {
		t.Fatalf("resolving Gemini BYOK target: %v", err)
	}
	if !strings.Contains(geminiURL, "/v1beta/models/gemini-2.5-flash:generateContent") || geminiKey != "gemini-client-key" {
		t.Fatalf("unexpected Gemini BYOK target: url=%q key=%q", geminiURL, geminiKey)
	}
}
