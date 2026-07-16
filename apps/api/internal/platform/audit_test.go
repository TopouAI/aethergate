package platform

import (
	"context"
	"testing"
	"time"
)

func TestAuditAppendVerifyRetentionAndExport(t *testing.T) {
	repository := NewMemoryRepository()
	service := NewAuditService(repository)
	now := time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	initial, err := service.Verify(context.Background(), "org_topoai")
	if err != nil || !initial.Valid || initial.EventCount != 3 || initial.HeadHash == "" {
		t.Fatalf("verify initial chain: %#v %v", initial, err)
	}
	event, err := service.Append(context.Background(), AppendAuditEventInput{
		OrganizationID: "org_topoai", ActorID: "user_holden", ActorEmail: "holden@topoai.dev",
		Action: "routing_policy.activated", ResourceType: "routing_policy", ResourceID: "route_primary",
		Outcome: "success", RiskLevel: "high", Source: "control-plane", Reason: "Production rollout",
		RequestID: "req_audit_test", IPAddress: "10.0.0.9", BeforeState: map[string]any{"status": "paused"}, AfterState: map[string]any{"status": "active"},
	})
	if err != nil || event.PreviousHash != initial.HeadHash || len(event.IntegrityHash) != 64 {
		t.Fatalf("append audit event: %#v %v", event, err)
	}
	verified, err := service.Verify(context.Background(), "org_topoai")
	if err != nil || !verified.Valid || verified.EventCount != 4 || verified.HeadHash != event.IntegrityHash {
		t.Fatalf("verify appended chain: %#v %v", verified, err)
	}

	policy, err := service.UpsertRetention(context.Background(), UpsertAuditRetentionInput{OrganizationID: "org_topoai", RetentionDays: 730, LegalHold: true, ExportFormat: "jsonl", UpdatedBy: "security@topoai.dev"})
	if err != nil || policy.RetentionDays != 730 || !policy.LegalHold {
		t.Fatalf("upsert retention: %#v %v", policy, err)
	}
	_, err = service.UpsertRetention(context.Background(), UpsertAuditRetentionInput{OrganizationID: "org_topoai", RetentionDays: 7, ExportFormat: "csv"})
	assertValidationCode(t, err, "audit_retention_invalid")

	export, err := service.QueueExport(context.Background(), QueueAuditExportInput{
		OrganizationID: "org_topoai", RequestedBy: "holden@topoai.dev", Format: "csv",
		PeriodStart: "2026-07-01T00:00:00Z", PeriodEnd: "2026-07-15T08:00:00Z", Filters: map[string]string{"riskLevel": "high"},
	})
	if err != nil || export.Status != "queued" {
		t.Fatalf("queue export: %#v %v", export, err)
	}
	retry, err := service.RetryExport(context.Background(), "org_topoai", "aexp_failed", "holden@topoai.dev")
	if err != nil || retry.Status != "queued" || retry.ParentID == nil || *retry.ParentID != "aexp_failed" {
		t.Fatalf("retry export: %#v %v", retry, err)
	}
	_, err = service.RetryExport(context.Background(), "org_topoai", "aexp_q2_success", "")
	assertValidationCode(t, err, "audit_export_retry_invalid")
}
