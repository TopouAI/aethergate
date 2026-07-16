package platform

import (
	"context"
	"net/url"
	"strings"
	"time"
)

type ProviderConnection struct {
	ID                  string     `json:"id"`
	OrganizationID      string     `json:"organizationId"`
	Name                string     `json:"name"`
	Provider            string     `json:"provider"`
	BaseURL             string     `json:"baseUrl"`
	Status              string     `json:"status"`
	CredentialState     string     `json:"credentialState"`
	Models              int        `json:"models"`
	P95LatencyMS        int        `json:"p95LatencyMs"`
	SuccessRate         float64    `json:"successRate"`
	LastCheckedAt       *time.Time `json:"lastCheckedAt"`
	RoutingEligible     bool       `json:"routingEligible"`
	HealthSource        string     `json:"healthSource"`
	HealthReason        string     `json:"healthReason"`
	ErrorRate           float64    `json:"errorRate"`
	RequestCount24H     int64      `json:"requestCount24h"`
	AverageLatencyMS    int        `json:"averageLatencyMs"`
	ConsecutiveFailures int        `json:"consecutiveFailures"`
	LastTransitionAt    *time.Time `json:"lastTransitionAt"`
	MaintenanceUntil    *time.Time `json:"maintenanceUntil"`
	MaintenanceReason   string     `json:"maintenanceReason"`
	CreatedAt           time.Time  `json:"createdAt"`
}

type ProviderFilter struct {
	OrganizationID string
	Query          string
	Status         string
}

type CreateProviderInput struct {
	OrganizationID string `json:"organizationId"`
	Name           string `json:"name"`
	Provider       string `json:"provider"`
	BaseURL        string `json:"baseUrl"`
}

type ProviderRepository interface {
	Repository
	ListProviders(context.Context, ProviderFilter) ([]ProviderConnection, error)
	CreateProvider(context.Context, ProviderConnection) (ProviderConnection, error)
}

type ProviderService struct {
	repository ProviderRepository
	now        func() time.Time
}

func NewProviderService(repository ProviderRepository) *ProviderService {
	return &ProviderService{repository: repository, now: time.Now}
}

func (s *ProviderService) ListProviders(ctx context.Context, filter ProviderFilter) ([]ProviderConnection, error) {
	filter.OrganizationID = defaultOrganization(filter.OrganizationID)
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Status = strings.TrimSpace(filter.Status)
	return s.repository.ListProviders(ctx, filter)
}

func (s *ProviderService) CreateProvider(ctx context.Context, input CreateProviderInput) (ProviderConnection, error) {
	input.OrganizationID = defaultOrganization(input.OrganizationID)
	input.Name = strings.TrimSpace(input.Name)
	input.Provider = strings.TrimSpace(input.Provider)
	input.BaseURL = strings.TrimRight(strings.TrimSpace(input.BaseURL), "/")
	if input.Name == "" || input.Provider == "" || input.BaseURL == "" {
		return ProviderConnection{}, &ValidationError{Code: "provider_definition_required", Message: "Provider name, type, and base URL are required."}
	}
	parsed, err := url.ParseRequestURI(input.BaseURL)
	if err != nil || parsed.Host == "" || parsed.Scheme != "https" && parsed.Scheme != "http" {
		return ProviderConnection{}, &ValidationError{Code: "provider_base_url_invalid", Message: "Provider base URL must be an absolute HTTP or HTTPS URL."}
	}
	id, err := randomIdentifier("provider_", 9)
	if err != nil {
		return ProviderConnection{}, err
	}
	return s.repository.CreateProvider(ctx, ProviderConnection{
		ID: id, OrganizationID: input.OrganizationID, Name: input.Name, Provider: input.Provider,
		BaseURL: input.BaseURL, Status: "configuring", CredentialState: "missing",
		RoutingEligible: false, HealthSource: "manual",
		HealthReason: "Awaiting configured credentials and fresh health evidence.", CreatedAt: s.now().UTC(),
	})
}
