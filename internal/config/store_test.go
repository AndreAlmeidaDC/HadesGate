package config

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestConfigStore_CRUD(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("opening sqlite: %v", err)
	}
	defer db.Close()

	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	ctx := context.Background()

	// 1. Seed if empty
	initialCfg := DynamicConfig{
		EntityActions:    map[string]string{"CPF": "mask", "API_KEY": "block"},
		GlobalProxyKey:   "sec_HG_global_initial",
		WebhookURL:       "https://discord.com/api/webhooks/test",
		EnableEgressScan: true,
		WebhookEvents:    []string{"block", "egress_leak"},
	}
	initialTenants := []TenantConfig{
		{
			ID:            "01",
			Name:          "Financeiro",
			ProxyKey:      "acme-secret",
			RateLimit:     100,
			EntityActions: map[string]string{"CPF": "block"},
		},
	}

	if err := store.SeedIfEmpty(ctx, initialCfg, initialTenants); err != nil {
		t.Fatalf("SeedIfEmpty failed: %v", err)
	}

	// 2. Read back DynamicConfig
	cfg, err := store.GetDynamicConfig(ctx)
	if err != nil {
		t.Fatalf("GetDynamicConfig failed: %v", err)
	}
	if cfg.WebhookURL != initialCfg.WebhookURL {
		t.Errorf("got webhook URL %s, want %s", cfg.WebhookURL, initialCfg.WebhookURL)
	}
	if cfg.GlobalProxyKey != "sec_HG_global_initial" {
		t.Errorf("got GlobalProxyKey %s, want sec_HG_global_initial", cfg.GlobalProxyKey)
	}
	if !cfg.EnableEgressScan {
		t.Errorf("expected EnableEgressScan to be true")
	}
	if cfg.EntityActions["API_KEY"] != "block" {
		t.Errorf("expected API_KEY to be block, got %s", cfg.EntityActions["API_KEY"])
	}

	// 3. Update DynamicConfig with rotated global key
	cfg.WebhookURL = "https://hooks.slack.com/services/test"
	cfg.GlobalProxyKey = "sec_HG_global_rotated"
	cfg.EntityActions["SECRET"] = "block"
	if err := store.SaveDynamicConfig(ctx, cfg); err != nil {
		t.Fatalf("SaveDynamicConfig failed: %v", err)
	}
	updatedCfg, _ := store.GetDynamicConfig(ctx)
	if updatedCfg.GlobalProxyKey != "sec_HG_global_rotated" {
		t.Errorf("expected updated global proxy key, got %s", updatedCfg.GlobalProxyKey)
	}
	if updatedCfg.WebhookURL != "https://hooks.slack.com/services/test" {
		t.Errorf("expected updated webhook URL")
	}
	if updatedCfg.EntityActions["SECRET"] != "block" {
		t.Errorf("expected SECRET=block in updated config")
	}

	// 4. Test Tenants List, Upsert and Delete
	tenants, err := store.ListTenants(ctx)
	if err != nil {
		t.Fatalf("ListTenants failed: %v", err)
	}
	if len(tenants) != 1 || tenants[0].ID != "01" || tenants[0].Name != "Financeiro" {
		t.Fatalf("expected 1 tenant '01' ('Financeiro'), got %+v", tenants)
	}

	// Add second tenant WITHOUT specifying ID -> should auto-assign "02"
	newTenant := TenantConfig{
		Name:      "Marketing",
		ProxyKey:  "mkt-secret",
		RateLimit: 50,
	}
	saved02, err := store.SaveTenant(ctx, newTenant)
	if err != nil {
		t.Fatalf("SaveTenant failed: %v", err)
	}
	if saved02.ID != "02" {
		t.Fatalf("expected auto-assigned ID '02', got %q", saved02.ID)
	}

	tenants, _ = store.ListTenants(ctx)
	if len(tenants) != 2 {
		t.Fatalf("expected 2 tenants, got %d", len(tenants))
	}

	// 5. Test UpdateTenant without changing ID (updating Name, Key, Limit, Actions)
	updated02 := TenantConfig{
		ID:            "02",
		Name:          "Marketing & Growth",
		ProxyKey:      "mkt-new-secret",
		RateLimit:     75,
		EntityActions: map[string]string{"EMAIL": "block"},
	}
	if err := store.UpdateTenant(ctx, "02", updated02); err != nil {
		t.Fatalf("UpdateTenant failed: %v", err)
	}
	tenants, _ = store.ListTenants(ctx)
	var found02 *TenantConfig
	for _, tItem := range tenants {
		if tItem.ID == "02" {
			found02 = &tItem
			break
		}
	}
	if found02 == nil || found02.Name != "Marketing & Growth" || found02.ProxyKey != "mkt-new-secret" || found02.EntityActions["EMAIL"] != "block" {
		t.Fatalf("expected updated tenant '02', got %+v", found02)
	}

	// Delete tenant 01
	if err := store.DeleteTenant(ctx, "01"); err != nil {
		t.Fatalf("DeleteTenant failed: %v", err)
	}
	tenants, _ = store.ListTenants(ctx)
	if len(tenants) != 1 || tenants[0].ID != "02" {
		t.Fatalf("expected 1 tenant '02' after delete, got %+v", tenants)
	}

	// Rename tenant ID
	renamedTenant := TenantConfig{
		ID:        "99",
		Name:      "Marketing & Growth",
		ProxyKey:  "mkt-new-secret",
		RateLimit: 75,
	}
	if err := store.UpdateTenant(ctx, "02", renamedTenant); err != nil {
		t.Fatalf("UpdateTenant with rename failed: %v", err)
	}
	tenants, _ = store.ListTenants(ctx)
	if len(tenants) != 1 || tenants[0].ID != "99" {
		t.Fatalf("expected tenant '99' after rename, got %+v", tenants)
	}
}

