package httpapi

import (
	"net/http"
	"time"

	"github.com/topoai/aethergate/apps/api/internal/platform"
)

type auditHandler struct {
	service *platform.AuditService
	source  string
}

func registerAuditRoutes(mux *http.ServeMux, repository platform.Repository, source string) {
	repo, ok := repository.(platform.AuditRepository)
	if !ok {
		return
	}
	handler := &auditHandler{service: platform.NewAuditService(repo), source: source}
	mux.HandleFunc("GET /api/v1/audit-events", handler.list)
	mux.HandleFunc("POST /api/v1/audit-events", handler.append)
	mux.HandleFunc("GET /api/v1/audit-events/verify", handler.verify)
	mux.HandleFunc("GET /api/v1/audit-retention", handler.retention)
	mux.HandleFunc("PUT /api/v1/audit-retention", handler.updateRetention)
	mux.HandleFunc("GET /api/v1/audit-exports", handler.exports)
	mux.HandleFunc("POST /api/v1/audit-exports", handler.queueExport)
	mux.HandleFunc("POST /api/v1/audit-exports/{exportID}/retry", handler.retryExport)
}

func (h *auditHandler) list(w http.ResponseWriter, r *http.Request) {
	start, err := optionalAuditTime(r.URL.Query().Get("startAt"))
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "audit_time_invalid", "Audit startAt must be RFC3339.")
		return
	}
	end, err := optionalAuditTime(r.URL.Query().Get("endAt"))
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "audit_time_invalid", "Audit endAt must be RFC3339.")
		return
	}
	items, err := h.service.List(r.Context(), platform.AuditFilter{
		OrganizationID: r.URL.Query().Get("organizationId"), Query: r.URL.Query().Get("q"),
		Actor: r.URL.Query().Get("actor"), Action: r.URL.Query().Get("action"),
		ResourceType: r.URL.Query().Get("resourceType"), ResourceID: r.URL.Query().Get("resourceId"),
		Outcome: r.URL.Query().Get("outcome"), RiskLevel: r.URL.Query().Get("riskLevel"), StartAt: start, EndAt: end,
	})
	if err != nil {
		writePlatformError(w, err, "audit_events_unavailable", "Audit events could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, listPayload(items, h.source))
}

func (h *auditHandler) append(w http.ResponseWriter, r *http.Request) {
	var input platform.AppendAuditEventInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	event, err := h.service.Append(r.Context(), input)
	if err != nil {
		writePlatformError(w, err, "audit_append_failed", "Audit event could not be appended.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": event, "meta": map[string]any{"source": h.source, "storage": "append-only", "integrity": "sha256-chain"}})
}

func (h *auditHandler) verify(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.Verify(r.Context(), r.URL.Query().Get("organizationId"))
	if err != nil {
		writePlatformError(w, err, "audit_verify_failed", "Audit integrity could not be verified.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": result, "meta": map[string]string{"source": h.source, "algorithm": "sha256-chain"}})
}

func (h *auditHandler) retention(w http.ResponseWriter, r *http.Request) {
	policy, err := h.service.Retention(r.Context(), r.URL.Query().Get("organizationId"))
	if err != nil {
		writePlatformError(w, err, "audit_retention_unavailable", "Audit retention policy could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": policy, "meta": map[string]string{"source": h.source}})
}

func (h *auditHandler) updateRetention(w http.ResponseWriter, r *http.Request) {
	var input platform.UpsertAuditRetentionInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	policy, err := h.service.UpsertRetention(r.Context(), input)
	if err != nil {
		writePlatformError(w, err, "audit_retention_update_failed", "Audit retention policy could not be saved.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": policy, "meta": map[string]string{"source": h.source}})
}

func (h *auditHandler) exports(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListExports(r.Context(), platform.AuditExportFilter{OrganizationID: r.URL.Query().Get("organizationId"), Status: r.URL.Query().Get("status"), Format: r.URL.Query().Get("format")})
	if err != nil {
		writePlatformError(w, err, "audit_exports_unavailable", "Audit exports could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, listPayload(items, h.source))
}

func (h *auditHandler) queueExport(w http.ResponseWriter, r *http.Request) {
	var input platform.QueueAuditExportInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	export, err := h.service.QueueExport(r.Context(), input)
	if err != nil {
		writePlatformError(w, err, "audit_export_failed", "Audit export could not be queued.")
		return
	}
	h.writeQueued(w, export)
}

func (h *auditHandler) retryExport(w http.ResponseWriter, r *http.Request) {
	var input struct {
		RequestedBy string `json:"requestedBy"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	export, err := h.service.RetryExport(r.Context(), r.URL.Query().Get("organizationId"), r.PathValue("exportID"), input.RequestedBy)
	if err != nil {
		writePlatformError(w, err, "audit_export_retry_failed", "Audit export could not be retried.")
		return
	}
	h.writeQueued(w, export)
}

func (h *auditHandler) writeQueued(w http.ResponseWriter, export platform.AuditExport) {
	writeJSON(w, http.StatusAccepted, map[string]any{"data": export, "meta": map[string]string{"source": h.source, "queueState": "accepted", "dispatchBoundary": "audit-export-worker"}})
}

func optionalAuditTime(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	parsed = parsed.UTC()
	return &parsed, nil
}
