package platform

import "testing"

func TestAuthorizerEnforcesTenantAndScopeBoundaries(t *testing.T) {
	authorizer := Authorizer{}
	principal := Principal{
		ID: "member_1", Email: "developer@example.com", OrganizationID: "org_a",
		Bindings: []RoleBinding{{RoleKey: "developer", ScopeType: "workspace", ScopeID: "ws_1", Permissions: []string{"project:read", "api-key:create-self"}}},
	}
	tests := []struct {
		name     string
		action   string
		resource Scope
		want     bool
	}{
		{name: "workspace permission", action: "project:read", resource: Scope{OrganizationID: "org_a", WorkspaceID: "ws_1", ProjectID: "project_1"}, want: true},
		{name: "different workspace", action: "project:read", resource: Scope{OrganizationID: "org_a", WorkspaceID: "ws_2", ProjectID: "project_2"}, want: false},
		{name: "different tenant", action: "project:read", resource: Scope{OrganizationID: "org_b", WorkspaceID: "ws_1"}, want: false},
		{name: "ungranted action", action: "member:delete", resource: Scope{OrganizationID: "org_a", WorkspaceID: "ws_1"}, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := authorizer.Can(principal, test.action, test.resource); got != test.want {
				t.Fatalf("Can()=%v, want %v", got, test.want)
			}
		})
	}
}

func TestAuthorizerSupportsResourceWildcards(t *testing.T) {
	authorizer := Authorizer{}
	principal := Principal{
		ID: "member_admin", OrganizationID: "org_a",
		Bindings: []RoleBinding{{RoleKey: "admin", ScopeType: "organization", ScopeID: "org_a", Permissions: []string{"workspace:*", "project:*"}}},
	}
	if !authorizer.Can(principal, "project:delete", Scope{OrganizationID: "org_a", ProjectID: "project_1"}) {
		t.Fatal("expected project wildcard permission")
	}
	if authorizer.Can(principal, "billing:contract:update", Scope{OrganizationID: "org_a"}) {
		t.Fatal("unexpected billing permission")
	}
}
