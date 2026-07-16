package platform

import (
	"context"
	"testing"
)

func TestRoutingPolicyLifecycle(t *testing.T) {
	service := NewRoutingService(NewMemoryRepository())
	created, err := service.Create(context.Background(), CreateRoutingPolicyInput{OrganizationID: "org_topoai", Name: "Support weighted", Strategy: "weighted", ModelPattern: "support/*", MaxRetries: 2, RequestTimeoutMS: 30000, Targets: []CreateRoutingTargetInput{{ProviderID: "provider_openai_primary", Model: "gpt-5-mini", Priority: 1, Weight: 70, Enabled: true}, {ProviderID: "provider_anthropic_primary", Model: "claude-sonnet-4", Priority: 2, Weight: 30, Enabled: true}}})
	if err != nil {
		t.Fatalf("create routing policy: %v", err)
	}
	activated, err := service.Activate(context.Background(), "org_topoai", created.ID)
	if err != nil || activated.Status != "active" {
		t.Fatalf("activate routing policy: policy=%+v err=%v", activated, err)
	}
	paused, err := service.Pause(context.Background(), "org_topoai", created.ID)
	if err != nil || paused.Status != "paused" {
		t.Fatalf("pause routing policy: policy=%+v err=%v", paused, err)
	}
}

func TestRoutingPolicyValidation(t *testing.T) {
	service := NewRoutingService(NewMemoryRepository())
	_, err := service.Create(context.Background(), CreateRoutingPolicyInput{Name: "Bad weights", Strategy: "weighted", ModelPattern: "*", RequestTimeoutMS: 30000, Targets: []CreateRoutingTargetInput{{ProviderID: "provider_openai_primary", Model: "gpt-5-mini", Priority: 1, Weight: 80, Enabled: true}}})
	if err == nil {
		t.Fatal("expected invalid weight total")
	}
	_, err = service.Activate(context.Background(), "org_topoai", "route_research_fallback")
	if err == nil {
		t.Fatal("expected degraded provider to block activation")
	}
}
