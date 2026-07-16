package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebhookRoutesCreateQueueDisableAndReplay(t *testing.T) {
	handler := NewHandler(discardLogger())
	body, _ := json.Marshal(map[string]any{
		"organizationId": "org_topoai", "name": "HTTP automation", "destination": "https://example.com/aethergate",
		"events": []string{"request.completed", "alert.triggered"}, "sampleRate": 50, "includeData": true,
		"propertyFilters": []map[string]string{{"key": "project", "value": "copilot"}}, "maxAttempts": 4, "timeoutSeconds": 8,
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create=%d %s", response.Code, response.Body.String())
	}
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
		SigningSecret string `json:"signingSecret"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil || !strings.HasPrefix(created.SigningSecret, "whsec_") {
		t.Fatalf("created=%+v err=%v", created, err)
	}

	body, _ = json.Marshal(map[string]string{"eventType": "request.completed"})
	request = httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/"+created.Data.ID+"/test?organizationId=org_topoai", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted || !bytes.Contains(response.Body.Bytes(), []byte(`"trigger":"test"`)) {
		t.Fatalf("test=%d %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/"+created.Data.ID+"/disable?organizationId=org_topoai", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"status":"disabled"`)) {
		t.Fatalf("disable=%d %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/webhook-deliveries/whd_dead_03/replay?organizationId=org_topoai", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted || !bytes.Contains(response.Body.Bytes(), []byte(`"trigger":"replay"`)) {
		t.Fatalf("replay=%d %s", response.Code, response.Body.String())
	}
}
