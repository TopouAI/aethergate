package httpapi

import (
	"errors"
	"net/http"

	"github.com/topoai/aethergate/apps/api/internal/platform"
)

type webhookHandler struct {
	service *platform.WebhookService
	source  string
}

func registerWebhookRoutes(mux *http.ServeMux, repository platform.Repository, source string) {
	webhookRepository, ok := repository.(platform.WebhookRepository)
	if !ok {
		return
	}
	handler := &webhookHandler{service: platform.NewWebhookService(webhookRepository), source: source}
	mux.HandleFunc("GET /api/v1/webhooks", handler.list)
	mux.HandleFunc("POST /api/v1/webhooks", handler.create)
	mux.HandleFunc("POST /api/v1/webhooks/{webhookID}/enable", handler.enable)
	mux.HandleFunc("POST /api/v1/webhooks/{webhookID}/disable", handler.disable)
	mux.HandleFunc("POST /api/v1/webhooks/{webhookID}/test", handler.test)
	mux.HandleFunc("GET /api/v1/webhook-deliveries", handler.deliveries)
	mux.HandleFunc("POST /api/v1/webhook-deliveries/{deliveryID}/retry", handler.retry)
	mux.HandleFunc("POST /api/v1/webhook-deliveries/{deliveryID}/replay", handler.replay)
}

func (h *webhookHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context(), platform.WebhookFilter{
		OrganizationID: r.URL.Query().Get("organizationId"), Query: r.URL.Query().Get("q"),
		Status: r.URL.Query().Get("status"), Event: r.URL.Query().Get("event"),
	})
	if err != nil {
		writePlatformError(w, err, "webhooks_unavailable", "Webhooks could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, listPayload(items, h.source))
}

func (h *webhookHandler) deliveries(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListDeliveries(r.Context(), platform.WebhookDeliveryFilter{
		OrganizationID: r.URL.Query().Get("organizationId"), WebhookID: r.URL.Query().Get("webhookId"),
		Status: r.URL.Query().Get("status"), EventType: r.URL.Query().Get("eventType"),
	})
	if err != nil {
		writePlatformError(w, err, "webhook_deliveries_unavailable", "Webhook deliveries could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, listPayload(items, h.source))
}

func (h *webhookHandler) create(w http.ResponseWriter, r *http.Request) {
	var input platform.CreateWebhookInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	created, err := h.service.Create(r.Context(), input)
	if err != nil {
		writePlatformError(w, err, "webhook_create_failed", "Webhook could not be created.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"data": created.Record, "signingSecret": created.SigningSecret,
		"meta": map[string]string{"source": h.source, "secretVisibility": "one-time", "delivery": "worker-queued"},
	})
}

func (h *webhookHandler) enable(w http.ResponseWriter, r *http.Request)  { h.changeStatus(w, r, true) }
func (h *webhookHandler) disable(w http.ResponseWriter, r *http.Request) { h.changeStatus(w, r, false) }

func (h *webhookHandler) changeStatus(w http.ResponseWriter, r *http.Request, enabled bool) {
	var endpoint platform.WebhookEndpoint
	var err error
	if enabled {
		endpoint, err = h.service.Enable(r.Context(), r.URL.Query().Get("organizationId"), r.PathValue("webhookID"))
	} else {
		endpoint, err = h.service.Disable(r.Context(), r.URL.Query().Get("organizationId"), r.PathValue("webhookID"))
	}
	if err != nil {
		writePlatformError(w, err, "webhook_status_failed", "Webhook state could not be changed.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": endpoint, "meta": map[string]string{"source": h.source}})
}

func (h *webhookHandler) test(w http.ResponseWriter, r *http.Request) {
	var input platform.WebhookTestInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	delivery, err := h.service.QueueTest(r.Context(), r.URL.Query().Get("organizationId"), r.PathValue("webhookID"), input)
	if err != nil {
		h.writeDeliveryError(w, err, "webhook_test_failed", "Webhook test could not be queued.")
		return
	}
	h.writeQueued(w, delivery)
}

func (h *webhookHandler) retry(w http.ResponseWriter, r *http.Request) {
	delivery, err := h.service.Retry(r.Context(), r.URL.Query().Get("organizationId"), r.PathValue("deliveryID"))
	if err != nil {
		h.writeDeliveryError(w, err, "webhook_retry_failed", "Webhook retry could not be queued.")
		return
	}
	h.writeQueued(w, delivery)
}

func (h *webhookHandler) replay(w http.ResponseWriter, r *http.Request) {
	delivery, err := h.service.Replay(r.Context(), r.URL.Query().Get("organizationId"), r.PathValue("deliveryID"))
	if err != nil {
		h.writeDeliveryError(w, err, "webhook_replay_failed", "Webhook replay could not be queued.")
		return
	}
	h.writeQueued(w, delivery)
}

func (h *webhookHandler) writeQueued(w http.ResponseWriter, delivery platform.WebhookDelivery) {
	writeJSON(w, http.StatusAccepted, map[string]any{
		"data": delivery,
		"meta": map[string]string{"source": h.source, "queueState": "accepted", "dispatchBoundary": "webhook-worker"},
	})
}

func (h *webhookHandler) writeDeliveryError(w http.ResponseWriter, err error, code, message string) {
	if errors.Is(err, platform.ErrInactive) {
		writeError(w, http.StatusConflict, "webhook_not_active", "Only an active webhook can queue deliveries.")
		return
	}
	writePlatformError(w, err, code, message)
}
