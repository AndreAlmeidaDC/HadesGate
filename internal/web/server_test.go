package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/hadesgate/hadesgate/internal/anonymizer"
	"github.com/hadesgate/hadesgate/internal/anonymizer/detector"
	"github.com/hadesgate/hadesgate/internal/anonymizer/vault"
	"github.com/hadesgate/hadesgate/internal/config"
	"github.com/hadesgate/hadesgate/internal/observability"
	"github.com/hadesgate/hadesgate/internal/proxy"
	"github.com/hadesgate/hadesgate/internal/webhook"
)

func setupTestServer(adminKey string) (*Server, *http.ServeMux) {
	v := vault.NewMemoryVault(5 * time.Minute)
	eng := anonymizer.NewEngine(anonymizer.DefaultConfig(), v)
	traces := observability.NewTraceLogger(100)
	wh := webhook.NewDispatcher(webhook.Config{})
	gw := proxy.NewGatewayWithOptions(eng, nil, traces, proxy.GatewayOptions{
		Webhook: wh,
	})
	srv := NewServer(eng, gw, adminKey, nil)
	store, _ := config.NewStore(traces.DB(), []byte("0123456789abcdef0123456789abcdef"))
	srv.SetStore(store)
	srv.SetWebhook(wh)

	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)
	return srv, mux
}

func TestServer_AuthVerify(t *testing.T) {
	const secretKey = "test_secret_key_123"
	_, mux := setupTestServer(secretKey)

	tests := []struct {
		name       string
		payload    string
		wantStatus int
		wantValid  bool
	}{
		{
			name:       "Valid API Key",
			payload:    `{"key":"test_secret_key_123"}`,
			wantStatus: http.StatusOK,
			wantValid:  true,
		},
		{
			name:       "Invalid API Key",
			payload:    `{"key":"wrong_key"}`,
			wantStatus: http.StatusOK,
			wantValid:  false,
		},
		{
			name:       "Empty API Key",
			payload:    `{"key":""}`,
			wantStatus: http.StatusOK,
			wantValid:  false,
		},
		{
			name:       "Invalid JSON",
			payload:    `{invalid-json}`,
			wantStatus: http.StatusBadRequest,
			wantValid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/auth/verify", bytes.NewBufferString(tt.payload))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}

			if w.Code == http.StatusOK {
				var resp AuthVerifyResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if resp.Valid != tt.wantValid {
					t.Errorf("expected Valid=%v, got %v", tt.wantValid, resp.Valid)
				}
			}
		})
	}
}

func TestServer_AccountlessAdministrativeRoleMatrix(t *testing.T) {
	const bootstrapKey = "bootstrap-admin-key"
	server, mux := setupTestServer(bootstrapKey)
	viewer, err := server.store.CreateAdminCredential(context.Background(), "Read only", config.RoleViewer)
	if err != nil {
		t.Fatal(err)
	}
	operator, err := server.store.CreateAdminCredential(context.Background(), "Operations", config.RoleOperator)
	if err != nil {
		t.Fatal(err)
	}

	request := func(method, path, key, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+key)
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		return response
	}

	if response := request(http.MethodGet, "/api/traces", viewer.Secret, ""); response.Code != http.StatusOK {
		t.Fatalf("viewer could not read traces: %d %s", response.Code, response.Body.String())
	}
	if response := request(http.MethodDelete, "/api/traces", viewer.Secret, ""); response.Code != http.StatusForbidden {
		t.Fatalf("viewer mutated traces: %d", response.Code)
	}
	if response := request(http.MethodDelete, "/api/traces", operator.Secret, ""); response.Code != http.StatusOK {
		t.Fatalf("operator could not clear traces: %d %s", response.Code, response.Body.String())
	}
	if response := request(http.MethodGet, "/api/admin-credentials", operator.Secret, ""); response.Code != http.StatusForbidden {
		t.Fatalf("operator listed admin credentials: %d", response.Code)
	}
	if response := request(http.MethodGet, "/api/admin-credentials", bootstrapKey, ""); response.Code != http.StatusOK {
		t.Fatalf("bootstrap admin could not list credentials: %d", response.Code)
	}

	verify := httptest.NewRequest(http.MethodPost, "/api/auth/verify", strings.NewReader(`{"key":"`+viewer.Secret+`"}`))
	verifyResponse := httptest.NewRecorder()
	mux.ServeHTTP(verifyResponse, verify)
	var auth AuthVerifyResponse
	if err := json.NewDecoder(verifyResponse.Body).Decode(&auth); err != nil {
		t.Fatal(err)
	}
	if !auth.Valid || auth.Role != config.RoleViewer || auth.KeyID != viewer.KeyID {
		t.Fatalf("role was not returned during authentication: %+v", auth)
	}
}

