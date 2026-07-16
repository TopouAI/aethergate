package httpapi

import (
	"net/http"

	"github.com/topoai/aethergate/apps/api/internal/platform"
)

type providerHealthHandler struct {
	service *platform.ProviderHealthService
	source  string
}

func registerProviderHealthRoutes(mux *http.ServeMux, repository platform.Repository, source string) {
	healthRepository, ok := repository.(platform.ProviderHealthRepository)
	if !ok {
		return
	}
	handler := &providerHealthHandler{service: platform.NewProviderHealthService(healthRepository), source: source}
	mux.HandleFunc("GET /api/v1/provider-health-events", handler.events)
	mux.HandleFunc("GET /api/v1/provider-health-probes", handler.probes)
	mux.HandleFunc("POST /api/v1/providers/{providerID}/health/probes", handler.queueProbe)
	mux.HandleFunc("POST /api/v1/providers/{providerID}/health/observations", handler.recordObservation)
	mux.HandleFunc("POST /api/v1/providers/{providerID}/maintenance", handler.maintenance)
}

func (h *providerHealthHandler) events(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListEvents(r.Context(), platform.ProviderHealthFilter{
		OrganizationID: r.URL.Query().Get("organizationId"), ProviderID: r.URL.Query().Get("providerId"),
		Status: r.URL.Query().Get("status"), Source: r.URL.Query().Get("source"),
	})
	if err != nil {
		writePlatformError(w, err, "provider_health_events_unavailable", "Provider health events could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, listPayload(items, h.source))
}

func (h *providerHealthHandler) probes(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListProbes(r.Context(), platform.ProviderHealthProbeFilter{
		OrganizationID: r.URL.Query().Get("organizationId"), ProviderID: r.URL.Query().Get("providerId"),
		Status: r.URL.Query().Get("status"),
	})
	if err != nil {
		writePlatformError(w, err, "provider_health_probes_unavailable", "Provider health probes could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, listPayload(items, h.source))
}

func (h *providerHealthHandler) queueProbe(w http.ResponseWriter, r *http.Request) {
	var input platform.QueueProviderProbeInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	probe, err := h.service.QueueProbe(r.Context(), r.PathValue("providerID"), input)
	if err != nil {
		writePlatformError(w, err, "provider_probe_failed", "Provider probe could not be queued.")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"data": probe, "meta": map[string]string{"source": h.source, "queueState": "accepted", "dispatchBoundary": "provider-health-worker"}})
}

func (h *providerHealthHandler) recordObservation(w http.ResponseWriter, r *http.Request) {
	var input platform.RecordProviderHealthInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	event, err := h.service.Record(r.Context(), r.PathValue("providerID"), input)
	if err != nil {
		writePlatformError(w, err, "provider_observation_failed", "Provider health observation could not be recorded.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": event, "meta": map[string]string{"source": h.source, "routingDecision": "recomputed"}})
}

func (h *providerHealthHandler) maintenance(w http.ResponseWriter, r *http.Request) {
	var input platform.SetProviderMaintenanceInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	provider, err := h.service.SetMaintenance(r.Context(), r.PathValue("providerID"), input)
	if err != nil {
		writePlatformError(w, err, "provider_maintenance_failed", "Provider maintenance state could not be changed.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": provider, "meta": map[string]string{"source": h.source}})
}
