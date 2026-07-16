package httpapi

import (
	"net/http"

	"github.com/topoai/aethergate/apps/api/internal/platform"
)

type rateLimitHandler struct {
	service *platform.RateLimitService
	source  string
}

func registerRateLimitRoutes(mux *http.ServeMux, repository platform.Repository, source string) {
	rateLimitRepository, ok := repository.(platform.RateLimitRepository)
	if !ok {
		return
	}
	handler := &rateLimitHandler{service: platform.NewRateLimitService(rateLimitRepository), source: source}
	mux.HandleFunc("GET /api/v1/rate-limits", handler.list)
	mux.HandleFunc("POST /api/v1/rate-limits", handler.create)
	mux.HandleFunc("POST /api/v1/rate-limits/evaluate", handler.evaluate)
	mux.HandleFunc("POST /api/v1/rate-limits/{ruleID}/enforce", handler.enforce)
	mux.HandleFunc("POST /api/v1/rate-limits/{ruleID}/disable", handler.disable)
}

func (h *rateLimitHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context(), platform.RateLimitFilter{OrganizationID: r.URL.Query().Get("organizationId"), Query: r.URL.Query().Get("q"), Status: r.URL.Query().Get("status"), ScopeType: r.URL.Query().Get("scopeType")})
	if err != nil {
		writePlatformError(w, err, "rate_limits_unavailable", "Rate limits could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, listPayload(items, h.source))
}

func (h *rateLimitHandler) create(w http.ResponseWriter, r *http.Request) {
	var input platform.CreateRateLimitInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	item, err := h.service.Create(r.Context(), input)
	if err != nil {
		writePlatformError(w, err, "rate_limit_create_failed", "The rate limit could not be created.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": item, "meta": map[string]string{"source": h.source}})
}

func (h *rateLimitHandler) enforce(w http.ResponseWriter, r *http.Request) {
	h.changeStatus(w, r, "enforced")
}
func (h *rateLimitHandler) disable(w http.ResponseWriter, r *http.Request) {
	h.changeStatus(w, r, "disabled")
}

func (h *rateLimitHandler) changeStatus(w http.ResponseWriter, r *http.Request, status string) {
	var item platform.RateLimitRule
	var err error
	if status == "enforced" {
		item, err = h.service.Enforce(r.Context(), r.URL.Query().Get("organizationId"), r.PathValue("ruleID"))
	} else {
		item, err = h.service.Disable(r.Context(), r.URL.Query().Get("organizationId"), r.PathValue("ruleID"))
	}
	if err != nil {
		writePlatformError(w, err, "rate_limit_status_failed", "The rate-limit state could not be changed.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": item, "meta": map[string]string{"source": h.source}})
}

func (h *rateLimitHandler) evaluate(w http.ResponseWriter, r *http.Request) {
	var input platform.RateLimitEvaluationInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	decision, err := h.service.Evaluate(r.Context(), input)
	if err != nil {
		writePlatformError(w, err, "rate_limit_evaluation_failed", "The rate-limit evaluation could not be completed.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": decision, "meta": map[string]string{"source": h.source}})
}
