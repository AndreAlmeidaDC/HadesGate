package config

import "testing"

func TestLoadGeminiConfig(t *testing.T) {
	t.Setenv("HADESGATE_ADMIN_KEY", "test_admin_key")
	t.Setenv("GEMINI_BASE_URL", "")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "")

	cfg := Load()
	if cfg.GeminiBaseURL != "https://generativelanguage.googleapis.com" {
		t.Fatalf("expected default GeminiBaseURL, got %q", cfg.GeminiBaseURL)
	}
	if cfg.GeminiKey != "" {
		t.Fatalf("expected empty GeminiKey by default, got %q", cfg.GeminiKey)
	}

	// Fallback to GOOGLE_API_KEY when GEMINI_API_KEY is unset
	t.Setenv("GOOGLE_API_KEY", "google_fallback_key")
	cfg = Load()
	if cfg.GeminiKey != "google_fallback_key" {
		t.Fatalf("expected GOOGLE_API_KEY fallback, got %q", cfg.GeminiKey)
	}

	// Explicit GEMINI_API_KEY and GEMINI_BASE_URL override defaults and fallback
	t.Setenv("GEMINI_API_KEY", "gemini_primary_key")
	t.Setenv("GEMINI_BASE_URL", "https://custom-gemini.example.com")
	cfg = Load()
	if cfg.GeminiKey != "gemini_primary_key" {
		t.Fatalf("expected GEMINI_API_KEY override, got %q", cfg.GeminiKey)
	}
	if cfg.GeminiBaseURL != "https://custom-gemini.example.com" {
		t.Fatalf("expected custom GEMINI_BASE_URL, got %q", cfg.GeminiBaseURL)
	}
}

func TestLoadSeparatesAdminProxyAndLoadsBYOKSecurity(t *testing.T) {
	t.Setenv("HADESGATE_ADMIN_KEY", "same-key-must-not-be-reused")
	t.Setenv("HADESGATE_PROXY_KEY", "same-key-must-not-be-reused")
	t.Setenv("HADESGATE_FINGERPRINT_SECRET", "stable-fingerprint-secret")
	t.Setenv("HADESGATE_ALLOW_LEGACY_URL_KEYS", "true")

	cfg := Load()
	if cfg.ProxyKey == cfg.AdminKey {
		t.Fatal("proxy key must never reuse the administrative key")
	}
	if cfg.ProviderFingerprintSecret != "stable-fingerprint-secret" {
		t.Fatalf("unexpected fingerprint secret: %q", cfg.ProviderFingerprintSecret)
	}
	if !cfg.AllowLegacyURLKeys {
		t.Fatal("expected legacy URL flag to load from environment")
	}
}
