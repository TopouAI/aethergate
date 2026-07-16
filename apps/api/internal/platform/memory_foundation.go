package platform

import (
	"context"
	"slices"
	"strings"
	"time"
)

func (r *MemoryRepository) ListWorkspaces(_ context.Context, organizationID string) ([]Workspace, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]Workspace, 0)
	for _, workspace := range r.workspaces {
		if workspace.OrganizationID == organizationID {
			workspace.Projects = countProjects(r.projects, workspace.ID)
			items = append(items, workspace)
		}
	}
	return items, nil
}

func (r *MemoryRepository) CreateWorkspace(_ context.Context, workspace Workspace) (Workspace, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, found := findOrganization(r.organizations, workspace.OrganizationID); !found {
		return Workspace{}, ErrNotFound
	}
	if slices.ContainsFunc(r.workspaces, func(existing Workspace) bool {
		return existing.OrganizationID == workspace.OrganizationID && strings.EqualFold(existing.Slug, workspace.Slug)
	}) {
		return Workspace{}, ErrConflict
	}
	r.workspaces = append([]Workspace{workspace}, r.workspaces...)
	return workspace, nil
}

func (r *MemoryRepository) ListProjects(_ context.Context, organizationID, workspaceID string) ([]Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]Project, 0)
	for _, project := range r.projects {
		if project.OrganizationID != organizationID || workspaceID != "" && project.WorkspaceID != workspaceID {
			continue
		}
		if workspace, found := findWorkspace(r.workspaces, project.WorkspaceID); found {
			project.Workspace = workspace.Name
		}
		items = append(items, project)
	}
	return items, nil
}

func (r *MemoryRepository) CreateProject(_ context.Context, project Project) (Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	workspace, found := findWorkspace(r.workspaces, project.WorkspaceID)
	if !found || workspace.OrganizationID != project.OrganizationID {
		return Project{}, ErrNotFound
	}
	if slices.ContainsFunc(r.projects, func(existing Project) bool {
		return existing.WorkspaceID == project.WorkspaceID && strings.EqualFold(existing.Slug, project.Slug)
	}) {
		return Project{}, ErrConflict
	}
	project.Workspace = workspace.Name
	r.projects = append([]Project{project}, r.projects...)
	return project, nil
}

func (r *MemoryRepository) ListMembers(_ context.Context, organizationID string) ([]Member, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]Member, 0)
	for _, member := range r.members {
		if member.OrganizationID == organizationID {
			items = append(items, member)
		}
	}
	return items, nil
}

func (r *MemoryRepository) CreateMember(_ context.Context, member Member, _ string, _ string) (Member, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, found := findOrganization(r.organizations, member.OrganizationID); !found {
		return Member{}, ErrNotFound
	}
	if slices.ContainsFunc(r.members, func(existing Member) bool {
		return existing.OrganizationID == member.OrganizationID && strings.EqualFold(existing.Email, member.Email)
	}) {
		return Member{}, ErrConflict
	}
	r.members = append([]Member{member}, r.members...)
	return member, nil
}

func (r *MemoryRepository) ListModels(_ context.Context, filter ModelFilter) ([]Model, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	query := strings.ToLower(filter.Query)
	items := make([]Model, 0)
	for _, model := range r.models {
		if filter.Provider != "" && filter.Provider != "all" && model.Provider != filter.Provider {
			continue
		}
		if filter.Status != "" && filter.Status != "all" && model.Status != filter.Status {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(model.ID+" "+model.DisplayName+" "+model.Provider), query) {
			continue
		}
		items = append(items, model)
	}
	return items, nil
}

func (r *MemoryRepository) UpsertModel(_ context.Context, model Model) (Model, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for index := range r.models {
		if r.models[index].ID == model.ID {
			model.CreatedAt = r.models[index].CreatedAt
			r.models[index] = model
			return model, nil
		}
	}
	r.models = append([]Model{model}, r.models...)
	return model, nil
}

func findWorkspace(items []Workspace, id string) (Workspace, bool) {
	for _, workspace := range items {
		if workspace.ID == id {
			return workspace, true
		}
	}
	return Workspace{}, false
}

func countProjects(items []Project, workspaceID string) int {
	count := 0
	for _, project := range items {
		if project.WorkspaceID == workspaceID {
			count++
		}
	}
	return count
}