func TestServer_GovernanceEndpointsCreateSignedAuditAndEncryptedBackup(t *testing.T) {
	const bootstrapKey = "bootstrap-admin-key"
	_, mux := setupTestServer(bootstrapKey)
	authorized := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+bootstrapKey)
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		return response
	}
	created := authorized(http.MethodPost, "/api/admin-credentials", `{"label":"Auditoria","role":"viewer"}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("credential creation failed: %d %s", created.Code, created.Body.String())
	}
	var credential config.AdminCredential
	if err := json.NewDecoder(created.Body).Decode(&credential); err != nil || credential.Secret == "" {
		t.Fatalf("one-time secret missing: %+v %v", credential, err)
	}
	audit := authorized(http.MethodGet, "/api/audit/events", "")
	var exported struct {
		Events []config.AuditEvent `json:"events"`
	}
	if err := json.NewDecoder(audit.Body).Decode(&exported); err != nil || len(exported.Events) == 0 || exported.Events[0].Signature == "" || exported.Events[0].Nonce == "" {
		t.Fatalf("signed audit export invalid: %s", audit.Body.String())
	}
	backup := authorized(http.MethodGet, "/api/config-backup", "")
	if backup.Code != http.StatusOK || strings.Contains(backup.Body.String(), credential.Secret) {
		t.Fatalf("encrypted backup invalid or leaked secret: %d %s", backup.Code, backup.Body.String())
	}
}

func TestServer_ProtectedEndpoints(t *testing.T) {
	const secretKey = "test_secret_key_123"
	_, mux := setupTestServer(secretKey)

	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodGet, path: "/api/traces", body: ""},
		{method: http.MethodGet, path: "/api/vault", body: ""},
		{method: http.MethodPost, path: "/api/test-anonymize", body: `{"prompt":"test"}`},
	}

	for _, ep := range endpoints {
		t.Run("NoAuth_"+ep.path, func(t *testing.T) {
			req := httptest.NewRequest(ep.method, ep.path, bytes.NewBufferString(ep.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Errorf("expected 401 Unauthorized for %s, got %d", ep.path, w.Code)
			}
		})

		t.Run("InvalidAuth_"+ep.path, func(t *testing.T) {
			req := httptest.NewRequest(ep.method, ep.path, bytes.NewBufferString(ep.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer wrong_token")
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Errorf("expected 401 Unauthorized for %s with bad token, got %d", ep.path, w.Code)
			}
		})

		t.Run("ValidBearer_"+ep.path, func(t *testing.T) {
			req := httptest.NewRequest(ep.method, ep.path, bytes.NewBufferString(ep.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+secretKey)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("expected 200 OK for %s with valid bearer, got %d", ep.path, w.Code)
			}
		})

		t.Run("ValidXAPIKey_"+ep.path, func(t *testing.T) {
			req := httptest.NewRequest(ep.method, ep.path, bytes.NewBufferString(ep.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-API-Key", secretKey)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("expected 200 OK for %s with valid X-API-Key, got %d", ep.path, w.Code)
			}
		})
	}
}

func TestServer_GlobalProxyKeyCannotAccessAdminEndpoints(t *testing.T) {
	const (
		adminKey       = "admin-secret-key"
		globalProxyKey = "transparent-proxy-key"
	)

	v := vault.NewMemoryVault(5 * time.Minute)
	t.Cleanup(v.Close)
	eng := anonymizer.NewEngine(anonymizer.DefaultConfig(), v)
	traces := observability.NewTraceLogger(100)
	t.Cleanup(func() { _ = traces.Close() })
	gw := proxy.NewGatewayWithOptions(eng, nil, traces, proxy.GatewayOptions{
		GlobalProxyKey: globalProxyKey,
	})
	srv := NewServer(eng, gw, adminKey, nil)
	store, err := config.NewStore(traces.DB(), []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("creating config store: %v", err)
	}
	srv.SetStore(store)

	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	for _, endpoint := range []string{"/api/config", "/api/tenants"} {
		req := httptest.NewRequest(http.MethodGet, endpoint, nil)
		req.Header.Set("Authorization", "Bearer "+globalProxyKey)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("proxy key accessed admin endpoint %s: got status %d, want %d", endpoint, w.Code, http.StatusUnauthorized)
		}
	}
}

func TestServer_ProxyTenantAndProviderKeysCannotAuthenticateAdminConsole(t *testing.T) {
	const (
		adminKey       = "admin-console-secret"
		globalProxyKey = "global-proxy-secret"
		tenantKey      = "tenant-proxy-secret"
		providerKey    = "sk-provider-secret"
		fingerprintKey = "fingerprint-secret"
	)

	v := vault.NewMemoryVault(5 * time.Minute)
	t.Cleanup(v.Close)
	eng := anonymizer.NewEngine(anonymizer.DefaultConfig(), v)
	traces := observability.NewTraceLogger(100)
	t.Cleanup(func() { _ = traces.Close() })
	fingerprint, err := proxy.ProviderKeyFingerprint(fingerprintKey, providerKey)
	if err != nil {
		t.Fatalf("fingerprinting provider key: %v", err)
	}
	gw := proxy.NewGatewayWithOptions(eng, nil, traces, proxy.GatewayOptions{
		GlobalProxyKey:            globalProxyKey,
		TenantKeys:                map[string]string{"finance": tenantKey},
		ProviderFingerprintSecret: fingerprintKey,
		ProviderKeyFingerprints:   map[string][]string{"finance": {fingerprint}},
	})
	srv := NewServer(eng, gw, adminKey, nil)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	for name, key := range map[string]string{
		"global proxy":  globalProxyKey,
		"tenant":        tenantKey,
		"provider BYOK": providerKey,
	} {
		t.Run(name, func(t *testing.T) {
			body, _ := json.Marshal(AuthVerifyRequest{Key: key})
			verifyReq := httptest.NewRequest(http.MethodPost, "/api/auth/verify", bytes.NewReader(body))
			verifyW := httptest.NewRecorder()
			mux.ServeHTTP(verifyW, verifyReq)
			var verifyResp AuthVerifyResponse
			if err := json.NewDecoder(verifyW.Body).Decode(&verifyResp); err != nil {
				t.Fatalf("decoding auth response: %v", err)
			}
			if verifyResp.Valid {
				t.Fatalf("%s key authenticated the admin console", name)
			}

			for _, endpoint := range []string{"/api/traces", "/api/vault", "/api/test-anonymize"} {
				req := httptest.NewRequest(http.MethodGet, endpoint, nil)
				req.Header.Set("Authorization", "Bearer "+key)
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, req)
				if w.Code != http.StatusUnauthorized {
					t.Fatalf("%s key accessed %s: got %d, want 401", name, endpoint, w.Code)
				}
			}
		})
	}
}

func TestServer_AdminCanFilterAndPurgeTenantTraces(t *testing.T) {
	const adminKey = "admin-secret-key"
	srv, mux := setupTestServer(adminKey)
	srv.gateway.UpdateDynamicConfig(nil,
		map[string]string{"tenant-a": "tenant-a-key", "tenant-b": "tenant-b-key"},
		nil, nil, 0, "")

	srv.gateway.Traces().Record(observability.Trace{ID: "trace-a", Tenant: "tenant-a", Timestamp: time.Now()})
	srv.gateway.Traces().Record(observability.Trace{ID: "trace-b", Tenant: "tenant-b", Timestamp: time.Now()})

	tenantReq := httptest.NewRequest(http.MethodGet, "/api/traces", nil)
	tenantReq.Header.Set("Authorization", "Bearer tenant-a-key")
	tenantW := httptest.NewRecorder()
	mux.ServeHTTP(tenantW, tenantReq)
	if tenantW.Code != http.StatusUnauthorized {
		t.Fatalf("tenant accessed admin traces: got %d, want 401", tenantW.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/traces?tenant_id=tenant-a", nil)
	req.Header.Set("Authorization", "Bearer "+adminKey)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("admin tenant filter returned %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "trace-b") || !strings.Contains(w.Body.String(), "trace-a") {
		t.Fatalf("admin tenant filter returned traces outside its scope: %s", w.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/traces?tenant_id=tenant-a", nil)
	deleteReq.Header.Set("Authorization", "Bearer "+adminKey)
	deleteW := httptest.NewRecorder()
	mux.ServeHTTP(deleteW, deleteReq)
	if deleteW.Code != http.StatusOK {
		t.Fatalf("admin tenant purge returned %d: %s", deleteW.Code, deleteW.Body.String())
	}

	remaining := srv.gateway.Traces().GetAll()
	if len(remaining) != 1 || remaining[0].ID != "trace-b" {
		t.Fatalf("tenant purge affected global telemetry: %+v", remaining)
	}
}

func TestServer_StatusEndpoint(t *testing.T) {
	const secretKey = "test_secret_key_123"
	_, mux := setupTestServer(secretKey)

	// 1. Unauthenticated status ping
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for unauthenticated status, got %d", w.Code)
	}

	var unauthResp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&unauthResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if unauthResp["status"] != "online" {
		t.Errorf("expected status=online, got %v", unauthResp["status"])
	}
	// Operational metrics should not be leaked in unauthenticated response
	if _, exists := unauthResp["metrics"]; exists {
		t.Errorf("metrics should not be present in unauthenticated status")
	}

	// 2. Authenticated status ping
	reqAuth := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	reqAuth.Header.Set("Authorization", "Bearer "+secretKey)
	wAuth := httptest.NewRecorder()
	mux.ServeHTTP(wAuth, reqAuth)

	if wAuth.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for authenticated status, got %d", wAuth.Code)
	}

	var authResp StatusResponse
	if err := json.NewDecoder(wAuth.Body).Decode(&authResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if authResp.Status != "online" {
		t.Errorf("expected status=online, got %v", authResp.Status)
	}
	if authResp.Version != "1.3.0" {
		t.Errorf("expected version=1.3.0, got %v", authResp.Version)
	}
}

func TestServer_StaticDoesNotInterceptProxyPrefixes(t *testing.T) {
	v := vault.NewMemoryVault(5 * time.Minute)
	eng := anonymizer.NewEngine(anonymizer.DefaultConfig(), v)
	traces := observability.NewTraceLogger(10)
	gw := proxy.NewGateway(eng, nil, "secret", traces)
	staticFS := fstest.MapFS{
		"index.html": {Data: []byte("<html>console</html>")},
	}
	srv := NewServer(eng, gw, "secret", staticFS)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	for _, path := range []string{
		"/v1/unknown",
		"/v1beta/models/gemini-2.5-flash:generateContent",
		"/gemini/v1beta/models/gemini-2.5-flash:generateContent",
		"/anthropic/v1/messages",
		"/api/unknown",
	} {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404 for proxy/api prefix %s, got %d", path, w.Code)
		}
	}
}

func TestServer_TestAnonymizeRecordsExtensionTraceZeroPII(t *testing.T) {
	const secretKey = "test_secret_key_123"
	srv, mux := setupTestServer(secretKey)

	payload := `{"prompt":"Cobrar João Silva, CPF 529.982.247-25","record_trace":true,"provider":"claude","model":"claude.ai"}`
	req := httptest.NewRequest(http.MethodPost, "/api/test-anonymize", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+secretKey)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	traces := srv.gateway.Traces().GetAll()
	if len(traces) != 1 {
		t.Fatalf("expected 1 recorded trace for browser extension, got %d", len(traces))
	}
	if traces[0].Provider != "claude" || traces[0].Model != "claude.ai" {
		t.Fatalf("unexpected trace metadata: %+v", traces[0])
	}
	if bytes.Contains([]byte(traces[0].SanitizedPrompt), []byte("529.982.247-25")) {
		t.Fatalf("raw CPF leaked into recorded trace: %s", traces[0].SanitizedPrompt)
	}
}

func TestServer_TestAnonymize_PolicyBlockAndWebhook(t *testing.T) {
	const secretKey = "test_secret_key_123"
	srv, mux := setupTestServer(secretKey)

	// Configure global policy to block CPF and a tenant key that must remain proxy-only.
	srv.gateway.UpdateDynamicConfig(
		map[string]string{"CPF": "block"},
		map[string]string{"tenant_01": "key_tenant_01"},
		map[string]map[string]string{
			"tenant_01": {"EMAIL": "block"},
		},
		nil, 0, "",
	)

	// 1. Global / admin key sending CPF -> should be blocked (403)
	payload := `{"prompt":"Contrato de João, CPF 529.982.247-25","record_trace":true,"provider":"openai","model":"chatgpt.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/test-anonymize", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+secretKey)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for blocked CPF, got %d: %s", w.Code, w.Body.String())
	}

	var blockedResp struct {
		Blocked         bool     `json:"blocked"`
		BlockedEntities []string `json:"blocked_entities"`
		Error           string   `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &blockedResp); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	if !blockedResp.Blocked {
		t.Fatalf("expected blocked=true, got %+v", blockedResp)
	}
	hasCPF := false
	for _, ent := range blockedResp.BlockedEntities {
		if ent == "CPF" {
			hasCPF = true
			break
		}
	}
	if !hasCPF {
		t.Fatalf("expected CPF in blocked_entities, got %+v", blockedResp.BlockedEntities)
	}

	// Verify trace recorded with StatusForbidden and PolicyAction block
	traces := srv.gateway.Traces().GetAll()
	if len(traces) != 1 {
		t.Fatalf("expected 1 recorded trace, got %d", len(traces))
	}
	if traces[0].PolicyAction != "block" || traces[0].StatusCode != http.StatusForbidden {
		t.Fatalf("unexpected trace status: %+v", traces[0])
	}

	// 2. Tenant keys cannot use the administrative Playground endpoint.
	payloadEmail := `{"prompt":"Contato suporte@empresa.com","record_trace":true,"provider":"anthropic","model":"claude.ai"}`
	reqTenant := httptest.NewRequest(http.MethodPost, "/api/test-anonymize", bytes.NewBufferString(payloadEmail))
	reqTenant.Header.Set("Content-Type", "application/json")
	reqTenant.Header.Set("Authorization", "Bearer key_tenant_01")
	wTenant := httptest.NewRecorder()

	mux.ServeHTTP(wTenant, reqTenant)
	if wTenant.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for tenant Playground access, got %d: %s", wTenant.Code, wTenant.Body.String())
	}

	tracesAfter := srv.gateway.Traces().GetAll()
	if len(tracesAfter) != 1 {
		t.Fatalf("unauthorized tenant request created a trace: %+v", tracesAfter)
	}
}

func TestServer_TestAnonymize_DisabledEntityDoesNotPersistPlaintext(t *testing.T) {
	const secretKey = "test_secret_key_123"
	const rawCPF = "529.982.247-25"
	srv, mux := setupTestServer(secretKey)
	srv.gateway.UpdateDynamicConfig(
		map[string]string{"CPF": "disabled"}, nil, nil, nil, 0, "",
	)

	payload := `{"prompt":"CPF 529.982.247-25","record_trace":true,"provider":"openai","model":"chatgpt.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/test-anonymize", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+secretKey)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected disabled entity request to pass, got %d: %s", w.Code, w.Body.String())
	}
	var response TestAnonymizeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode anonymize response: %v", err)
	}
	if !strings.Contains(response.Sanitized, rawCPF) {
		t.Fatalf("disabled CPF should remain unchanged in the administrative response: %q", response.Sanitized)
	}
	if len(response.Entities) != 0 || len(response.TokenMap) != 0 {
		t.Fatalf("disabled CPF must not create entities or vault mappings: %+v", response)
	}

	traces := srv.gateway.Traces().GetAll()
	if len(traces) != 1 || traces[0].SanitizedPrompt != "" {
		t.Fatalf("plaintext-capable policy persisted prompt content: %+v", traces)
	}
}

