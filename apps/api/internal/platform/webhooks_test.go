package platform

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestWebhookLifecycleQueuesTestAndReplay(t *testing.T) {
	service := NewWebhookService(NewMemoryRepository())
	created, err := service.Create(context.Background(), CreateWebhookInput{
		Name: "Engineering automation", Destination: "https://example.com/aethergate",
		Events: []string{"request.completed", "alert.triggered"}, SampleRate: 25,
		IncludeData: true, PropertyFilters: []WebhookPropertyFilter{{Key: "project", Value: "copilot"}},
		MaxAttempts: 4, TimeoutSeconds: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(created.SigningSecret, "whsec_") || created.Record.SigningSecretPrefix == "" {
		t.Fatalf("secret metadata=%+v", created)
	}
	encoded, _ := json.Marshal(created.Record)
	if strings.Contains(string(encoded), "SigningSecretDigest") || strings.Contains(string(encoded), "vault://") || strings.Contains(string(encoded), created.SigningSecret) {
		t.Fatalf("secret material leaked: %s", encoded)
	}
	delivery, err := service.QueueTest(context.Background(), "org_topoai", created.Record.ID, WebhookTestInput{EventType: "request.completed"})
	if err != nil || delivery.Status != "pending" || delivery.Trigger != "test" {
		t.Fatalf("test delivery=%+v err=%v", delivery, err)
	}
	if _, err := service.Disable(context.Background(), "org_topoai", created.Record.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.QueueTest(context.Background(), "org_topoai", created.Record.ID, WebhookTestInput{}); err != ErrInactive {
		t.Fatalf("disabled test error=%v", err)
	}
	if _, err := service.Enable(context.Background(), "org_topoai", created.Record.ID); err != nil {
		t.Fatal(err)
	}
	replayed, err := service.Replay(context.Background(), "org_topoai", "whd_dead_03")
	if err != nil || replayed.Trigger != "replay" || replayed.ReplayOfID == nil {
		t.Fatalf("replay=%+v err=%v", replayed, err)
	}
}

func TestWebhookDestinationAndSubscriptionValidation(t *testing.T) {
	service := NewWebhookService(NewMemoryRepository())
	_, err := service.Create(context.Background(), CreateWebhookInput{Name: "Insecure", Destination: "http://10.0.0.5/hook", Events: []string{"request.completed"}})
	var validation *ValidationError
	if err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("expected HTTPS validation, got %v (%T)", err, validation)
	}
	_, err = service.Create(context.Background(), CreateWebhookInput{Name: "Unknown", Destination: "https://example.com/hook", Events: []string{"unknown.event"}})
	if err == nil {
		t.Fatal("expected invalid event validation")
	}
}
