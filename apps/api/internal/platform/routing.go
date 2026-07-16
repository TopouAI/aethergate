package platform

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"
)

type RoutingTarget struct {
	ID           string `json:"id"`
	ProviderID   string `json:"providerId"`
	ProviderName string `json:"providerName"`
	Model        string `json:"model"`
	Priority     int    `json:"priority"`
	Weight       int    `json:"weight"`
	Enabled      bool   `json:"enabled"`
}

type RoutingPolicy struct {
	ID               string          `json:"id"`
	OrganizationID   string          `json:"organizationId"`
	Name             string          `json:"name"`
	Slug             string          `json:"slug"`
	Status           string          `json:"status"`
	Strategy         string          `json:"strategy"`
	ModelPattern     string          `json:"modelPattern"`
	MaxRetries       int             `json:"maxRetries"`
	RequestTimeoutMS int             `json:"requestTimeoutMs"`
	Targets          []RoutingTarget `json:"targets"`
	CreatedAt        time.Time       `json:"createdAt"`
	UpdatedAt        time.Time       `json:"updatedAt"`
}

type RoutingPolicyFilter struct {
	OrganizationID string
	Query          string
	Status         string
}

type CreateRoutingTargetInput struct {
	ProviderID string `json:"providerId"`
	Model      string `json:"model"`
	Priority   int    `json:"priority"`
	Weight     int    `json:"weight"`
	Enabled    bool   `json:"enabled"`
}

type CreateRoutingPolicyInput struct {
	OrganizationID   string                     `json:"organizationId"`
	Name             string                     `json:"name"`
	Slug             string                     `json:"slug"`
	Strategy         string                     `json:"strategy"`
	ModelPattern     string                     `json:"modelPattern"`
	MaxRetries       int                        `json:"maxRetries"`
	RequestTimeoutMS int                        `json:"requestTimeoutMs"`
	Targets          []CreateRoutingTargetInput `json:"targets"`
}

type RoutingRepository interface {
	ProviderRepository
	ListRoutingPolicies(context.Context, RoutingPolicyFilter) ([]RoutingPolicy, error)
	GetRoutingPolicy(context.Context, string, string) (RoutingPolicy, error)
	CreateRoutingPolicy(context.Context, RoutingPolicy) (RoutingPolicy, error)
	UpdateRoutingPolicyStatus(context.Context, string, string, string, time.Time) (RoutingPolicy, error)
}

type RoutingService struct {
	repository RoutingRepository
	now        func() time.Time
}

func NewRoutingService(repository RoutingRepository) *RoutingService {
	return &RoutingService{repository: repository, now: time.Now}
}

func (s *RoutingService) List(ctx context.Context, filter RoutingPolicyFilter) ([]RoutingPolicy, error) {
	filter.OrganizationID = defaultOrganization(filter.OrganizationID)
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Status = strings.TrimSpace(filter.Status)
	return s.repository.ListRoutingPolicies(ctx, filter)
}

func (s *RoutingService) Create(ctx context.Context, input CreateRoutingPolicyInput) (RoutingPolicy, error) {
	input.OrganizationID = defaultOrganization(input.OrganizationID)
	input.Name = strings.TrimSpace(input.Name)
	input.ModelPattern = strings.TrimSpace(input.ModelPattern)
	if input.Name == "" || input.ModelPattern == "" {
		return RoutingPolicy{}, &ValidationError{Code: "routing_policy_scope_required", Message: "Routing policy name and model pattern are required."}
	}
	if !slices.Contains([]string{"weighted", "priority", "latency"}, input.Strategy) {
		return RoutingPolicy{}, &ValidationError{Code: "routing_strategy_invalid", Message: "Routing strategy must be weighted, priority, or latency."}
	}
	if input.MaxRetries < 0 || input.MaxRetries > 5 {
		return RoutingPolicy{}, &ValidationError{Code: "routing_retries_invalid", Message: "Max retries must be between 0 and 5."}
	}
	if input.RequestTimeoutMS < 1000 || input.RequestTimeoutMS > 300000 {
		return RoutingPolicy{}, &ValidationError{Code: "routing_timeout_invalid", Message: "Request timeout must be between 1 and 300 seconds."}
	}
	providers, err := s.repository.ListProviders(ctx, ProviderFilter{OrganizationID: input.OrganizationID})
	if err != nil {
		return RoutingPolicy{}, err
	}
	providerByID := make(map[string]ProviderConnection, len(providers))
	for _, provider := range providers {
		providerByID[provider.ID] = provider
	}
	targets, err := s.validateTargets(input.Strategy, input.Targets, providerByID)
	if err != nil {
		return RoutingPolicy{}, err
	}
	id, err := randomIdentifier("route_", 9)
	if err != nil {
		return RoutingPolicy{}, err
	}
	slug := normalizeSlug(input.Slug)
	if slug == "" {
		slug = normalizeSlug(input.Name)
	}
	now := s.now().UTC()
	return s.repository.CreateRoutingPolicy(ctx, RoutingPolicy{ID: id, OrganizationID: input.OrganizationID, Name: input.Name, Slug: slug, Status: "draft", Strategy: input.Strategy, ModelPattern: input.ModelPattern, MaxRetries: input.MaxRetries, RequestTimeoutMS: input.RequestTimeoutMS, Targets: targets, CreatedAt: now, UpdatedAt: now})
}

