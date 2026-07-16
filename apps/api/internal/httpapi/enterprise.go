package httpapi

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/topoai/aethergate/apps/api/internal/platform"
)

type apiKeyRecord = platform.APIKey

type enterpriseHandler struct {
	service *platform.Service
	source  string
}

func registerEnterpriseRoutes(mux *http.ServeMux, logger *slog.Logger, repository platform.Repository, source string) {
	handler := &enterpriseHandler{service: platform.NewService(repository), source: source}
	mux.HandleFunc("GET /api/v1/organizations", handler.listOrganizations)
	mux.HandleFunc("POST /api/v1/organizations", handler.createOrganization)
	mux.HandleFunc("GET /api/v1/organizations/{organizationID}", handler.getOrganization)
	mux.HandleFunc("GET /api/v1/api-keys", handler.listAPIKeys)
	mux.HandleFunc("POST /api/v1/api-keys", handler.createAPIKey)
	mux.HandleFunc("POST /api/v1/api-keys/{apiKeyID}/revoke", handler.revokeAPIKey)
	logger.Info("enterprise routes registered", "repository", source)
}

func (h *enterpriseHandler) listOrganizations(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListOrganizations(r.Context(), platform.OrganizationFilter{
		Query: r.URL.Query().Get("q"), Status: r.URL.Query().Get("status"),
	})
	if err != nil {
		writePlatformError(w, err, "organizations_unavailable", "Organizations could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": map[string]any{"count": len(items), "source": h.source},
	})
}

func (h *enterpriseHandler) getOrganization(w http.ResponseWriter, r *http.Request) {
	organization, err := h.service.GetOrganization(r.Context(), r.PathValue("organizationID"))
	if err != nil {
		writePlatformError(w, err, "organization_not_found", "The organization does not exist.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": organization})
}

func (h *enterpriseHandler) createOrganization(w http.ResponseWriter, r *http.Request) {
	var input platform.CreateOrganizationInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	organization, err := h.service.CreateOrganization(r.Context(), input)
	if err != nil {
		writePlatformError(w, err, "organization_create_failed", "The organization could not be created.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"data": organization,
		"meta": map[string]string{"source": h.source},
	})
}

func (h *enterpriseHandler) listAPIKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := h.service.ListAPIKeys(r.Context(), platform.APIKeyFilter{
		OrganizationID: r.URL.Query().Get("organizationId"),
		Query:          r.URL.Query().Get("q"),
		Status:         r.URL.Query().Get("status"),
	})
	if err != nil {
		writePlatformError(w, err, "api_keys_unavailable", "API keys could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": keys,
		"meta": map[string]any{"count": len(keys), "source": h.source},
	})
}

func (h *enterpriseHandler) createAPIKey(w http.ResponseWriter, r *http.Request) {
	var input platform.CreateAPIKeyInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	created, err := h.service.CreateAPIKey(r.Context(), input)
	if err != nil {
		writePlatformError(w, err, "api_key_create_failed", "The API key could not be created.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"data":   created.Record,
		"secret": created.Secret,
		"meta":   map[string]string{"source": h.source, "secretVisibility": "one-time"},
	})
}

func (h *enterpriseHandler) revokeAPIKey(w http.ResponseWriter, r *http.Request) {
	key, err := h.service.RevokeAPIKey(r.Context(), r.PathValue("apiKeyID"), r.Header.Get("X-AetherGate-Actor"))
	if err != nil {
		writePlatformError(w, err, "api_key_revoke_failed", "The API key could not be revoked.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": key})
}

func writePlatformError(w http.ResponseWriter, err error, fallbackCode, fallbackMessage string) {
	var validation *platform.ValidationError
	switch {
	case errors.As(err, &validation):
		writeError(w, http.StatusUnprocessableEntity, validation.Code, validation.Message)
	case errors.Is(err, platform.ErrNotFound):
		writeError(w, http.StatusNotFound, fallbackCode, fallbackMessage)
	case errors.Is(err, platform.ErrConflict):
		writeError(w, http.StatusConflict, fallbackCode, fallbackMessage)
	case errors.Is(err, platform.ErrInactive):
		writeError(w, http.StatusConflict, "api_key_not_active", "Only an active API key can be revoked.")
	default:
		writeError(w, http.StatusInternalServerError, fallbackCode, fallbackMessage)
	}
}
