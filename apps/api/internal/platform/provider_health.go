package platform

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"
)

type ProviderHealthEvent struct {
	ID                  string    `json:"id"`
	OrganizationID      string    `json:"organizationId"`
	ProviderID          string    `json:"providerId"`
	ProviderName        string    `json:"providerName"`
	ProbeID             *string   `json:"probeId"`
	Source              string    `json:"source"`
	PreviousStatus      string    `json:"previousStatus"`
	Status              string    `json:"status"`
	Transition          bool      `json:"transition"`
	Success             bool      `json:"success"`
	RoutingEligible     bool      `json:"routingEligible"`
	RequestCount        int64     `json:"requestCount"`
	ErrorCount          int64     `json:"errorCount"`
	ErrorRate           float64   `json:"errorRate"`
	AverageLatencyMS    int       `json:"averageLatencyMs"`
	P95LatencyMS        int       `json:"p95LatencyMs"`
	HTTPStatus          *int      `json:"httpStatus"`
	ConsecutiveFailures int       `json:"consecutiveFailures"`
	Reason              string    `json:"reason"`
	ObservedAt          time.Time `json:"observedAt"`
}

type ProviderHealthProbe struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organizationId"`
	ProviderID     string     `json:"providerId"`
	ProviderName   string     `json:"providerName"`
	Status         string     `json:"status"`
	Region         string     `json:"region"`
	Model          string     `json:"model"`
	RequestedBy    string     `json:"requestedBy"`
	RequestedAt    time.Time  `json:"requestedAt"`
	StartedAt      *time.Time `json:"startedAt"`
	CompletedAt    *time.Time `json:"completedAt"`
	EventID        *string    `json:"eventId"`
	ErrorMessage   string     `json:"errorMessage"`
}

type ProviderHealthFilter struct {
	OrganizationID string
	ProviderID     string
	Status         string
	Source         string
}

type ProviderHealthProbeFilter struct {
	OrganizationID string
	ProviderID     string
	Status         string
}

type QueueProviderProbeInput struct {
	OrganizationID string `json:"organizationId"`
	Region         string `json:"region"`
	Model          string `json:"model"`
	RequestedBy    string `json:"requestedBy"`
}

type RecordProviderHealthInput struct {
	OrganizationID   string  `json:"organizationId"`
	ProbeID          *string `json:"probeId"`
	Source           string  `json:"source"`
	Success          bool    `json:"success"`
	RequestCount     int64   `json:"requestCount"`
	ErrorCount       int64   `json:"errorCount"`
	AverageLatencyMS int     `json:"averageLatencyMs"`
	P95LatencyMS     int     `json:"p95LatencyMs"`
	HTTPStatus       *int    `json:"httpStatus"`
	Message          string  `json:"message"`
}

type SetProviderMaintenanceInput struct {
	OrganizationID string  `json:"organizationId"`
	Enabled        bool    `json:"enabled"`
	Until          *string `json:"until"`
	Reason         string  `json:"reason"`
}

type ProviderHealthRepository interface {
	ProviderRepository
	GetProvider(context.Context, string, string) (ProviderConnection, error)
	ListProviderHealthEvents(context.Context, ProviderHealthFilter) ([]ProviderHealthEvent, error)
	ListProviderHealthProbes(context.Context, ProviderHealthProbeFilter) ([]ProviderHealthProbe, error)
	CreateProviderHealthProbe(context.Context, ProviderHealthProbe) (ProviderHealthProbe, error)
	RecordProviderHealth(context.Context, ProviderConnection, ProviderHealthEvent) (ProviderHealthEvent, error)
	UpdateProviderMaintenance(context.Context, ProviderConnection) (ProviderConnection, error)
}

type ProviderHealthService struct {
	repository ProviderHealthRepository
	now        func() time.Time
}

func NewProviderHealthService(repository ProviderHealthRepository) *ProviderHealthService {
	return &ProviderHealthService{repository: repository, now: time.Now}
}

func (s *ProviderHealthService) ListEvents(ctx context.Context, filter ProviderHealthFilter) ([]ProviderHealthEvent, error) {
	filter.OrganizationID = defaultOrganization(filter.OrganizationID)
	return s.repository.ListProviderHealthEvents(ctx, filter)
}

