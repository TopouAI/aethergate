package httpapi

import (
	"net/http"

	"github.com/topoai/aethergate/apps/api/internal/platform"
)

type routingHandler struct {
	service *platform.RoutingService
	source  string
}

func registerRoutingRoutes(mux *http.ServeMux, repository platform.Repository, source string) {
	routingRepository, ok := repository.(platform.RoutingRepository)
	if !ok {
		return
	}
	handler := &routingHandler{service: platform.NewRoutingService(routingRepository), source: source}
	mux.HandleFunc("GET /api/v1/routing-policies", handler.list)
	mux.HandleFunc("POST /api/v1/routing-policies", handler.create)
	mux.HandleFunc("POST /api/v1/routing-policies/{policyID}/activate", handler.activate)
	mux.HandleFunc("POST /api/v1/routing-policies/{policyID}/pause", handler.pause)
}

func (h *routingHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context(), platform.RoutingPolicyFilter{OrganizationID: r.URL.Query().Get("organizationId"), Query: r.URL.Query().Get("q"), Status: r.URL.Query().Get("status")})
	if err != nil {
		writePlatformError(w, err, "routing_policies_unavailable", "Routing policies could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, listPayload(items, h.source))
}

func (h *routingHandler) create(w http.ResponseWriter, r *http.Request) {
	var input platform.CreateRoutingPolicyInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	item, err := h.service.Create(r.Context(), input)
	if err != nil {
		writePlatformError(w, err, "routing_policy_create_failed", "The routing policy could not be created.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": item, "meta": map[string]string{"source": h.source}})
}

func (h *routingHandler) activate(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.Activate(r.Context(), r.URL.Query().Get("organizationId"), r.PathValue("policyID"))
	if err != nil {
		writePlatformError(w, err, "routing_policy_activate_failed", "The routing policy could not be activated.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": item, "meta": map[string]string{"source": h.source}})
}

func (h *routingHandler) pause(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.Pause(r.Context(), r.URL.Query().Get("organizationId"), r.PathValue("policyID"))
	if err != nil {
		writePlatformError(w, err, "routing_policy_pause_failed", "The routing policy could not be paused.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": item, "meta": map[string]string{"source": h.source}})
}
