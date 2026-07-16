package platform

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNotificationInboxDispatchAndReadLifecycle(t *testing.T) {
	repository := NewMemoryRepository()
	service := NewNotificationService(repository)
	now := time.Date(2026, 7, 15, 2, 30, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	dispatch, err := service.Create(context.Background(), CreateNotificationInput{
		OrganizationID: "org_topoai", RecipientID: "holden@topoai.dev", Category: "provider", Severity: "critical",
		Title: "Provider route removed", Body: "Provider health made the route ineligible.",
		SourceType: "provider_health", SourceID: "provider_openai", ActionURL: "/providers",
	})
	if err != nil {
		t.Fatalf("create notification: %v", err)
	}
	if dispatch.Notification.Status != "unread" || len(dispatch.Deliveries) != 1 || dispatch.Deliveries[0].Channel != "slack" || dispatch.Deliveries[0].Status != "queued" {
		t.Fatalf("unexpected dispatch: %#v", dispatch)
	}

	read, err := service.MarkRead(context.Background(), "org_topoai", "holden@topoai.dev", dispatch.Notification.ID)
	if err != nil || read.Status != "read" || read.ReadAt == nil {
		t.Fatalf("mark read: %#v %v", read, err)
	}
	unread, err := service.MarkUnread(context.Background(), "org_topoai", "holden@topoai.dev", dispatch.Notification.ID)
	if err != nil || unread.Status != "unread" || unread.ReadAt != nil {
		t.Fatalf("mark unread: %#v %v", unread, err)
	}
	archived, err := service.Archive(context.Background(), "org_topoai", "holden@topoai.dev", dispatch.Notification.ID)
	if err != nil || archived.Status != "archived" || archived.ReadAt == nil {
		t.Fatalf("archive: %#v %v", archived, err)
	}

	updated, err := service.MarkAllRead(context.Background(), "org_topoai", "holden@topoai.dev")
	if err != nil || updated != 3 {
		t.Fatalf("mark all read updated=%d err=%v", updated, err)
	}
}

func TestNotificationPreferenceValidationAndDeferredDelivery(t *testing.T) {
	repository := NewMemoryRepository()
	service := NewNotificationService(repository)
	now := time.Date(2026, 7, 15, 15, 0, 0, 0, time.UTC) // 23:00 Asia/Shanghai.
	service.now = func() time.Time { return now }

	_, err := service.UpsertPreference(context.Background(), UpsertNotificationPreferenceInput{
		OrganizationID: "org_topoai", RecipientID: "holden@topoai.dev", DigestFrequency: "realtime",
		MinimumSeverity: "info", Timezone: "Mars/Olympus", QuietStart: "22:00", QuietEnd: "08:00",
		Destinations:     []NotificationDestination{{Channel: "in_app", Target: "self"}},
		CategoryChannels: map[string][]string{"alert": {"in_app"}},
	})
	assertValidationCode(t, err, "notification_timezone_invalid")

	preference, err := service.UpsertPreference(context.Background(), UpsertNotificationPreferenceInput{
		OrganizationID: "org_topoai", RecipientID: "holden@topoai.dev", DigestFrequency: "realtime",
		MinimumSeverity: "warning", Timezone: "Asia/Shanghai", QuietHoursEnabled: true, QuietStart: "22:00", QuietEnd: "08:00",
		Destinations: []NotificationDestination{
			{Channel: "in_app", Target: "self", DisplayName: "Inbox"},
			{Channel: "email", Target: "holden@topoai.dev", DisplayName: "Work email"},
		},
		CategoryChannels: map[string][]string{"alert": {"in_app", "email"}},
	})
	if err != nil || preference.QuietStart != "22:00" {
		t.Fatalf("upsert preference: %#v %v", preference, err)
	}

	dispatch, err := service.Create(context.Background(), CreateNotificationInput{
		OrganizationID: "org_topoai", RecipientID: "holden@topoai.dev", Category: "alert", Severity: "critical",
		Title: "Critical error rate", Body: "Production error rate exceeded the critical threshold.", ActionURL: "/alerts",
	})
	if err != nil {
		t.Fatalf("create deferred notification: %v", err)
	}
	if len(dispatch.Deliveries) != 1 || dispatch.Deliveries[0].Status != "deferred" || !dispatch.Deliveries[0].AvailableAt.Equal(time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected quiet-hours delivery: %#v", dispatch.Deliveries)
	}
}

func TestNotificationEscalationAndDeliveryRetry(t *testing.T) {
	repository := NewMemoryRepository()
	service := NewNotificationService(repository)
	now := time.Date(2026, 7, 15, 2, 30, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	policy, err := service.CreatePolicy(context.Background(), CreateNotificationEscalationPolicyInput{
		OrganizationID: "org_topoai", Name: "Security escalation", Status: "active",
		Categories: []string{"security"}, MinimumSeverity: "warning", AcknowledgeWithinMinutes: 10,
		RepeatEveryMinutes: 15, MaxEscalations: 2,
		Routes: []NotificationEscalationRoute{
			{Level: 1, DelayMinutes: 0, Channel: "slack", Target: "C_SECURITY"},
			{Level: 2, DelayMinutes: 15, Channel: "email", Target: "security@topoai.dev"},
		},
	})
	if err != nil || policy.Status != "active" {
		t.Fatalf("create escalation policy: %#v %v", policy, err)
	}
	evaluation, err := service.EvaluateEscalation(context.Background(), EvaluateNotificationEscalationInput{
		OrganizationID: "org_topoai", Category: "security", Severity: "critical", UnacknowledgedMinutes: 26,
	})
	if err != nil || !evaluation.Matched {
		t.Fatalf("evaluate escalation: %#v %v", evaluation, err)
	}
	createdPolicyMatched := false
	for _, match := range evaluation.Matches {
		if match.PolicyID == policy.ID && len(match.Routes) == 2 {
			createdPolicyMatched = true
		}
	}
	if !createdPolicyMatched {
		t.Fatalf("created escalation policy was not fully matched: %#v", evaluation)
	}
	paused, err := service.PausePolicy(context.Background(), "org_topoai", policy.ID)
	if err != nil || paused.Status != "paused" {
		t.Fatalf("pause policy: %#v %v", paused, err)
	}

	retry, err := service.RetryDelivery(context.Background(), "org_topoai", "ndel_budget_email_failed")
	if err != nil || retry.Status != "queued" || retry.Attempt != 2 || retry.ParentID == nil || *retry.ParentID != "ndel_budget_email_failed" {
		t.Fatalf("retry delivery: %#v %v", retry, err)
	}
	_, err = service.RetryDelivery(context.Background(), "org_topoai", "ndel_provider_slack")
	assertValidationCode(t, err, "notification_delivery_retry_invalid")
}

func assertValidationCode(t *testing.T, err error, expected string) {
	t.Helper()
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("expected validation error %q, got %v", expected, err)
	}
	if validation.Code != expected {
		t.Fatalf("expected validation code %q, got %q", expected, validation.Code)
	}
}
