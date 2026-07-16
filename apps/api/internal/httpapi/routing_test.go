package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRoutingPolicyRoutes(t *testing.T) {
	handler := NewHandler(discardLogger())
	body, _ := json.Marshal(map[string]any{"organizationId": "org_topoai", "name": "HTTP weighted", "strategy": "weighted", "modelPattern": "http/*", "maxRetries": 2, "requestTimeoutMs": 30000, "targets": []map[string]any{{"providerId": "provider_openai_primary", "model": "gpt-5-mini", "priority": 1, "weight": 100, "enabled": true}}})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/routing-policies", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create route status=%d body=%s", response.Code, response.Body.String())
	}
	var payload struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	request = httptest.NewRequest(http.MethodPost, "/api/v1/routing-policies/"+payload.Data.ID+"/activate?organizationId=org_topoai", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"status":"active"`)) {
		t.Fatalf("activate route status=%d body=%s", response.Code, response.Body.String())
	}
}
