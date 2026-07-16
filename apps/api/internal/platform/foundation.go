package platform

import (
	"context"
	"net/mail"
	"slices"
	"strings"
	"time"
)

type Workspace struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organizationId"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	Status         string    `json:"status"`
	Environment    string    `json:"environment"`
	Projects       int       `json:"projects"`
	CreatedAt      time.Time `json:"createdAt"`
}

type CreateWorkspaceInput struct {
	OrganizationID string `json:"organizationId"`
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	Environment    string `json:"environment"`
}

type Project struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organizationId"`
	WorkspaceID    string    `json:"workspaceId"`
	Workspace      string    `json:"workspace"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	Status         string    `json:"status"`
	Owner          string    `json:"owner"`
	BudgetUSD      float64   `json:"budgetUsd"`
	MonthlyCostUSD float64   `json:"monthlyCostUsd"`
	Requests       int64     `json:"requests"`
	CreatedAt      time.Time `json:"createdAt"`
}

type CreateProjectInput struct {
	OrganizationID string  `json:"organizationId"`
	WorkspaceID    string  `json:"workspaceId"`
	Name           string  `json:"name"`
	Slug           string  `json:"slug"`
	Owner          string  `json:"owner"`
	BudgetUSD      float64 `json:"budgetUsd"`
}

type Member struct {
	ID               string     `json:"id"`
	OrganizationID   string     `json:"organizationId"`
	Email            string     `json:"email"`
	DisplayName      string     `json:"displayName"`
	Status           string     `json:"status"`
	IdentityProvider string     `json:"identityProvider"`
	Roles            []string   `json:"roles"`
	LastActiveAt     *time.Time `json:"lastActiveAt"`
	CreatedAt        time.Time  `json:"createdAt"`
}

type InviteMemberInput struct {
	OrganizationID string `json:"organizationId"`
	Email          string `json:"email"`
	DisplayName    string `json:"displayName"`
	Role           string `json:"role"`
	InvitedBy      string `json:"invitedBy"`
}

type Model struct {
	ID                    string    `json:"id"`
	Provider              string    `json:"provider"`
	DisplayName           string    `json:"displayName"`
	Status                string    `json:"status"`
	ContextWindow         int       `json:"contextWindow"`
	MaxOutputTokens       int       `json:"maxOutputTokens"`
	InputPricePerMillion  float64   `json:"inputPricePerMillion"`
	OutputPricePerMillion float64   `json:"outputPricePerMillion"`
	SupportsTools         bool      `json:"supportsTools"`
	SupportsVision        bool      `json:"supportsVision"`
	SupportsJSON          bool      `json:"supportsJson"`
	Regions               []string  `json:"regions"`
	CreatedAt             time.Time `json:"createdAt"`
}

type ModelFilter struct {
	Query    string
	Provider string
	Status   string
}

type UpsertModelInput struct {
	ID                    string   `json:"id"`
	Provider              string   `json:"provider"`
	DisplayName           string   `json:"displayName"`
	Status                string   `json:"status"`
	ContextWindow         int      `json:"contextWindow"`
	MaxOutputTokens       int      `json:"maxOutputTokens"`
	InputPricePerMillion  float64  `json:"inputPricePerMillion"`
	OutputPricePerMillion float64  `json:"outputPricePerMillion"`
	SupportsTools         bool     `json:"supportsTools"`
	SupportsVision        bool     `json:"supportsVision"`
	SupportsJSON          bool     `json:"supportsJson"`
	Regions               []string `json:"regions"`
}

type FoundationRepository interface {
	Repository
	ListWorkspaces(context.Context, string) ([]Workspace, error)
	CreateWorkspace(context.Context, Workspace) (Workspace, error)
	ListProjects(context.Context, string, string) ([]Project, error)
	CreateProject(context.Context, Project) (Project, error)
	ListMembers(context.Context, string) ([]Member, error)
	CreateMember(context.Context, Member, string, string) (Member, error)
	ListModels(context.Context, ModelFilter) ([]Model, error)
	UpsertModel(context.Context, Model) (Model, error)
}

type FoundationService struct {
	repository FoundationRepository
	now        func() time.Time
}

func NewFoundationService(repository FoundationRepository) *FoundationService {
	return &FoundationService{repository: repository, now: time.Now}
}

func (s *FoundationService) ListWorkspaces(ctx context.Context, organizationID string) ([]Workspace, error) {
	return s.repository.ListWorkspaces(ctx, defaultOrganization(organizationID))
}

func (s *FoundationService) CreateWorkspace(ctx context.Context, input CreateWorkspaceInput) (Workspace, error) {
	input.OrganizationID = defaultOrganization(input.OrganizationID)
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return Workspace{}, &ValidationError{Code: "workspace_name_required", Message: "Workspace name is required."}
	}
	if input.Environment == "" {
		input.Environment = "production"
	}
	if !slices.Contains([]string{"development", "staging", "production", "shared"}, input.Environment) {
		return Workspace{}, &ValidationError{Code: "workspace_environment_invalid", Message: "Workspace environment is invalid."}
	}
	slug := normalizeSlug(input.Slug)
	if slug == "" {
		slug = normalizeSlug(input.Name)
	}
	id, err := randomIdentifier("ws_", 9)
	if err != nil {
		return Workspace{}, err
	}
	return s.repository.CreateWorkspace(ctx, Workspace{ID: id, OrganizationID: input.OrganizationID, Name: input.Name, Slug: slug, Status: "active", Environment: input.Environment, CreatedAt: s.now().UTC()})
}

func (s *FoundationService) ListProjects(ctx context.Context, organizationID, workspaceID string) ([]Project, error) {
	return s.repository.ListProjects(ctx, defaultOrganization(organizationID), strings.TrimSpace(workspaceID))
}

func (s *FoundationService) CreateProject(ctx context.Context, input CreateProjectInput) (Project, error) {
	input.OrganizationID = defaultOrganization(input.OrganizationID)
	input.Name = strings.TrimSpace(input.Name)
	input.WorkspaceID = strings.TrimSpace(input.WorkspaceID)
	if input.Name == "" || input.WorkspaceID == "" {
		return Project{}, &ValidationError{Code: "project_scope_required", Message: "Project name and workspace are required."}
	}
	if input.Owner == "" {
		input.Owner = "holden@topoai.dev"
	}
	if input.BudgetUSD < 0 {
		return Project{}, &ValidationError{Code: "project_budget_invalid", Message: "Project budget cannot be negative."}
	}
	slug := normalizeSlug(input.Slug)
	if slug == "" {
		slug = normalizeSlug(input.Name)
	}
	id, err := randomIdentifier("project_", 9)
	if err != nil {
		return Project{}, err
	}
	return s.repository.CreateProject(ctx, Project{ID: id, OrganizationID: input.OrganizationID, WorkspaceID: input.WorkspaceID, Name: input.Name, Slug: slug, Status: "active", Owner: input.Owner, BudgetUSD: input.BudgetUSD, CreatedAt: s.now().UTC()})
}

func (s *FoundationService) ListMembers(ctx context.Context, organizationID string) ([]Member, error) {
	return s.repository.ListMembers(ctx, defaultOrganization(organizationID))
}

func (s *FoundationService) InviteMember(ctx context.Context, input InviteMemberInput) (Member, error) {
	input.OrganizationID = defaultOrganization(input.OrganizationID)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if _, err := mail.ParseAddress(input.Email); err != nil {
		return Member{}, &ValidationError{Code: "member_email_invalid", Message: "A valid member email is required."}
	}
	if input.DisplayName == "" {
		input.DisplayName = strings.Split(input.Email, "@")[0]
	}
	if input.Role == "" {
		input.Role = "viewer"
	}
	if !slices.Contains([]string{"owner", "admin", "developer", "analyst", "billing", "viewer"}, input.Role) {
		return Member{}, &ValidationError{Code: "member_role_invalid", Message: "The selected member role is invalid."}
	}
	if input.InvitedBy == "" {
		input.InvitedBy = "holden@topoai.dev"
	}
	id, err := randomIdentifier("member_", 9)
	if err != nil {
		return Member{}, err
	}
	member := Member{ID: id, OrganizationID: input.OrganizationID, Email: input.Email, DisplayName: input.DisplayName, Status: "invited", IdentityProvider: "local", Roles: []string{input.Role}, CreatedAt: s.now().UTC()}
	return s.repository.CreateMember(ctx, member, input.Role, input.InvitedBy)
}

func (s *FoundationService) ListModels(ctx context.Context, filter ModelFilter) ([]Model, error) {
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Provider = strings.TrimSpace(filter.Provider)
	filter.Status = strings.TrimSpace(filter.Status)
	return s.repository.ListModels(ctx, filter)
}

func (s *FoundationService) UpsertModel(ctx context.Context, input UpsertModelInput) (Model, error) {
	input.ID = strings.TrimSpace(input.ID)
	input.Provider = strings.TrimSpace(input.Provider)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.ID == "" || input.Provider == "" || input.DisplayName == "" || input.ContextWindow <= 0 || input.MaxOutputTokens <= 0 {
		return Model{}, &ValidationError{Code: "model_definition_invalid", Message: "Model ID, provider, name, context window, and output limit are required."}
	}
	if input.Status == "" {
		input.Status = "active"
	}
	if !slices.Contains([]string{"active", "preview", "deprecated", "disabled"}, input.Status) {
		return Model{}, &ValidationError{Code: "model_status_invalid", Message: "Model status is invalid."}
	}
	if input.InputPricePerMillion < 0 || input.OutputPricePerMillion < 0 {
		return Model{}, &ValidationError{Code: "model_price_invalid", Message: "Model prices cannot be negative."}
	}
	return s.repository.UpsertModel(ctx, Model{
		ID: input.ID, Provider: input.Provider, DisplayName: input.DisplayName, Status: input.Status,
		ContextWindow: input.ContextWindow, MaxOutputTokens: input.MaxOutputTokens,
		InputPricePerMillion: input.InputPricePerMillion, OutputPricePerMillion: input.OutputPricePerMillion,
		SupportsTools: input.SupportsTools, SupportsVision: input.SupportsVision, SupportsJSON: input.SupportsJSON,
		Regions: slices.Clone(input.Regions), CreatedAt: s.now().UTC(),
	})
}

func defaultOrganization(value string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}
	return "org_topoai"
}
