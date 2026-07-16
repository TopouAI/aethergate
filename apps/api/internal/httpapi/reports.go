package httpapi

import (
	"net/http"

	"github.com/topoai/aethergate/apps/api/internal/platform"
)

type reportHandler struct {
	service *platform.ReportService
	source  string
}

func registerReportRoutes(mux *http.ServeMux, repository platform.Repository, source string) {
	reportRepository, ok := repository.(platform.ReportRepository)
	if !ok {
		return
	}
	handler := &reportHandler{service: platform.NewReportService(reportRepository), source: source}
	mux.HandleFunc("GET /api/v1/reports", handler.list)
	mux.HandleFunc("POST /api/v1/reports", handler.create)
	mux.HandleFunc("POST /api/v1/reports/{reportID}/activate", handler.activate)
	mux.HandleFunc("POST /api/v1/reports/{reportID}/pause", handler.pause)
	mux.HandleFunc("POST /api/v1/reports/{reportID}/run", handler.run)
	mux.HandleFunc("GET /api/v1/report-runs", handler.runs)
	mux.HandleFunc("POST /api/v1/report-runs/{runID}/retry", handler.retry)
}

func (h *reportHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context(), platform.ReportFilter{
		OrganizationID: r.URL.Query().Get("organizationId"), Query: r.URL.Query().Get("q"),
		Status: r.URL.Query().Get("status"), Template: r.URL.Query().Get("template"),
	})
	if err != nil {
		writePlatformError(w, err, "reports_unavailable", "Reports could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, listPayload(items, h.source))
}

func (h *reportHandler) runs(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListRuns(r.Context(), platform.ReportRunFilter{
		OrganizationID: r.URL.Query().Get("organizationId"), ReportID: r.URL.Query().Get("reportId"),
		Status: r.URL.Query().Get("status"), Trigger: r.URL.Query().Get("trigger"),
	})
	if err != nil {
		writePlatformError(w, err, "report_runs_unavailable", "Report runs could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, listPayload(items, h.source))
}

func (h *reportHandler) create(w http.ResponseWriter, r *http.Request) {
	var input platform.CreateReportInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	report, err := h.service.Create(r.Context(), input)
	if err != nil {
		writePlatformError(w, err, "report_create_failed", "Report could not be created.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": report, "meta": map[string]string{"source": h.source}})
}

func (h *reportHandler) activate(w http.ResponseWriter, r *http.Request) {
	report, err := h.service.Activate(r.Context(), r.URL.Query().Get("organizationId"), r.PathValue("reportID"))
	h.writeStatus(w, report, err)
}

func (h *reportHandler) pause(w http.ResponseWriter, r *http.Request) {
	report, err := h.service.Pause(r.Context(), r.URL.Query().Get("organizationId"), r.PathValue("reportID"))
	h.writeStatus(w, report, err)
}

func (h *reportHandler) writeStatus(w http.ResponseWriter, report platform.ReportSchedule, err error) {
	if err != nil {
		writePlatformError(w, err, "report_status_failed", "Report status could not be changed.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": report, "meta": map[string]string{"source": h.source}})
}

func (h *reportHandler) run(w http.ResponseWriter, r *http.Request) {
	var input platform.QueueReportRunInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	run, err := h.service.QueueRun(r.Context(), r.PathValue("reportID"), input)
	if err != nil {
		writePlatformError(w, err, "report_run_failed", "Report run could not be queued.")
		return
	}
	h.writeQueued(w, run)
}

func (h *reportHandler) retry(w http.ResponseWriter, r *http.Request) {
	var input struct {
		RequestedBy string `json:"requestedBy"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	run, err := h.service.RetryRun(r.Context(), r.URL.Query().Get("organizationId"), r.PathValue("runID"), input.RequestedBy)
	if err != nil {
		writePlatformError(w, err, "report_retry_failed", "Report retry could not be queued.")
		return
	}
	h.writeQueued(w, run)
}

func (h *reportHandler) writeQueued(w http.ResponseWriter, run platform.ReportRun) {
	writeJSON(w, http.StatusAccepted, map[string]any{
		"data": run,
		"meta": map[string]string{"source": h.source, "queueState": "accepted", "dispatchBoundary": "reports-worker"},
	})
}