func TestServer_MetricsEndpoint(t *testing.T) {
	const secretKey = "test_secret_key_123"
	_, mux := setupTestServer(secretKey)

	for _, path := range []string{"/metrics", "/api/metrics"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for %s, got %d", path, w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if !strings.Contains(contentType, "text/plain") {
			t.Errorf("expected Content-Type text/plain, got %q", contentType)
		}

		body := w.Body.String()
		expectedMetrics := []string{
			"hadesgate_requests_total",
			"hadesgate_streaming_requests_total",
			"hadesgate_entities_masked_total",
			"hadesgate_active_streams",
			"hadesgate_active_vault_sessions",
			"hadesgate_uptime_seconds",
			"hadesgate_memory_alloc_bytes",
			"hadesgate_goroutines",
		}
		for _, em := range expectedMetrics {
			if !strings.Contains(body, em) {
				t.Errorf("metrics response for %s missing expected metric %q", path, em)
			}
		}
	}
}

func TestServer_DynamicConfigAndTenants(t *testing.T) {
	const secretKey = "test_secret_key_123"
	_, mux := setupTestServer(secretKey)

	// 1. GET /api/config
	reqGet := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	reqGet.Header.Set("Authorization", "Bearer "+secretKey)
	wGet := httptest.NewRecorder()
	mux.ServeHTTP(wGet, reqGet)
	if wGet.Code != http.StatusOK {
		t.Fatalf("GET /api/config returned %d: %s", wGet.Code, wGet.Body.String())
	}

	// 2. PUT /api/config
	newCfg := config.DynamicConfig{
		EntityActions:    map[string]string{"CPF": "block"},
		WebhookURL:       "https://discord.com/api/webhooks/test",
		EnableEgressScan: true,
		WebhookEvents:    []string{"block", "egress_leak"},
		FamilyThresholds: map[string]float64{"HEALTH_PHI": 0.65},
		EntityThresholds: map[string]float64{"PERSON": 0.85},
	}
	body, _ := json.Marshal(newCfg)
	reqPut := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewReader(body))
	reqPut.Header.Set("Authorization", "Bearer "+secretKey)
	wPut := httptest.NewRecorder()
	mux.ServeHTTP(wPut, reqPut)
	if wPut.Code != http.StatusOK {
		t.Fatalf("PUT /api/config returned %d: %s", wPut.Code, wPut.Body.String())
	}

	// 3. POST /api/tenants
	tenant := config.TenantConfig{
		ID:            "marketing",
		ProxyKey:      "mkt-proxy-key",
		RateLimit:     150,
		EntityActions: map[string]string{"EMAIL": "mask"},
	}
	tenantBody, _ := json.Marshal(tenant)
	reqTenantPost := httptest.NewRequest(http.MethodPost, "/api/tenants", bytes.NewReader(tenantBody))
	reqTenantPost.Header.Set("Authorization", "Bearer "+secretKey)
	wTenantPost := httptest.NewRecorder()
	mux.ServeHTTP(wTenantPost, reqTenantPost)
	if wTenantPost.Code != http.StatusOK {
		t.Fatalf("POST /api/tenants returned %d: %s", wTenantPost.Code, wTenantPost.Body.String())
	}

	// 4. GET /api/tenants
	reqTenantGet := httptest.NewRequest(http.MethodGet, "/api/tenants", nil)
	reqTenantGet.Header.Set("Authorization", "Bearer "+secretKey)
	wTenantGet := httptest.NewRecorder()
	mux.ServeHTTP(wTenantGet, reqTenantGet)
	if wTenantGet.Code != http.StatusOK {
		t.Fatalf("GET /api/tenants returned %d: %s", wTenantGet.Code, wTenantGet.Body.String())
	}
	var tenants []config.TenantConfig
	if err := json.Unmarshal(wTenantGet.Body.Bytes(), &tenants); err != nil {
		t.Fatalf("failed to decode tenants list: %v", err)
	}
	if len(tenants) != 1 || tenants[0].ID != "marketing" {
		t.Fatalf("expected tenant 'marketing', got %+v", tenants)
	}

	// 5. PUT /api/tenants (Edit tenant)
	updatedTenant := struct {
		config.TenantConfig
		OriginalID string `json:"original_id"`
	}{
		TenantConfig: config.TenantConfig{
			ID:            "marketing",
			Name:          "Growth Marketing",
			ProxyKey:      "mkt-updated-key",
			RateLimit:     300,
			EntityActions: map[string]string{"CPF": "block"},
		},
		OriginalID: "marketing",
	}
	updateBody, _ := json.Marshal(updatedTenant)
	reqTenantPut := httptest.NewRequest(http.MethodPut, "/api/tenants", bytes.NewReader(updateBody))
	reqTenantPut.Header.Set("Authorization", "Bearer "+secretKey)
	wTenantPut := httptest.NewRecorder()
	mux.ServeHTTP(wTenantPut, reqTenantPut)
	if wTenantPut.Code != http.StatusOK {
		t.Fatalf("PUT /api/tenants returned %d: %s", wTenantPut.Code, wTenantPut.Body.String())
	}

	// 6. DELETE /api/tenants
	reqTenantDel := httptest.NewRequest(http.MethodDelete, "/api/tenants?id=marketing", nil)
	reqTenantDel.Header.Set("Authorization", "Bearer "+secretKey)
	wTenantDel := httptest.NewRecorder()
	mux.ServeHTTP(wTenantDel, reqTenantDel)
	if wTenantDel.Code != http.StatusOK {
		t.Fatalf("DELETE /api/tenants returned %d: %s", wTenantDel.Code, wTenantDel.Body.String())
	}
}

