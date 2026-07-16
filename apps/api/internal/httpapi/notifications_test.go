package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNotificationRoutes(t *testing.T) {
	handler := NewHandler(discardLogger())

	request := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?organizationId=org_topoai&recipientId=holden%40topoai.dev&status=unread", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"note_provider_offline"`)) {
		t.Fatalf("list=%d %s", response.Code, response.Body.String())
	}

	body, _ := json.Marshal(map[string]any{
		"organizationId": "org_topoai", "recipientId": "holden@topoai.dev",
		"category": "security", "severity": "critical", "title": "HTTP security event",
		"body": "A security-sensitive role change requires review.", "sourceType": "audit_event",
		"sourceId": "audit_http_test", "actionUrl": "/audit",
	})
	request = httptest.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || !bytes.Contains(response.Body.Bytes(), []byte(`"dispatchBoundary":"notifications-worker"`)) {
		t.Fatalf("create=%d %s", response.Code, response.Body.String())
	}
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil || created.Data.ID == "" {
		t.Fatalf("decode create: %v %s", err, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/notifications/"+created.Data.ID+"/read?organizationId=org_topoai&recipientId=holden%40topoai.dev", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"status":"read"`)) {
		t.Fatalf("read=%d %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/notifications/"+created.Data.ID+"/unread?organizationId=org_topoai&recipientId=holden%40topoai.dev", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"status":"unread"`)) {
		t.Fatalf("unread=%d %s", response.Code, response.Body.String())
	}

	body, _ = json.Marshal(map[string]string{"organizationId": "org_topoai", "recipientId": "holden@topoai.dev"})
	request = httptest.NewRequest(http.MethodPost, "/api/v1/notifications/read-all", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"updated":`)) {
		t.Fatalf("read all=%d %s", response.Code, response.Body.String())
	}

	body, _ = json.Marshal(map[string]any{
		"organizationId": "org_topoai", "recipientId": "holden@topoai.dev",
		"destinations": []map[string]string{
			{"channel": "in_app", "target": "self", "displayName": "Inbox"},
			{"channel": "email", "target": "holden@topoai.dev", "displayName": "Work email"},
		},
		"categoryChannels": map[string][]string{"security": {"in_app", "email"}, "platform": {"in_app"}},
		"digestFrequency":  "daily", "minimumSeverity": "warning", "timezone": "Asia/Shanghai",
		"quietHoursEnabled": true, "quietStart": "22:00", "quietEnd": "08:00",
	})
	request = httptest.NewRequest(http.MethodPut, "/api/v1/notification-preferences", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"digestFrequency":"daily"`)) {
		t.Fatalf("preference=%d %s", response.Code, response.Body.String())
	}

	body, _ = json.Marshal(map[string]any{
		"organizationId": "org_topoai", "name": "HTTP access escalation", "status": "active",
		"categories": []string{"access", "security"}, "minimumSeverity": "warning",
		"acknowledgeWithinMinutes": 10, "repeatEveryMinutes": 15, "maxEscalations": 2,
		"routes": []map[string]any{
			{"level": 1, "delayMinutes": 0, "channel": "slack", "target": "C_SECURITY"},
			{"level": 2, "delayMinutes": 15, "channel": "email", "target": "security@topoai.dev"},
		},
	})
	request = httptest.NewRequest(http.MethodPost, "/api/v1/notification-escalation-policies", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || !bytes.Contains(response.Body.Bytes(), []byte(`"HTTP access escalation"`)) {
		t.Fatalf("create policy=%d %s", response.Code, response.Body.String())
	}

	body, _ = json.Marshal(map[string]any{"organizationId": "org_topoai", "category": "security", "severity": "critical", "unacknowledgedMinutes": 40})
	request = httptest.NewRequest(http.MethodPost, "/api/v1/notification-escalation-policies/evaluate", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"matched":true`)) || !bytes.Contains(response.Body.Bytes(), []byte(`"mode":"dry-run"`)) {
		t.Fatalf("evaluate=%d %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/notification-deliveries/ndel_budget_email_failed/retry?organizationId=org_topoai", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted || !bytes.Contains(response.Body.Bytes(), []byte(`"attempt":2`)) || !bytes.Contains(response.Body.Bytes(), []byte(`"dispatchBoundary":"notifications-worker"`)) {
		t.Fatalf("retry delivery=%d %s", response.Code, response.Body.String())
	}
}
