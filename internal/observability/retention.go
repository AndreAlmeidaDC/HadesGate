package observability

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// RetentionPolicy bounds Zero-PII telemetry by age and row count. TenantID "*"
// is the global fallback; an exact tenant policy takes precedence.
type RetentionPolicy struct {
	TenantID      string    `json:"tenant_id"`
	RetentionDays int       `json:"retention_days"`
	MaxRows       int       `json:"max_rows"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (l *TraceLogger) SetRetentionPolicy(ctx context.Context, policy RetentionPolicy) error {
	if l == nil || l.db == nil {
		return nil
	}
	policy.TenantID = strings.TrimSpace(policy.TenantID)
	if policy.TenantID == "" {
		policy.TenantID = "*"
	}
	if policy.RetentionDays < 0 || policy.MaxRows < 0 {
		return fmt.Errorf("retention days and max rows cannot be negative")
	}
	policy.UpdatedAt = time.Now().UTC()
	_, err := l.db.ExecContext(ctx, `INSERT INTO telemetry_retention(tenant_id,retention_days,max_rows,updated_at) VALUES(?,?,?,?) ON CONFLICT(tenant_id) DO UPDATE SET retention_days=excluded.retention_days,max_rows=excluded.max_rows,updated_at=excluded.updated_at`, policy.TenantID, policy.RetentionDays, policy.MaxRows, policy.UpdatedAt)
	if err != nil {
		return fmt.Errorf("saving telemetry retention: %w", err)
	}
	return l.applyRetention(ctx, policy.TenantID)
}

func (l *TraceLogger) DeleteRetentionPolicy(ctx context.Context, tenantID string) error {
	if l == nil || l.db == nil {
		return nil
	}
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		tenantID = "*"
	}
	_, err := l.db.ExecContext(ctx, `DELETE FROM telemetry_retention WHERE tenant_id = ?`, tenantID)
	return err
}

func (l *TraceLogger) ListRetentionPolicies(ctx context.Context) ([]RetentionPolicy, error) {
	if l == nil || l.db == nil {
		return []RetentionPolicy{}, nil
	}
	rows, err := l.db.QueryContext(ctx, `SELECT tenant_id,retention_days,max_rows,updated_at FROM telemetry_retention ORDER BY tenant_id`)
	if err != nil {
		return nil, fmt.Errorf("listing telemetry retention: %w", err)
	}
	defer rows.Close()
	result := []RetentionPolicy{}
	for rows.Next() {
		var policy RetentionPolicy
		if err := rows.Scan(&policy.TenantID, &policy.RetentionDays, &policy.MaxRows, &policy.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, policy)
	}
	return result, rows.Err()
}

func (l *TraceLogger) effectiveRetention(ctx context.Context, tenant string) (RetentionPolicy, error) {
	var policy RetentionPolicy
	err := l.db.QueryRowContext(ctx, `SELECT tenant_id,retention_days,max_rows,updated_at FROM telemetry_retention WHERE tenant_id = ?`, tenant).Scan(&policy.TenantID, &policy.RetentionDays, &policy.MaxRows, &policy.UpdatedAt)
	if err == nil {
		return policy, nil
	}
	err = l.db.QueryRowContext(ctx, `SELECT tenant_id,retention_days,max_rows,updated_at FROM telemetry_retention WHERE tenant_id = '*'`).Scan(&policy.TenantID, &policy.RetentionDays, &policy.MaxRows, &policy.UpdatedAt)
	return policy, err
}

func (l *TraceLogger) applyRetention(ctx context.Context, tenant string) error {
	if tenant == "*" {
		policies, err := l.ListRetentionPolicies(ctx)
		if err != nil {
			return err
		}
		if len(policies) == 0 {
			return nil
		}
		rows, err := l.db.QueryContext(ctx, `SELECT DISTINCT tenant FROM audit_traces`)
		if err != nil {
			return err
		}
		var tenants []string
		for rows.Next() {
			var item string
			if err := rows.Scan(&item); err != nil {
				rows.Close()
				return err
			}
			tenants = append(tenants, item)
		}
		rows.Close()
		for _, item := range tenants {
			if err := l.applyRetention(ctx, item); err != nil {
				return err
			}
		}
		return nil
	}
	policy, err := l.effectiveRetention(ctx, tenant)
	if err != nil {
		return nil
	}
	if policy.RetentionDays > 0 {
		cutoff := time.Now().UTC().AddDate(0, 0, -policy.RetentionDays).Unix()
		if _, err := l.db.ExecContext(ctx, `DELETE FROM audit_traces WHERE tenant = ? AND timestamp_unix < ?`, tenant, cutoff); err != nil {
			return err
		}
	}
	if policy.MaxRows > 0 {
		_, err := l.db.ExecContext(ctx, `DELETE FROM audit_traces WHERE tenant = ? AND seq NOT IN (SELECT seq FROM audit_traces WHERE tenant = ? ORDER BY seq DESC LIMIT ?)`, tenant, tenant, policy.MaxRows)
		if err != nil {
			return err
		}
	}
	return nil
}
