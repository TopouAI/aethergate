package platform

import (
	"context"
	"testing"
)

func TestAlertLifecycleAndEvaluation(t *testing.T) {
	s := NewAlertService(NewMemoryRepository())
	a, err := s.Create(context.Background(), CreateAlertInput{Name: "Token spike", Metric: "tokens", Operator: "gte", Threshold: 1000, Window: "5m", CooldownMinutes: 10, Severity: "warning", Channels: []string{"in_app"}, Filters: map[string]string{"project": "copilot"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Enable(context.Background(), "org_topoai", a.ID); err != nil {
		t.Fatal(err)
	}
	e, err := s.Evaluate(context.Background(), AlertEvaluationInput{OrganizationID: "org_topoai", Metric: "tokens", Value: 1200, Dimensions: map[string]string{"project": "copilot"}})
	if err != nil || !e.Triggered || len(e.Matches) != 1 {
		t.Fatalf("evaluation=%+v err=%v", e, err)
	}
}
