package httpapi

import (
	"github.com/topoai/aethergate/apps/api/internal/platform"
	"net/http"
)

type budgetHandler struct {
	service *platform.BudgetService
	source  string
}

func registerBudgetRoutes(mux *http.ServeMux, repository platform.Repository, source string) {
	repo, ok := repository.(platform.BudgetRepository)
	if !ok {
		return
	}
	h := &budgetHandler{service: platform.NewBudgetService(repo), source: source}
	mux.HandleFunc("GET /api/v1/budgets", h.list)
	mux.HandleFunc("POST /api/v1/budgets", h.create)
	mux.HandleFunc("POST /api/v1/budgets/evaluate", h.evaluate)
	mux.HandleFunc("POST /api/v1/budgets/{budgetID}/activate", h.activate)
	mux.HandleFunc("POST /api/v1/budgets/{budgetID}/pause", h.pause)
}
func (h *budgetHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context(), platform.BudgetFilter{OrganizationID: r.URL.Query().Get("organizationId"), Query: r.URL.Query().Get("q"), Status: r.URL.Query().Get("status"), ScopeType: r.URL.Query().Get("scopeType")})
	if err != nil {
		writePlatformError(w, err, "budgets_unavailable", "Budgets could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, listPayload(items, h.source))
}
func (h *budgetHandler) create(w http.ResponseWriter, r *http.Request) {
	var input platform.CreateBudgetInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	item, err := h.service.Create(r.Context(), input)
	if err != nil {
		writePlatformError(w, err, "budget_create_failed", "The budget could not be created.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": item, "meta": map[string]string{"source": h.source}})
}
func (h *budgetHandler) activate(w http.ResponseWriter, r *http.Request) { h.status(w, r, "active") }
func (h *budgetHandler) pause(w http.ResponseWriter, r *http.Request)    { h.status(w, r, "paused") }
func (h *budgetHandler) status(w http.ResponseWriter, r *http.Request, status string) {
	var item platform.Budget
	var err error
	if status == "active" {
		item, err = h.service.Activate(r.Context(), r.URL.Query().Get("organizationId"), r.PathValue("budgetID"))
	} else {
		item, err = h.service.Pause(r.Context(), r.URL.Query().Get("organizationId"), r.PathValue("budgetID"))
	}
	if err != nil {
		writePlatformError(w, err, "budget_status_failed", "Budget state could not be changed.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": item, "meta": map[string]string{"source": h.source}})
}
func (h *budgetHandler) evaluate(w http.ResponseWriter, r *http.Request) {
	var input platform.BudgetEvaluationInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	d, err := h.service.Evaluate(r.Context(), input)
	if err != nil {
		writePlatformError(w, err, "budget_evaluation_failed", "Budget evaluation failed.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": d, "meta": map[string]string{"source": h.source}})
}
