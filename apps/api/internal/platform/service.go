package platform

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"regexp"
	"slices"
	"strings"
	"time"
)

type Service struct {
	repository Repository
	now        func() time.Time
}

var nonSlugCharacter = regexp.MustCompile(`[^a-z0-9]+`)

func NewService(repository Repository) *Service {
	return &Service{repository: repository, now: time.Now}
}

func (s *Service) ListOrganizations(ctx context.Context, filter OrganizationFilter) ([]Organization, error) {
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Status = strings.TrimSpace(filter.Status)
	return s.repository.ListOrganizations(ctx, filter)
}

func (s *Service) GetOrganization(ctx context.Context, id string) (Organization, error) {
	return s.repository.GetOrganization(ctx, strings.TrimSpace(id))
}

func (s *Service) CreateOrganization(ctx context.Context, input CreateOrganizationInput) (Organization, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return Organization{}, &ValidationError{Code: "name_required", Message: "Organization name is required."}
	}
	if input.Plan == "" {
		input.Plan = "Evaluation"
	}
	if input.Region == "" {
		input.Region = "Singapore"
	}
	if input.Owner == "" {
		input.Owner = "holden@topoai.dev"
	}
	slug := normalizeSlug(input.Slug)
	if slug == "" {
		slug = normalizeSlug(input.Name)
	}
	if slug == "" {
		return Organization{}, &ValidationError{Code: "slug_required", Message: "A URL-safe organization slug is required."}
	}
	id, err := randomIdentifier("org_", 8)
	if err != nil {
		return Organization{}, err
	}
	return s.repository.CreateOrganization(ctx, Organization{
		ID: id, Name: input.Name, Slug: slug, Status: "provisioning", Plan: input.Plan,
		Region: input.Region, Members: 1, Owner: input.Owner, CreatedAt: s.now().UTC(),
	})
}

func (s *Service) ListAPIKeys(ctx context.Context, filter APIKeyFilter) ([]APIKey, error) {
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Status = strings.TrimSpace(filter.Status)
	return s.repository.ListAPIKeys(ctx, filter)
}

func (s *Service) CreateAPIKey(ctx context.Context, input CreateAPIKeyInput) (CreatedAPIKey, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Project = strings.TrimSpace(input.Project)
	input.Models = slices.DeleteFunc(slices.Clone(input.Models), func(model string) bool { return strings.TrimSpace(model) == "" })
	if input.Name == "" || input.Project == "" || len(input.Models) == 0 {
		return CreatedAPIKey{}, &ValidationError{Code: "key_scope_required", Message: "Name, project, and at least one model are required."}
	}
	if input.OrganizationID == "" {
		input.OrganizationID = "org_topoai"
	}
	if input.RPM <= 0 {
		input.RPM = 300
	}
	if input.TPM <= 0 {
		input.TPM = input.RPM * 2_000
	}
	if input.CreatedBy == "" {
		input.CreatedBy = "holden@topoai.dev"
	}
	expiresAt, err := parseExpiration(input.ExpiresAt)
	if err != nil {
		return CreatedAPIKey{}, &ValidationError{Code: "invalid_expiration", Message: "Expiration must be an RFC 3339 timestamp or YYYY-MM-DD date."}
	}

	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return CreatedAPIKey{}, err
	}
	secret := "ag_live_" + base64.RawURLEncoding.EncodeToString(randomBytes)
	id, err := randomIdentifier("key_", 12)
	if err != nil {
		return CreatedAPIKey{}, err
	}
	record, err := s.repository.CreateAPIKey(ctx, APIKey{
		ID: id, Organization: input.OrganizationID, Name: input.Name, Prefix: secret[:12],
		ProjectID: input.ProjectID, Project: input.Project, Status: "active", Models: slices.Clone(input.Models),
		RPM: input.RPM, TPM: input.TPM, CreatedBy: input.CreatedBy, CreatedAt: s.now().UTC(),
		ExpiresAt: expiresAt, SecretDigest: sha256.Sum256([]byte(secret)),
	})
	if err != nil {
		return CreatedAPIKey{}, err
	}
	return CreatedAPIKey{Record: record, Secret: secret}, nil
}

func (s *Service) RevokeAPIKey(ctx context.Context, id, actor string) (APIKey, error) {
	if actor == "" {
		actor = "holden@topoai.dev"
	}
	return s.repository.RevokeAPIKey(ctx, strings.TrimSpace(id), actor, s.now().UTC())
}

func normalizeSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = nonSlugCharacter.ReplaceAllString(value, "-")
	return strings.Trim(value, "-")
}

func randomIdentifier(prefix string, byteCount int) (string, error) {
	value := make([]byte, byteCount)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return prefix + strings.ToUpper(base64.RawURLEncoding.EncodeToString(value)), nil
}

func parseExpiration(value *string) (*time.Time, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*value)
	if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
		parsed = parsed.UTC()
		return &parsed, nil
	}
	parsed, err := time.Parse(time.DateOnly, trimmed)
	if err != nil {
		return nil, err
	}
	parsed = parsed.UTC()
	return &parsed, nil
}