func (s *ProviderHealthService) ListProbes(ctx context.Context, filter ProviderHealthProbeFilter) ([]ProviderHealthProbe, error) {
	filter.OrganizationID = defaultOrganization(filter.OrganizationID)
	return s.repository.ListProviderHealthProbes(ctx, filter)
}

func (s *ProviderHealthService) QueueProbe(ctx context.Context, providerID string, input QueueProviderProbeInput) (ProviderHealthProbe, error) {
	input.OrganizationID = defaultOrganization(input.OrganizationID)
	provider, err := s.repository.GetProvider(ctx, input.OrganizationID, strings.TrimSpace(providerID))
	if err != nil {
		return ProviderHealthProbe{}, err
	}
	if provider.CredentialState != "configured" {
		return ProviderHealthProbe{}, &ValidationError{Code: "provider_probe_credentials_missing", Message: "Active probes require configured provider credentials."}
	}
	input.Region = strings.TrimSpace(input.Region)
	if input.Region == "" {
		input.Region = "automatic"
	}
	input.Model = strings.TrimSpace(input.Model)
	if input.Model == "" {
		input.Model = "provider-default"
	}
	input.RequestedBy = strings.TrimSpace(input.RequestedBy)
	if input.RequestedBy == "" {
		input.RequestedBy = "holden@topoai.dev"
	}
	id, err := randomIdentifier("probe_", 9)
	if err != nil {
		return ProviderHealthProbe{}, err
	}
	return s.repository.CreateProviderHealthProbe(ctx, ProviderHealthProbe{
		ID: id, OrganizationID: input.OrganizationID, ProviderID: provider.ID, ProviderName: provider.Name,
		Status: "queued", Region: input.Region, Model: input.Model, RequestedBy: input.RequestedBy,
		RequestedAt: s.now().UTC(),
	})
}

func (s *ProviderHealthService) Record(ctx context.Context, providerID string, input RecordProviderHealthInput) (ProviderHealthEvent, error) {
	input.OrganizationID = defaultOrganization(input.OrganizationID)
	input.Source = strings.TrimSpace(input.Source)
	if !slices.Contains([]string{"active_probe", "passive_telemetry"}, input.Source) {
		return ProviderHealthEvent{}, &ValidationError{Code: "provider_health_source_invalid", Message: "Health source must be active_probe or passive_telemetry."}
	}
	if input.RequestCount < 0 || input.ErrorCount < 0 || input.ErrorCount > input.RequestCount || input.AverageLatencyMS < 0 || input.P95LatencyMS < 0 {
		return ProviderHealthEvent{}, &ValidationError{Code: "provider_health_metrics_invalid", Message: "Provider health metrics are outside the supported range."}
	}
	if input.Source == "passive_telemetry" && input.RequestCount == 0 {
		return ProviderHealthEvent{}, &ValidationError{Code: "provider_health_sample_empty", Message: "Passive telemetry requires at least one request."}
	}
	if input.Source == "active_probe" && input.RequestCount == 0 {
		input.RequestCount = 1
		if !input.Success {
			input.ErrorCount = 1
		}
	}
	if input.HTTPStatus != nil && (*input.HTTPStatus < 100 || *input.HTTPStatus > 599) {
		return ProviderHealthEvent{}, &ValidationError{Code: "provider_health_status_invalid", Message: "Observed HTTP status must be between 100 and 599."}
	}
	provider, err := s.repository.GetProvider(ctx, input.OrganizationID, strings.TrimSpace(providerID))
	if err != nil {
		return ProviderHealthEvent{}, err
	}
	now := s.now().UTC()
	errorRate := float64(input.ErrorCount) / float64(input.RequestCount) * 100
	previousStatus := provider.Status
	status, failures, reason := classifyProviderHealth(provider, input, errorRate, now)
	routingEligible := status == "healthy" && provider.CredentialState == "configured"
	provider.Status = status
	provider.RoutingEligible = routingEligible
	provider.HealthSource = input.Source
	provider.HealthReason = reason
	provider.ConsecutiveFailures = failures
	provider.ErrorRate = errorRate
	provider.RequestCount24H = input.RequestCount
	provider.AverageLatencyMS = input.AverageLatencyMS
	provider.P95LatencyMS = input.P95LatencyMS
	provider.SuccessRate = 100 - errorRate
	provider.LastCheckedAt = &now
	if previousStatus != status {
		provider.LastTransitionAt = &now
	}
	id, err := randomIdentifier("phe_", 10)
	if err != nil {
		return ProviderHealthEvent{}, err
	}
	event := ProviderHealthEvent{
		ID: id, OrganizationID: provider.OrganizationID, ProviderID: provider.ID, ProviderName: provider.Name,
		ProbeID: input.ProbeID, Source: input.Source, PreviousStatus: previousStatus, Status: status,
		Transition: previousStatus != status, Success: input.Success, RoutingEligible: routingEligible,
		RequestCount: input.RequestCount, ErrorCount: input.ErrorCount, ErrorRate: errorRate,
		AverageLatencyMS: input.AverageLatencyMS, P95LatencyMS: input.P95LatencyMS,
		HTTPStatus: input.HTTPStatus, ConsecutiveFailures: failures, Reason: reason, ObservedAt: now,
	}
	return s.repository.RecordProviderHealth(ctx, provider, event)
}