func TestServer_ConfigurationSecretsAreWriteOnly(t *testing.T) {
	const adminKey = "admin-secret-key"
	const globalKey = "sec_HG_global_must_not_be_returned"
	const webhookSecret = "webhook_secret_must_not_be_returned"
	const webhookURL = "https://hooks.example.test/must-not-be-returned"
	const tenantKey = "sec_HG_tenant_must_not_be_returned"
	srv, mux := setupTestServer(adminKey)
	if err := srv.store.SaveDynamicConfig(context.Background(), config.DynamicConfig{
		GlobalProxyKey: globalKey,
		WebhookURL:     webhookURL,
		WebhookSecret:  webhookSecret,
	}); err != nil {
		t.Fatalf("saving dynamic config: %v", err)
	}
	if _, err := srv.store.SaveTenant(context.Background(), config.TenantConfig{
		ID: "01", Name: "Financeiro", ProxyKey: tenantKey,
	}); err != nil {
		t.Fatalf("saving tenant: %v", err)
	}

	for _, path := range []string{"/api/config", "/api/tenants"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+adminKey)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("GET %s returned %d: %s", path, w.Code, w.Body.String())
		}
		body := w.Body.String()
		if strings.Contains(body, globalKey) || strings.Contains(body, webhookURL) || strings.Contains(body, webhookSecret) || strings.Contains(body, tenantKey) {
			t.Fatalf("GET %s exposed write-only secret material: %s", path, body)
		}
		if !strings.Contains(body, "configured") {
			t.Fatalf("GET %s did not report secret configuration state: %s", path, body)
		}
	}

	configUpdate := `{"entity_actions":{},"global_proxy_key":"","webhook_secret":"","webhook_url":"","webhook_events":[]}`
	configReq := httptest.NewRequest(http.MethodPut, "/api/config", strings.NewReader(configUpdate))
	configReq.Header.Set("Authorization", "Bearer "+adminKey)
	configReq.Header.Set("Content-Type", "application/json")
	configW := httptest.NewRecorder()
	mux.ServeHTTP(configW, configReq)
	if configW.Code != http.StatusOK {
		t.Fatalf("write-only config update failed: %d %s", configW.Code, configW.Body.String())
	}

	tenantUpdate := `{"original_id":"01","id":"01","name":"Financeiro Atualizado","proxy_key":"","rate_limit":50,"entity_actions":{}}`
	tenantReq := httptest.NewRequest(http.MethodPut, "/api/tenants", strings.NewReader(tenantUpdate))
	tenantReq.Header.Set("Authorization", "Bearer "+adminKey)
	tenantReq.Header.Set("Content-Type", "application/json")
	tenantW := httptest.NewRecorder()
	mux.ServeHTTP(tenantW, tenantReq)
	if tenantW.Code != http.StatusOK {
		t.Fatalf("write-only tenant update failed: %d %s", tenantW.Code, tenantW.Body.String())
	}

	persistedCfg, err := srv.store.GetDynamicConfig(context.Background())
	if err != nil || persistedCfg.GlobalProxyKey != globalKey || persistedCfg.WebhookURL != webhookURL || persistedCfg.WebhookSecret != webhookSecret {
		t.Fatalf("blank write-only config fields erased secrets: cfg=%+v err=%v", persistedCfg, err)
	}
	persistedTenants, err := srv.store.ListTenants(context.Background())
	if err != nil || len(persistedTenants) != 1 || persistedTenants[0].ProxyKey != tenantKey {
		t.Fatalf("blank write-only tenant field erased secret: tenants=%+v err=%v", persistedTenants, err)
	}
}

