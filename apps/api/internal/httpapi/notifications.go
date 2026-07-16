package httpapi

import (
	"net/http"

	"github.com/topoai/aethergate/apps/api/internal/platform"
)

type notificationHandler struct {
	service *platform.NotificationService
	source  string
}

func registerNotificationRoutes(mux *http.ServeMux, repository platform.Repository, source string) {
	repo, ok := repository.(platform.NotificationRepository)
	if !ok {
		return
	}
	handler := &notificationHandler{service: platform.NewNotificationService(repo), source: source}
	mux.HandleFunc("GET /api/v1/notifications", handler.list)
	mux.HandleFunc("POST /api/v1/notifications", handler.create)
	mux.HandleFunc("POST /api/v1/notifications/read-all", handler.readAll)
	mux.HandleFunc("POST /api/v1/notifications/{notificationID}/read", handler.read)
	mux.HandleFunc("POST /api/v1/notifications/{notificationID}/unread", handler.unread)
	mux.HandleFunc("POST /api/v1/notifications/{notificationID}/archive", handler.archive)
	mux.HandleFunc("GET /api/v1/notification-preferences", handler.preference)
	mux.HandleFunc("PUT /api/v1/notification-preferences", handler.updatePreference)
	mux.HandleFunc("GET /api/v1/notification-escalation-policies", handler.policies)
	mux.HandleFunc("POST /api/v1/notification-escalation-policies", handler.createPolicy)
	mux.HandleFunc("POST /api/v1/notification-escalation-policies/evaluate", handler.evaluatePolicy)
	mux.HandleFunc("POST /api/v1/notification-escalation-policies/{policyID}/activate", handler.activatePolicy)
	mux.HandleFunc("POST /api/v1/notification-escalation-policies/{policyID}/pause", handler.pausePolicy)
	mux.HandleFunc("GET /api/v1/notification-deliveries", handler.deliveries)
	mux.HandleFunc("POST /api/v1/notification-deliveries/{deliveryID}/retry", handler.retryDelivery)
}