func (s *ProviderHealthService) SetMaintenance(ctx context.Context, providerID string, input SetProviderMaintenanceInput) (ProviderConnection, error) {
	input.OrganizationID = defaultOrganization(input.OrganizationID)
	provider, err := s.repository.GetProvider(ctx, input.OrganizationID, strings.TrimSpace(providerID))
	if err != nil {
		return ProviderConnection{}, err
	}
	now := s.now().UTC()
	if !input.Enabled {
		provider.Status = "configuring"
		provider.RoutingEligible = false
		provider.MaintenanceUntil = nil
		provider.MaintenanceReason = ""
		provider.HealthSource = "manual"
		provider.HealthReason = "Maintenance ended; fresh health evidence is required before routing."
		provider.LastTransitionAt = &now
		return s.repository.UpdateProviderMaintenance(ctx, provider)
	}
	input.Reason = strings.TrimSpace(input.Reason)
	if input.Until == nil || strings.TrimSpace(*input.Until) == "" || input.Reason == "" {
		return ProviderConnection{}, &ValidationError{Code: "provider_maintenance_required", Message: "Maintenance end time and reason are required."}
	}
	until, err := time.Parse(time.RFC3339, strings.TrimSpace(*input.Until))
	if err != nil || !until.After(now) || until.After(now.Add(30*24*time.Hour)) {
		return ProviderConnection{}, &ValidationError{Code: "provider_maintenance_window_invalid", Message: "Maintenance must end in the future and within 30 days."}
	}
	until = until.UTC()
	provider.Status = "maintenance"
	provider.RoutingEligible = false
	provider.MaintenanceUntil = &until
	provider.MaintenanceReason = input.Reason
	provider.HealthSource = "manual"
	provider.HealthReason = "Routing suppressed during scheduled maintenance."
	provider.LastTransitionAt = &now
	return s.repository.UpdateProviderMaintenance(ctx, provider)
}

func classifyProviderHealth(provider ProviderConnection, input RecordProviderHealthInput, errorRate float64, now time.Time) (string, int, string) {
	if provider.MaintenanceUntil != nil && provider.MaintenanceUntil.After(now) {
		return "maintenance", provider.ConsecutiveFailures, "Observation recorded while routing remains suppressed by maintenance."
	}
	if input.Source == "active_probe" {
		if input.Success {
			return "healthy", 0, "Active probe succeeded."
		}
		failures := provider.ConsecutiveFailures + 1
		if failures >= 3 {
			return "offline", failures, fmt.Sprintf("Active probe failed %d consecutive times.", failures)
		}
		return "degraded", failures, fmt.Sprintf("Active probe failed; %d of 3 consecutive failures before offline.", failures)
	}
	if errorRate > 10 || input.P95LatencyMS > 15000 {
		return "offline", provider.ConsecutiveFailures, "Passive telemetry exceeded the offline error-rate or latency threshold."
	}
	if errorRate > 2 || input.P95LatencyMS > 5000 {
		return "degraded", provider.ConsecutiveFailures, "Passive telemetry exceeded the degraded error-rate or latency threshold."
	}
	return "healthy", 0, "Passive telemetry is within routing-safe thresholds."
}
