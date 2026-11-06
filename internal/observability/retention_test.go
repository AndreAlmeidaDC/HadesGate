package observability

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestTenantRetentionAppliesAgeAndRowQuotaWithoutAffectingOtherTenants(t *testing.T) {
	logger := NewTraceLogger(100)
	t.Cleanup(func() { _ = logger.Close() })
	ctx := context.Background()
	if err := logger.SetRetentionPolicy(ctx, RetentionPolicy{TenantID: "finance", RetentionDays: 7, MaxRows: 2}); err != nil {
		t.Fatal(err)
	}
	logger.Record(Trace{ID: "old", Tenant: "finance", Timestamp: time.Now().AddDate(0, 0, -10)})
	for index := 0; index < 3; index++ {
		logger.Record(Trace{ID: fmt.Sprintf("new-%d", index), Tenant: "finance"})
	}
	logger.Record(Trace{ID: "other", Tenant: "engineering", Timestamp: time.Now().AddDate(0, 0, -30)})
	finance, total := logger.GetPagedForTenant("finance", 10, 0)
	if total != 2 || len(finance) != 2 {
		t.Fatalf("tenant quota not applied: total=%d traces=%+v", total, finance)
	}
	other, total := logger.GetPagedForTenant("engineering", 10, 0)
	if total != 1 || len(other) != 1 {
		t.Fatalf("other tenant was affected: %+v", other)
	}
}
