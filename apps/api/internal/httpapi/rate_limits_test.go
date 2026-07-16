package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimitRoutesAndDryRun(t *testing.T) {
	handler := NewHandler(discardLogger())
	body, _ := json.Marshal(map[string]any{"organizationId": "org_topoai", "name": "HTTP project cap", "scopeType": "project", "scopeId": "project_engineering_copilot", "metric": "requests", "window": "minute", "limit": 100, "burst": 10, "action": "reject", "priority": 500})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/rate-limits", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create rate limit status=%d body=%s", response.Code, response.Body.String())
	}
	var payload struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(response.Body.Bytes(), &payload)
	request = httptest.NewRequest(http.MethodPost, "/api/v1/rate-limits/"+payload.Data.ID+"/enforce?organizationId=org_topoai", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("enforce rate limit status=%d body=%s", response.Code, response.Body.String())
	}
	body, _ = json.Marshal(map[string]any{"organizationId": "org_topoai", "projectId": "project_engineering_copilot", "metric": "requests", "currentUsage": 105, "requestedUnits": 10})
	request = httptest.NewRequest(http.MethodPost, "/api/v1/rate-limits/evaluate", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"allowed":false`)) {
		t.Fatalf("evaluate status=%d body=%s", response.Code, response.Body.String())
	}
}
