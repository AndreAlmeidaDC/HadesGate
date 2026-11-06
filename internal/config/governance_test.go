package config

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func newGovernanceTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	store, err := NewStore(db, []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func TestAdminCredentialsAreAccountlessScopedAndOneTimeReveal(t *testing.T) {
	store := newGovernanceTestStore(t)
	ctx := context.Background()
	created, err := store.CreateAdminCredential(ctx, "NOC", RoleOperator)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(created.Secret, "hg_admin_") {
		t.Fatalf("secret was not generated: %+v", created)
	}
	if credential, ok := store.AuthenticateAdminCredential(ctx, created.Secret); !ok || credential.KeyID != created.KeyID || credential.Role != RoleOperator {
		t.Fatalf("generated credential did not authenticate: %+v, %v", credential, ok)
	}
	if _, ok := store.AuthenticateAdminCredential(ctx, "wrong-secret"); ok {
		t.Fatal("invalid secret authenticated")
	}
	listed, err := store.ListAdminCredentials(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].Secret != "" {
		t.Fatalf("listing exposed or lost credential: %+v", listed)
	}
	if err := store.DeleteAdminCredential(ctx, created.KeyID); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.AuthenticateAdminCredential(ctx, created.Secret); ok {
		t.Fatal("revoked credential authenticated")
	}
}

func TestGovernanceAuditUsesUniqueNonceAndSignature(t *testing.T) {
	store := newGovernanceTestStore(t)
	ctx := context.Background()
	for index := 0; index < 2; index++ {
		if err := store.RecordAuditEvent(ctx, AuditEvent{ActorKeyID: "bootstrap", ActorRole: RoleAdmin, Action: "policy.updated", TargetType: "tenant", TargetID: "finance"}); err != nil {
			t.Fatal(err)
		}
	}
	events, err := store.ListAuditEvents(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Nonce == events[1].Nonce {
		t.Fatalf("audit nonces are not unique: %+v", events)
	}
	for _, event := range events {
		if event.Signature == "" || event.Signature != store.auditSignature(event) {
			t.Fatalf("invalid audit signature: %+v", event)
		}
	}
}

func TestEncryptedBackupExcludesTelemetryAndRestoresCiphertextConfiguration(t *testing.T) {
	store := newGovernanceTestStore(t)
	ctx := context.Background()
	if err := store.SaveDynamicConfig(ctx, DynamicConfig{GlobalProxyKey: "proxy-plaintext", WebhookSecret: "webhook-plaintext"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `CREATE TABLE audit_traces(sanitized_prompt TEXT); INSERT INTO audit_traces VALUES('trace-canary')`); err != nil {
		t.Fatal(err)
	}
	credential, err := store.CreateAdminCredential(ctx, "Auditoria", RoleViewer)
	if err != nil {
		t.Fatal(err)
	}
	backup, err := store.ExportEncryptedBackup(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(backup, "proxy-plaintext") || strings.Contains(backup, "webhook-plaintext") || strings.Contains(backup, credential.Secret) {
		t.Fatal("backup exposed plaintext secret")
	}
	decrypted, err := store.secrets.decrypt(backup, backupAAD)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(decrypted, "trace-canary") || strings.Contains(decrypted, "audit_traces") {
		t.Fatal("backup included telemetry")
	}
	if _, err := store.db.ExecContext(ctx, `DELETE FROM system_config; DELETE FROM admin_credentials`); err != nil {
		t.Fatal(err)
	}
	if err := store.RestoreEncryptedBackup(ctx, backup); err != nil {
		t.Fatal(err)
	}
	cfg, err := store.GetDynamicConfig(ctx)
	if err != nil || cfg.GlobalProxyKey != "proxy-plaintext" || cfg.WebhookSecret != "webhook-plaintext" {
		t.Fatalf("configuration was not restored: %+v, %v", cfg, err)
	}
	if _, ok := store.AuthenticateAdminCredential(ctx, credential.Secret); !ok {
		t.Fatal("admin credential fingerprint was not restored")
	}
}
