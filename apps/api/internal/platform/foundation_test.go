package platform

import (
	"context"
	"testing"
)

func TestFoundationResourceLifecycle(t *testing.T) {
	repository := NewMemoryRepository()
	service := NewFoundationService(repository)
	workspace, err := service.CreateWorkspace(context.Background(), CreateWorkspaceInput{Name: "AI Platform", Environment: "production"})
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	project, err := service.CreateProject(context.Background(), CreateProjectInput{WorkspaceID: workspace.ID, Name: "Support Copilot", BudgetUSD: 5000})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	member, err := service.InviteMember(context.Background(), InviteMemberInput{Email: "developer@example.com", Role: "developer"})
	if err != nil {
		t.Fatalf("invite member: %v", err)
	}
	if project.WorkspaceID != workspace.ID || member.Roles[0] != "developer" {
		t.Fatalf("unexpected resources: workspace=%+v project=%+v member=%+v", workspace, project, member)
	}
	workspaces, _ := service.ListWorkspaces(context.Background(), "org_topoai")
	projects, _ := service.ListProjects(context.Background(), "org_topoai", workspace.ID)
	members, _ := service.ListMembers(context.Background(), "org_topoai")
	if len(workspaces) == 0 || len(projects) != 1 || len(members) == 0 {
		t.Fatalf("unexpected list counts: workspaces=%d projects=%d members=%d", len(workspaces), len(projects), len(members))
	}
}

func TestFoundationRejectsInvalidMemberAndModel(t *testing.T) {
	service := NewFoundationService(NewMemoryRepository())
	if _, err := service.InviteMember(context.Background(), InviteMemberInput{Email: "not-an-email"}); err == nil {
		t.Fatal("expected invalid email error")
	}
	if _, err := service.UpsertModel(context.Background(), UpsertModelInput{ID: "model", Provider: "provider", DisplayName: "Model"}); err == nil {
		t.Fatal("expected invalid model limits error")
	}
}
