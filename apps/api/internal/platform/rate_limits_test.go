package platform

import (
	"context"
	"testing"
)

func TestRateLimitLifecycleAndEvaluation(t *testing.T) {
	service := NewRateLimitService(NewMemoryRepository())
	created, err := service.Create(context.Background(), CreateRateLimitInput{Name: "Project request cap", ScopeType: "project", ScopeID: "project_engineering_copilot", Metric: "requests", Window: "minute", Limit: 100, Burst: 10, Action: "reject", Priority: 500})
	if err != nil {
		t.Fatalf("create rate limit: %v", err)
	}
	if _, err := service.Enforce(context.Background(), "org_topoai", created.ID); err != nil {
		t.Fatalf("enforce rate limit: %v", err)
	}
	decision, err := service.Evaluate(context.Background(), RateLimitEvaluationInput{OrganizationID: "org_topoai", ProjectID: "project_engineering_copilot", Metric: "requests", CurrentUsage: 105, RequestedUnits: 10})
	if err != nil || decision.Allowed || len(decision.Matches) != 1 || !decision.Matches[0].Exceeded {
		t.Fatalf("unexpected decision: %+v err=%v", decision, err)
	}
}

func TestRateLimitRejectsInvalidBounds(t *testing.T) {
	service := NewRateLimitService(NewMemoryRepository())
	if _, err := service.Create(context.Background(), CreateRateLimitInput{Name: "Invalid", ScopeType: "organization", ScopeID: "org_topoai", Metric: "requests", Window: "minute", Limit: 0, Action: "reject"}); err == nil {
		t.Fatal("expected invalid limit")
	}
}
