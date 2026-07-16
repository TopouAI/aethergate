package platform

import (
	"context"
	"testing"
	"time"
)

func TestProviderHealthFailureDebounceAndRecovery(t *testing.T) {
	repository := NewMemoryRepository()
	service := NewProviderHealthService(repository)
	probe, err := service.QueueProbe(context.Background(), "provider_openai_primary", QueueProviderProbeInput{OrganizationID: "org_topoai", Region: "apac", Model: "gpt-5-mini"})
	if err != nil || probe.Status != "queued" {
		t.Fatalf("probe=%+v err=%v", probe, err)
	}
	status := 503
	for attempt := 1; attempt <= 3; attempt++ {
		event, err := service.Record(context.Background(), "provider_openai_primary", RecordProviderHealthInput{OrganizationID: "org_topoai", Source: "active_probe", Success: false, HTTPStatus: &status})
		if err != nil {
			t.Fatal(err)
		}
		expected := "degraded"
		if attempt == 3 {
			expected = "offline"
		}
		if event.Status != expected || event.ConsecutiveFailures != attempt || event.RoutingEligible {
			t.Fatalf("attempt %d event=%+v", attempt, event)
		}
	}
	event, err := service.Record(context.Background(), "provider_openai_primary", RecordProviderHealthInput{OrganizationID: "org_topoai", ProbeID: &probe.ID, Source: "active_probe", Success: true, P95LatencyMS: 800, AverageLatencyMS: 420, HTTPStatus: intPointer(200)})
	if err != nil || event.Status != "healthy" || !event.RoutingEligible || event.ConsecutiveFailures != 0 {
		t.Fatalf("recovery=%+v err=%v", event, err)
	}
}

func TestProviderMaintenanceSuppressesRoutingUntilFreshEvidence(t *testing.T) {
	service := NewProviderHealthService(NewMemoryRepository())
	now := time.Date(2026, 7, 15, 1, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	until := now.Add(2 * time.Hour).Format(time.RFC3339)
	provider, err := service.SetMaintenance(context.Background(), "provider_openai_primary", SetProviderMaintenanceInput{OrganizationID: "org_topoai", Enabled: true, Until: &until, Reason: "Regional network maintenance"})
	if err != nil || provider.Status != "maintenance" || provider.RoutingEligible {
		t.Fatalf("maintenance=%+v err=%v", provider, err)
	}
	event, err := service.Record(context.Background(), provider.ID, RecordProviderHealthInput{OrganizationID: "org_topoai", Source: "active_probe", Success: true, HTTPStatus: intPointer(200)})
	if err != nil || event.Status != "maintenance" || event.RoutingEligible {
		t.Fatalf("suppressed=%+v err=%v", event, err)
	}
	provider, err = service.SetMaintenance(context.Background(), provider.ID, SetProviderMaintenanceInput{OrganizationID: "org_topoai", Enabled: false})
	if err != nil || provider.Status != "configuring" || provider.RoutingEligible {
		t.Fatalf("ended=%+v err=%v", provider, err)
	}
}

func intPointer(value int) *int { return &value }