func (s *RoutingService) Activate(ctx context.Context, organizationID, id string) (RoutingPolicy, error) {
	organizationID = defaultOrganization(organizationID)
	policy, err := s.repository.GetRoutingPolicy(ctx, organizationID, strings.TrimSpace(id))
	if err != nil {
		return RoutingPolicy{}, err
	}
	providers, err := s.repository.ListProviders(ctx, ProviderFilter{OrganizationID: organizationID})
	if err != nil {
		return RoutingPolicy{}, err
	}
	providerByID := make(map[string]ProviderConnection, len(providers))
	for _, provider := range providers {
		providerByID[provider.ID] = provider
	}
	for _, target := range policy.Targets {
		if !target.Enabled {
			continue
		}
		provider, found := providerByID[target.ProviderID]
		if !found || provider.Status != "healthy" || !provider.RoutingEligible || provider.CredentialState != "configured" {
			return RoutingPolicy{}, &ValidationError{Code: "routing_target_not_ready", Message: fmt.Sprintf("Provider %s is not healthy with configured credentials.", target.ProviderName)}
		}
	}
	return s.repository.UpdateRoutingPolicyStatus(ctx, organizationID, id, "active", s.now().UTC())
}

func (s *RoutingService) Pause(ctx context.Context, organizationID, id string) (RoutingPolicy, error) {
	return s.repository.UpdateRoutingPolicyStatus(ctx, defaultOrganization(organizationID), strings.TrimSpace(id), "paused", s.now().UTC())
}

func (s *RoutingService) validateTargets(strategy string, inputs []CreateRoutingTargetInput, providers map[string]ProviderConnection) ([]RoutingTarget, error) {
	if len(inputs) == 0 {
		return nil, &ValidationError{Code: "routing_targets_required", Message: "At least one routing target is required."}
	}
	targets := make([]RoutingTarget, 0, len(inputs))
	seen := make(map[string]struct{}, len(inputs))
	weight := 0
	enabled := 0
	for _, input := range inputs {
		input.ProviderID = strings.TrimSpace(input.ProviderID)
		input.Model = strings.TrimSpace(input.Model)
		provider, found := providers[input.ProviderID]
		if !found || input.Model == "" {
			return nil, &ValidationError{Code: "routing_target_invalid", Message: "Every target must reference an existing provider and model."}
		}
		key := input.ProviderID + "\x00" + input.Model
		if _, duplicate := seen[key]; duplicate {
			return nil, &ValidationError{Code: "routing_target_duplicate", Message: "Provider and model targets must be unique within a policy."}
		}
		seen[key] = struct{}{}
		if input.Priority < 1 || input.Priority > 100 || input.Weight < 0 || input.Weight > 100 {
			return nil, &ValidationError{Code: "routing_target_bounds_invalid", Message: "Target priority or weight is outside the supported range."}
		}
		if input.Enabled {
			enabled++
			weight += input.Weight
		}
		targetID, err := randomIdentifier("target_", 8)
		if err != nil {
			return nil, err
		}
		targets = append(targets, RoutingTarget{ID: targetID, ProviderID: input.ProviderID, ProviderName: provider.Name, Model: input.Model, Priority: input.Priority, Weight: input.Weight, Enabled: input.Enabled})
	}
	if enabled == 0 {
		return nil, &ValidationError{Code: "routing_target_enabled_required", Message: "At least one routing target must be enabled."}
	}
	if strategy == "weighted" && weight != 100 {
		return nil, &ValidationError{Code: "routing_weight_total_invalid", Message: "Enabled target weights must total 100 for weighted routing."}
	}
	return targets, nil
}
