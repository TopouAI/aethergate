package httpapi

import (
	"github.com/topoai/aethergate/apps/api/internal/platform"
	"net/http"
)

type alertHandler struct {
	service *platform.AlertService
	source  string
}

func registerAlertRoutes(mux *http.ServeMux, r platform.Repository, source string) {
	repo, ok := r.(platform.AlertRepository)
	if !ok {
		return
	}
	h := &alertHandler{platform.NewAlertService(repo), source}
	mux.HandleFunc("GET /api/v1/alerts", h.list)
	mux.HandleFunc("POST /api/v1/alerts", h.create)
	mux.HandleFunc("GET /api/v1/alert-incidents", h.incidents)
	mux.HandleFunc("POST /api/v1/alerts/evaluate", h.evaluate)
	mux.HandleFunc("POST /api/v1/alerts/{alertID}/enable", h.enable)
	mux.HandleFunc("POST /api/v1/alerts/{alertID}/disable", h.disable)
}
func (h *alertHandler) list(w http.ResponseWriter, r *http.Request) {
	x, e := h.service.List(r.Context(), platform.AlertFilter{OrganizationID: r.URL.Query().Get("organizationId"), Query: r.URL.Query().Get("q"), Status: r.URL.Query().Get("status"), Severity: r.URL.Query().Get("severity")})
	if e != nil {
		writePlatformError(w, e, "alerts_unavailable", "Alerts could not be loaded.")
		return
	}
	writeJSON(w, 200, listPayload(x, h.source))
}
func (h *alertHandler) incidents(w http.ResponseWriter, r *http.Request) {
	x, e := h.service.ListIncidents(r.Context(), platform.AlertFilter{OrganizationID: r.URL.Query().Get("organizationId"), Status: r.URL.Query().Get("status"), Severity: r.URL.Query().Get("severity")})
	if e != nil {
		writePlatformError(w, e, "incidents_unavailable", "Incidents could not be loaded.")
		return
	}
	writeJSON(w, 200, listPayload(x, h.source))
}
func (h *alertHandler) create(w http.ResponseWriter, r *http.Request) {
	var i platform.CreateAlertInput
	if e := decodeJSON(r, &i); e != nil {
		writeError(w, 400, "invalid_json", "The request body must be valid JSON.")
		return
	}
	x, e := h.service.Create(r.Context(), i)
	if e != nil {
		writePlatformError(w, e, "alert_create_failed", "Alert could not be created.")
		return
	}
	writeJSON(w, 201, map[string]any{"data": x, "meta": map[string]string{"source": h.source}})
}
func (h *alertHandler) enable(w http.ResponseWriter, r *http.Request)  { h.status(w, r, true) }
func (h *alertHandler) disable(w http.ResponseWriter, r *http.Request) { h.status(w, r, false) }
func (h *alertHandler) status(w http.ResponseWriter, r *http.Request, on bool) {
	var x platform.AlertRule
	var e error
	if on {
		x, e = h.service.Enable(r.Context(), r.URL.Query().Get("organizationId"), r.PathValue("alertID"))
	} else {
		x, e = h.service.Disable(r.Context(), r.URL.Query().Get("organizationId"), r.PathValue("alertID"))
	}
	if e != nil {
		writePlatformError(w, e, "alert_status_failed", "Alert state could not be changed.")
		return
	}
	writeJSON(w, 200, map[string]any{"data": x, "meta": map[string]string{"source": h.source}})
}
func (h *alertHandler) evaluate(w http.ResponseWriter, r *http.Request) {
	var i platform.AlertEvaluationInput
	if e := decodeJSON(r, &i); e != nil {
		writeError(w, 400, "invalid_json", "The request body must be valid JSON.")
		return
	}
	x, e := h.service.Evaluate(r.Context(), i)
	if e != nil {
		writePlatformError(w, e, "alert_evaluation_failed", "Alert evaluation failed.")
		return
	}
	writeJSON(w, 200, map[string]any{"data": x, "meta": map[string]string{"source": h.source}})
}
