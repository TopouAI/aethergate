package httpapi

import (
	"net/http"

	"github.com/topoai/aethergate/apps/api/internal/platform"
)

type providerHandler struct {
	service *platform.ProviderService
	source  string
}

func registerProviderRoutes(mux *http.ServeMux, repository platform.Repository, source string) {
	providerRepository, ok := repository.(platform.ProviderRepository)
	if !ok {
		return
	}
	handler := &providerHandler{service: platform.NewProviderService(providerRepository), source: source}
	mux.HandleFunc("GET /api/v1/providers", handler.list)
	mux.HandleFunc("POST /api/v1/providers", handler.create)
}

func (h *providerHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListProviders(r.Context(), platform.ProviderFilter{OrganizationID: r.URL.Query().Get("organizationId"), Query: r.URL.Query().Get("q"), Status: r.URL.Query().Get("status")})
	if err != nil {
		writePlatformError(w, err, "providers_unavailable", "Providers could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, listPayload(items, h.source))
}

func (h *providerHandler) create(w http.ResponseWriter, r *http.Request) {
	var input platform.CreateProviderInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	item, err := h.service.CreateProvider(r.Context(), input)
	if err != nil {
		writePlatformError(w, err, "provider_create_failed", "The provider could not be created.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": item, "meta": map[string]string{"source": h.source}})
}