func (h *notificationHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context(), platform.NotificationFilter{
		OrganizationID: r.URL.Query().Get("organizationId"), RecipientID: r.URL.Query().Get("recipientId"),
		Query: r.URL.Query().Get("q"), Status: r.URL.Query().Get("status"),
		Category: r.URL.Query().Get("category"), Severity: r.URL.Query().Get("severity"),
	})
	if err != nil {
		writePlatformError(w, err, "notifications_unavailable", "Notifications could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, listPayload(items, h.source))
}

func (h *notificationHandler) deliveries(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListDeliveries(r.Context(), platform.NotificationDeliveryFilter{
		OrganizationID: r.URL.Query().Get("organizationId"), RecipientID: r.URL.Query().Get("recipientId"),
		NotificationID: r.URL.Query().Get("notificationId"), Status: r.URL.Query().Get("status"), Channel: r.URL.Query().Get("channel"),
	})
	if err != nil {
		writePlatformError(w, err, "notification_deliveries_unavailable", "Notification deliveries could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, listPayload(items, h.source))
}

func (h *notificationHandler) create(w http.ResponseWriter, r *http.Request) {
	var input platform.CreateNotificationInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	dispatch, err := h.service.Create(r.Context(), input)
	if err != nil {
		writePlatformError(w, err, "notification_create_failed", "Notification could not be created.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"data": dispatch.Notification,
		"meta": map[string]any{"source": h.source, "externalDeliveries": len(dispatch.Deliveries), "dispatchBoundary": "notifications-worker"},
	})
}

func (h *notificationHandler) read(w http.ResponseWriter, r *http.Request) { h.status(w, r, "read") }
func (h *notificationHandler) unread(w http.ResponseWriter, r *http.Request) {
	h.status(w, r, "unread")
}
func (h *notificationHandler) archive(w http.ResponseWriter, r *http.Request) {
	h.status(w, r, "archive")
}

func (h *notificationHandler) status(w http.ResponseWriter, r *http.Request, action string) {
	organizationID := r.URL.Query().Get("organizationId")
	recipientID := r.URL.Query().Get("recipientId")
	id := r.PathValue("notificationID")
	var notification platform.Notification
	var err error
	switch action {
	case "read":
		notification, err = h.service.MarkRead(r.Context(), organizationID, recipientID, id)
	case "unread":
		notification, err = h.service.MarkUnread(r.Context(), organizationID, recipientID, id)
	default:
		notification, err = h.service.Archive(r.Context(), organizationID, recipientID, id)
	}
	if err != nil {
		writePlatformError(w, err, "notification_status_failed", "Notification state could not be changed.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": notification, "meta": map[string]string{"source": h.source}})
}

func (h *notificationHandler) readAll(w http.ResponseWriter, r *http.Request) {
	var input struct {
		OrganizationID string `json:"organizationId"`
		RecipientID    string `json:"recipientId"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	count, err := h.service.MarkAllRead(r.Context(), input.OrganizationID, input.RecipientID)
	if err != nil {
		writePlatformError(w, err, "notification_read_all_failed", "Notifications could not be marked as read.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]int64{"updated": count}, "meta": map[string]string{"source": h.source}})
}

func (h *notificationHandler) preference(w http.ResponseWriter, r *http.Request) {
	preference, err := h.service.Preference(r.Context(), r.URL.Query().Get("organizationId"), r.URL.Query().Get("recipientId"))
	if err != nil {
		writePlatformError(w, err, "notification_preference_unavailable", "Notification preference could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": preference, "meta": map[string]string{"source": h.source}})
}

func (h *notificationHandler) updatePreference(w http.ResponseWriter, r *http.Request) {
	var input platform.UpsertNotificationPreferenceInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	preference, err := h.service.UpsertPreference(r.Context(), input)
	if err != nil {
		writePlatformError(w, err, "notification_preference_update_failed", "Notification preference could not be saved.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": preference, "meta": map[string]string{"source": h.source}})
}

func (h *notificationHandler) policies(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListPolicies(r.Context(), platform.NotificationPolicyFilter{
		OrganizationID: r.URL.Query().Get("organizationId"), Query: r.URL.Query().Get("q"),
		Status: r.URL.Query().Get("status"), Category: r.URL.Query().Get("category"),
	})
	if err != nil {
		writePlatformError(w, err, "notification_policies_unavailable", "Notification escalation policies could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, listPayload(items, h.source))
}

func (h *notificationHandler) createPolicy(w http.ResponseWriter, r *http.Request) {
	var input platform.CreateNotificationEscalationPolicyInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	policy, err := h.service.CreatePolicy(r.Context(), input)
	if err != nil {
		writePlatformError(w, err, "notification_policy_create_failed", "Notification escalation policy could not be created.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": policy, "meta": map[string]string{"source": h.source}})
}

func (h *notificationHandler) activatePolicy(w http.ResponseWriter, r *http.Request) {
	h.policyStatus(w, r, true)
}

func (h *notificationHandler) pausePolicy(w http.ResponseWriter, r *http.Request) {
	h.policyStatus(w, r, false)
}

func (h *notificationHandler) policyStatus(w http.ResponseWriter, r *http.Request, active bool) {
	var policy platform.NotificationEscalationPolicy
	var err error
	if active {
		policy, err = h.service.ActivatePolicy(r.Context(), r.URL.Query().Get("organizationId"), r.PathValue("policyID"))
	} else {
		policy, err = h.service.PausePolicy(r.Context(), r.URL.Query().Get("organizationId"), r.PathValue("policyID"))
	}
	if err != nil {
		writePlatformError(w, err, "notification_policy_status_failed", "Notification escalation policy state could not be changed.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": policy, "meta": map[string]string{"source": h.source}})
}

func (h *notificationHandler) evaluatePolicy(w http.ResponseWriter, r *http.Request) {
	var input platform.EvaluateNotificationEscalationInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	evaluation, err := h.service.EvaluateEscalation(r.Context(), input)
	if err != nil {
		writePlatformError(w, err, "notification_policy_evaluation_failed", "Notification escalation policy could not be evaluated.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": evaluation, "meta": map[string]string{"source": h.source, "mode": "dry-run"}})
}

func (h *notificationHandler) retryDelivery(w http.ResponseWriter, r *http.Request) {
	delivery, err := h.service.RetryDelivery(r.Context(), r.URL.Query().Get("organizationId"), r.PathValue("deliveryID"))
	if err != nil {
		writePlatformError(w, err, "notification_delivery_retry_failed", "Notification delivery could not be retried.")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"data": delivery,
		"meta": map[string]string{"source": h.source, "queueState": "accepted", "dispatchBoundary": "notifications-worker"},
	})
}
