package platform

import "strings"

type Scope struct {
	OrganizationID string
	WorkspaceID    string
	ProjectID      string
}

type RoleBinding struct {
	RoleKey     string
	ScopeType   string
	ScopeID     string
	Permissions []string
}

type Principal struct {
	ID             string
	Email          string
	OrganizationID string
	Bindings       []RoleBinding
}

type Authorizer struct{}

func (Authorizer) Can(principal Principal, action string, resource Scope) bool {
	if principal.ID == "" || principal.OrganizationID == "" || principal.OrganizationID != resource.OrganizationID {
		return false
	}
	for _, binding := range principal.Bindings {
		if !bindingCovers(binding, resource) {
			continue
		}
		for _, permission := range binding.Permissions {
			if permissionMatches(permission, action) {
				return true
			}
		}
	}
	return false
}

func bindingCovers(binding RoleBinding, resource Scope) bool {
	switch binding.ScopeType {
	case "organization":
		return binding.ScopeID == resource.OrganizationID
	case "workspace":
		return binding.ScopeID != "" && binding.ScopeID == resource.WorkspaceID
	case "project":
		return binding.ScopeID != "" && binding.ScopeID == resource.ProjectID
	default:
		return false
	}
}

func permissionMatches(permission, action string) bool {
	if permission == "*" || permission == action {
		return true
	}
	prefix, wildcard := strings.CutSuffix(permission, "*")
	return wildcard && strings.HasPrefix(action, prefix)
}