func developmentWorkspaces() []Workspace {
	return []Workspace{
		{ID: "ws_engineering", OrganizationID: "org_topoai", Name: "Engineering", Slug: "engineering", Status: "active", Environment: "production", CreatedAt: time.Date(2025, 11, 19, 0, 0, 0, 0, time.UTC)},
		{ID: "ws_business", OrganizationID: "org_topoai", Name: "Business Operations", Slug: "business-operations", Status: "active", Environment: "shared", CreatedAt: time.Date(2026, 1, 12, 0, 0, 0, 0, time.UTC)},
		{ID: "ws_acme_production", OrganizationID: "org_acme", Name: "Production AI", Slug: "production-ai", Status: "active", Environment: "production", CreatedAt: time.Date(2026, 1, 8, 0, 0, 0, 0, time.UTC)},
	}
}

func developmentProjects() []Project {
	return []Project{
		{ID: "project_engineering_copilot", OrganizationID: "org_topoai", WorkspaceID: "ws_engineering", Name: "Engineering Copilot", Slug: "engineering-copilot", Status: "active", Owner: "li.ming@topoai.dev", BudgetUSD: 6000, MonthlyCostUSD: 2148.42, Requests: 184230, CreatedAt: time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)},
		{ID: "project_code_modernization", OrganizationID: "org_topoai", WorkspaceID: "ws_engineering", Name: "Code Modernization", Slug: "code-modernization", Status: "active", Owner: "wang.lei@topoai.dev", BudgetUSD: 3000, MonthlyCostUSD: 631.88, Requests: 67341, CreatedAt: time.Date(2026, 2, 8, 0, 0, 0, 0, time.UTC)},
		{ID: "project_finance_analyst", OrganizationID: "org_topoai", WorkspaceID: "ws_business", Name: "Finance Analyst", Slug: "finance-analyst", Status: "active", Owner: "finance@topoai.dev", BudgetUSD: 2500, MonthlyCostUSD: 994.16, Requests: 52101, CreatedAt: time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC)},
	}
}

func developmentMembers() []Member {
	lastActive := time.Date(2026, 7, 14, 13, 58, 0, 0, time.FixedZone("CST", 8*60*60))
	return []Member{
		{ID: "member_holden", OrganizationID: "org_topoai", Email: "holden@topoai.dev", DisplayName: "Holden", Status: "active", IdentityProvider: "oidc", Roles: []string{"owner"}, LastActiveAt: &lastActive, CreatedAt: time.Date(2025, 11, 18, 0, 0, 0, 0, time.UTC)},
		{ID: "member_liming", OrganizationID: "org_topoai", Email: "li.ming@topoai.dev", DisplayName: "Li Ming", Status: "active", IdentityProvider: "oidc", Roles: []string{"admin"}, LastActiveAt: &lastActive, CreatedAt: time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)},
		{ID: "member_wanglei", OrganizationID: "org_topoai", Email: "wang.lei@topoai.dev", DisplayName: "Wang Lei", Status: "active", IdentityProvider: "oidc", Roles: []string{"developer"}, LastActiveAt: &lastActive, CreatedAt: time.Date(2026, 2, 8, 0, 0, 0, 0, time.UTC)},
	}
}

func developmentModels() []Model {
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return []Model{
		{ID: "claude-sonnet-4", Provider: "Anthropic", DisplayName: "Claude Sonnet 4", Status: "active", ContextWindow: 200000, MaxOutputTokens: 64000, InputPricePerMillion: 3, OutputPricePerMillion: 15, SupportsTools: true, SupportsVision: true, SupportsJSON: true, Regions: []string{"us", "eu", "apac"}, CreatedAt: created},
		{ID: "gpt-5-mini", Provider: "OpenAI", DisplayName: "GPT-5 mini", Status: "active", ContextWindow: 400000, MaxOutputTokens: 128000, InputPricePerMillion: 0.25, OutputPricePerMillion: 2, SupportsTools: true, SupportsVision: true, SupportsJSON: true, Regions: []string{"us", "eu", "apac"}, CreatedAt: created},
		{ID: "gemini-2.5-pro", Provider: "Google", DisplayName: "Gemini 2.5 Pro", Status: "active", ContextWindow: 1048576, MaxOutputTokens: 65536, InputPricePerMillion: 1.25, OutputPricePerMillion: 10, SupportsTools: true, SupportsVision: true, SupportsJSON: true, Regions: []string{"us", "eu", "apac"}, CreatedAt: created},
		{ID: "deepseek-v3", Provider: "DeepSeek", DisplayName: "DeepSeek V3", Status: "active", ContextWindow: 128000, MaxOutputTokens: 8192, InputPricePerMillion: 0.27, OutputPricePerMillion: 1.10, SupportsTools: true, SupportsJSON: true, Regions: []string{"apac"}, CreatedAt: created},
	}
}
