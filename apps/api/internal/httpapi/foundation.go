package httpapi

import (
	"net/http"

	"github.com/topoai/aethergate/apps/api/internal/platform"
)

type foundationHandler struct {
	service *platform.FoundationService
	source  string
}

func registerFoundationRoutes(mux *http.ServeMux, repository platform.Repository, source string) {
	foundationRepository, ok := repository.(platform.FoundationRepository)
	if !ok {
		return
	}
	handler := &foundationHandler{service: platform.NewFoundationService(foundationRepository), source: source}
	mux.HandleFunc("GET /api/v1/workspaces", handler.listWorkspaces)
	mux.HandleFunc("POST /api/v1/workspaces", handler.createWorkspace)
	mux.HandleFunc("GET /api/v1/projects", handler.listProjects)
	mux.HandleFunc("POST /api/v1/projects", handler.createProject)
	mux.HandleFunc("GET /api/v1/members", handler.listMembers)
	mux.HandleFunc("POST /api/v1/members", handler.inviteMember)
	mux.HandleFunc("GET /api/v1/models", handler.listModels)
	mux.HandleFunc("POST /api/v1/models", handler.upsertModel)
}

func (h *foundationHandler) listWorkspaces(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListWorkspaces(r.Context(), r.URL.Query().Get("organizationId"))
	if err != nil {
		writePlatformError(w, err, "workspaces_unavailable", "Workspaces could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, listPayload(items, h.source))
}

func (h *foundationHandler) createWorkspace(w http.ResponseWriter, r *http.Request) {
	var input platform.CreateWorkspaceInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	item, err := h.service.CreateWorkspace(r.Context(), input)
	if err != nil {
		writePlatformError(w, err, "workspace_create_failed", "The workspace could not be created.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": item, "meta": map[string]string{"source": h.source}})
}

func (h *foundationHandler) listProjects(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListProjects(r.Context(), r.URL.Query().Get("organizationId"), r.URL.Query().Get("workspaceId"))
	if err != nil {
		writePlatformError(w, err, "projects_unavailable", "Projects could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, listPayload(items, h.source))
}

func (h *foundationHandler) createProject(w http.ResponseWriter, r *http.Request) {
	var input platform.CreateProjectInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	item, err := h.service.CreateProject(r.Context(), input)
	if err != nil {
		writePlatformError(w, err, "project_create_failed", "The project could not be created.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": item, "meta": map[string]string{"source": h.source}})
}

func (h *foundationHandler) listMembers(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListMembers(r.Context(), r.URL.Query().Get("organizationId"))
	if err != nil {
		writePlatformError(w, err, "members_unavailable", "Members could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, listPayload(items, h.source))
}

func (h *foundationHandler) inviteMember(w http.ResponseWriter, r *http.Request) {
	var input platform.InviteMemberInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	item, err := h.service.InviteMember(r.Context(), input)
	if err != nil {
		writePlatformError(w, err, "member_invite_failed", "The member could not be invited.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": item, "meta": map[string]string{"source": h.source}})
}

func (h *foundationHandler) listModels(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListModels(r.Context(), platform.ModelFilter{Query: r.URL.Query().Get("q"), Provider: r.URL.Query().Get("provider"), Status: r.URL.Query().Get("status")})
	if err != nil {
		writePlatformError(w, err, "models_unavailable", "Models could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, listPayload(items, h.source))
}

func (h *foundationHandler) upsertModel(w http.ResponseWriter, r *http.Request) {
	var input platform.UpsertModelInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	item, err := h.service.UpsertModel(r.Context(), input)
	if err != nil {
		writePlatformError(w, err, "model_upsert_failed", "The model could not be saved.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": item, "meta": map[string]string{"source": h.source}})
}

func listPayload[T any](items []T, source string) map[string]any {
	return map[string]any{"data": items, "meta": map[string]any{"count": len(items), "source": source}}
}
