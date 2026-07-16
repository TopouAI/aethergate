package platform

import (
	"context"
	"testing"
)

func TestBudgetLifecycleAndDecision(t *testing.T) {
	s := NewBudgetService(NewMemoryRepository())
	b, err := s.Create(context.Background(), CreateBudgetInput{Name: "Project block", ScopeType: "project", ScopeID: "project_engineering_copilot", Period: "monthly", LimitUSD: 1000, WarningPercent: 70, CriticalPercent: 90, Action: "block"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Activate(context.Background(), "org_topoai", b.ID); err != nil {
		t.Fatal(err)
	}
	d, err := s.Evaluate(context.Background(), BudgetEvaluationInput{OrganizationID: "org_topoai", ProjectID: "project_engineering_copilot", CurrentSpendUSD: 950, ProposedSpendUSD: 100, ElapsedPercent: 50})
	if err != nil || d.Allowed || len(d.Matches) < 1 {
		t.Fatalf("decision=%+v err=%v", d, err)
	}
}
func TestBudgetThresholdValidation(t *testing.T) {
	s := NewBudgetService(NewMemoryRepository())
	if _, err := s.Create(context.Background(), CreateBudgetInput{Name: "Bad", ScopeType: "organization", ScopeID: "org_topoai", Period: "monthly", LimitUSD: 1, WarningPercent: 95, CriticalPercent: 90, Action: "alert"}); err == nil {
		t.Fatal("expected invalid thresholds")
	}
}