func TestStoreProviderCredentialsPersistOnlyFingerprint(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("opening sqlite: %v", err)
	}
	defer db.Close()

	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	ctx := context.Background()
	_, err = store.SaveTenant(ctx, TenantConfig{
		ID:       "financeiro",
		Name:     "Financeiro",
		ProxyKey: "legacy-proxy-key",
	})
	if err != nil {
		t.Fatalf("SaveTenant failed: %v", err)
	}

	const rawProviderKey = "sk-provider-must-never-be-persisted"
	credential, err := store.SaveProviderCredential(ctx, ProviderCredential{
		ID:          "cred-01",
		TenantID:    "financeiro",
		Provider:    "openai",
		Label:       "Produção",
		KeyHint:     "…sted",
		Fingerprint: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})
	if err != nil {
		t.Fatalf("SaveProviderCredential failed: %v", err)
	}
	encoded, err := json.Marshal(credential)
	if err != nil {
		t.Fatalf("marshaling credential: %v", err)
	}
	if strings.Contains(string(encoded), credential.Fingerprint) || strings.Contains(string(encoded), rawProviderKey) {
		t.Fatalf("credential response exposed sensitive material: %s", encoded)
	}

	credentials, err := store.ListProviderCredentials(ctx, "financeiro")
	if err != nil {
		t.Fatalf("ListProviderCredentials failed: %v", err)
	}
	if len(credentials) != 1 || credentials[0].Fingerprint != credential.Fingerprint {
		t.Fatalf("unexpected stored credentials: %+v", credentials)
	}
	fingerprints, err := store.ProviderKeyFingerprints(ctx)
	if err != nil {
		t.Fatalf("ProviderKeyFingerprints failed: %v", err)
	}
	if len(fingerprints["financeiro"]) != 1 || fingerprints["financeiro"][0] != credential.Fingerprint {
		t.Fatalf("unexpected grouped fingerprints: %+v", fingerprints)
	}

	if err := store.DeleteProviderCredential(ctx, credential.ID); err != nil {
		t.Fatalf("DeleteProviderCredential failed: %v", err)
	}
	credentials, err = store.ListProviderCredentials(ctx, "")
	if err != nil || len(credentials) != 0 {
		t.Fatalf("credential was not revoked: credentials=%+v err=%v", credentials, err)
	}
}