func TestServer_ProviderCredentialLifecycle(t *testing.T) {
	const (
		adminKey       = "admin-secret-key"
		providerKey    = "sk-provider-original-secret"
		fingerprintKey = "fingerprint-secret-kept-outside-database"
	)
	srv, mux := setupTestServer(adminKey)
	srv.SetProviderFingerprintSecret(fingerprintKey)

	_, err := srv.store.SaveTenant(context.Background(), config.TenantConfig{
		ID:       "financeiro",
		Name:     "Financeiro",
		ProxyKey: "legacy-disabled-secret",
	})
	if err != nil {
		t.Fatalf("creating tenant: %v", err)
	}

	body := []byte(`{"tenant_id":"financeiro","provider":"openai","label":"Produção","provider_key":"` + providerKey + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/provider-credentials", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminKey)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("creating provider credential: status=%d body=%s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), providerKey) || strings.Contains(w.Body.String(), "fingerprint") {
		t.Fatalf("provider credential response exposed secret material: %s", w.Body.String())
	}
	var created struct {
		Credential config.ProviderCredential `json:"credential"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decoding credential response: %v", err)
	}
	if created.Credential.ID == "" || created.Credential.KeyHint != "••••cret" {
		t.Fatalf("unexpected credential metadata: %+v", created.Credential)
	}

	proxyBody := `{"model":"mock","messages":[{"role":"user","content":"olá"}]}`
	proxyReq := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(proxyBody))
	proxyReq.Header.Set("Authorization", "Bearer "+providerKey)
	proxyResponse := httptest.NewRecorder()
	srv.gateway.HandleChatCompletions(proxyResponse, proxyReq)
	if proxyResponse.Code != http.StatusOK {
		t.Fatalf("registered provider key was not accepted: status=%d body=%s", proxyResponse.Code, proxyResponse.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/provider-credentials?id="+created.Credential.ID, nil)
	deleteReq.Header.Set("Authorization", "Bearer "+adminKey)
	deleteResponse := httptest.NewRecorder()
	mux.ServeHTTP(deleteResponse, deleteReq)
	if deleteResponse.Code != http.StatusOK {
		t.Fatalf("revoking provider credential: status=%d body=%s", deleteResponse.Code, deleteResponse.Body.String())
	}

	revokedReq := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(proxyBody))
	revokedReq.Header.Set("Authorization", "Bearer "+providerKey)
	revokedResponse := httptest.NewRecorder()
	srv.gateway.HandleChatCompletions(revokedResponse, revokedReq)
	if revokedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("revoked provider key remained active: status=%d", revokedResponse.Code)
	}
}

func TestServer_AnvisaEndpoints(t *testing.T) {
	const secretKey = "test_secret_key_123"
	_, mux := setupTestServer(secretKey)

	// 1. GET /api/anvisa/status without auth -> 401
	reqNoAuth := httptest.NewRequest(http.MethodGet, "/api/anvisa/status", nil)
	wNoAuth := httptest.NewRecorder()
	mux.ServeHTTP(wNoAuth, reqNoAuth)
	if wNoAuth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthorized /api/anvisa/status, got %d", wNoAuth.Code)
	}

	// 2. GET /api/anvisa/status with admin auth -> 200
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/anvisa/status", nil)
	reqStatus.Header.Set("Authorization", "Bearer "+secretKey)
	wStatus := httptest.NewRecorder()
	mux.ServeHTTP(wStatus, reqStatus)
	if wStatus.Code != http.StatusOK {
		t.Fatalf("expected 200 for /api/anvisa/status, got %d: %s", wStatus.Code, wStatus.Body.String())
	}

	var status detector.MedicationCatalogStatus
	if err := json.Unmarshal(wStatus.Body.Bytes(), &status); err != nil {
		t.Fatalf("failed to decode status: %v", err)
	}
	if status.TermsCount < 20000 {
		t.Errorf("expected >= 20000 terms in status, got %d", status.TermsCount)
	}
}

func TestServer_ConfigUpdateHotReloadsCORS(t *testing.T) {
	const adminKey = "admin-secret-key"
	srv, mux := setupTestServer(adminKey)
	policy := NewSecurityPolicy([]string{"https://old.example"})
	srv.SetSecurityPolicy(policy)

	body := `{"entity_actions":{},"cors_allowed_origins":["https://new.example"],"webhook_events":[]}`
	req := httptest.NewRequest(http.MethodPut, "/api/config", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminKey)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("updating CORS config: status=%d body=%s", w.Code, w.Body.String())
	}

	handler := policy.Handler(mux)
	preflight := httptest.NewRequest(http.MethodOptions, "/api/config", nil)
	preflight.Header.Set("Origin", "https://new.example")
	preflightW := httptest.NewRecorder()
	handler.ServeHTTP(preflightW, preflight)
	if preflightW.Code != http.StatusNoContent || preflightW.Header().Get("Access-Control-Allow-Origin") != "https://new.example" {
		t.Fatalf("CORS update was not applied without restart: status=%d headers=%v", preflightW.Code, preflightW.Header())
	}
}

func TestServer_PlaygroundProxy(t *testing.T) {
	const adminKey = "admin-secret-key"
	const proxyKey = "proxy-secret-key"

	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("unexpected upstream path: %s", r.URL.Path)
		}
		if r.Header.Get("X-API-Key") != "anthropic-key" {
			t.Fatalf("Anthropic key missing or incorrect: %s", r.Header.Get("X-API-Key"))
		}
		var request map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode upstream request: %v", err)
		}
		messages := request["messages"].([]interface{})
		first := messages[0].(map[string]interface{})["content"].(string)
		if strings.Contains(first, "529.982.247-25") {
			t.Fatalf("raw CPF leaked upstream: %s", first)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "msg_test", "type": "message", "role": "assistant",
			"content": []interface{}{map[string]interface{}{"type": "text", "text": "Recebido: " + first}},
		})
	}))
	defer upstreamServer.Close()

	v := vault.NewMemoryVault(5 * time.Minute)
	eng := anonymizer.NewEngine(anonymizer.DefaultConfig(), v)
	traces := observability.NewTraceLogger(100)
	wh := webhook.NewDispatcher(webhook.Config{})
	upstream := proxy.NewUpstreamDispatcher(proxy.UpstreamConfig{
		AnthropicURL: upstreamServer.URL,
		AnthropicKey: "anthropic-key",
	})
	gw := proxy.NewGatewayWithOptions(eng, upstream, traces, proxy.GatewayOptions{
		GlobalProxyKey: proxyKey,
		Webhook:        wh,
	})
	srv := NewServer(eng, gw, adminKey, nil)
	store, _ := config.NewStore(traces.DB(), []byte("0123456789abcdef0123456789abcdef"))
	srv.SetStore(store)
	srv.SetWebhook(wh)
	srv.SetUpstream(upstream)

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/messages", gw.HandleAnthropicMessages)
	mux.HandleFunc("/v1/chat/completions", gw.HandleChatCompletions)
	srv.RegisterRoutes(mux)

	// 1. Calling public /v1/messages with AdminKey must be rejected with 401
	pubReq := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"claude-test","messages":[{"role":"user","content":"CPF 529.982.247-25"}]}`))
	pubReq.Header.Set("Authorization", "Bearer "+adminKey)
	pubW := httptest.NewRecorder()
	mux.ServeHTTP(pubW, pubReq)
	if pubW.Code != http.StatusUnauthorized {
		t.Fatalf("public /v1/messages accepted admin key: got %d, want 401", pubW.Code)
	}

	// 2. Calling /api/playground/v1/messages without auth must be rejected with 401
	unauthReq := httptest.NewRequest(http.MethodPost, "/api/playground/v1/messages", strings.NewReader(`{"model":"claude-test","messages":[{"role":"user","content":"CPF 529.982.247-25"}]}`))
	unauthW := httptest.NewRecorder()
	mux.ServeHTTP(unauthW, unauthReq)
	if unauthW.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated /api/playground/v1/messages got %d, want 401", unauthW.Code)
	}

	// 3. Calling /api/playground/v1/messages with AdminKey must succeed and mask/detokenize
	authReq := httptest.NewRequest(http.MethodPost, "/api/playground/v1/messages", strings.NewReader(`{"model":"claude-test","messages":[{"role":"user","content":"CPF 529.982.247-25"}]}`))
	authReq.Header.Set("Authorization", "Bearer "+adminKey)
	authW := httptest.NewRecorder()
	mux.ServeHTTP(authW, authReq)
	if authW.Code != http.StatusOK {
		t.Fatalf("/api/playground/v1/messages with admin session failed: got %d: %s", authW.Code, authW.Body.String())
	}
	if !strings.Contains(authW.Body.String(), "529.982.247-25") {
		t.Fatalf("detokenized response did not restore CPF: %s", authW.Body.String())
	}

	// 4. Calling /api/playground/v1/chat/completions with mock model must succeed
	mockReq := httptest.NewRequest(http.MethodPost, "/api/playground/v1/chat/completions", strings.NewReader(`{"model":"mock","messages":[{"role":"user","content":"hello"}]}`))
	mockReq.Header.Set("Authorization", "Bearer "+adminKey)
	mockW := httptest.NewRecorder()
	mux.ServeHTTP(mockW, mockReq)
	if mockW.Code != http.StatusOK {
		t.Fatalf("/api/playground/v1/chat/completions mock failed: got %d: %s", mockW.Code, mockW.Body.String())
	}
}
