package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProviderRoutes(t *testing.T) {
	handler := NewHandler(discardLogger())
	body, _ := json.Marshal(map[string]any{"organizationId": "org_topoai", "name": "Azure Test", "provider": "Azure OpenAI", "baseUrl": "https://example.openai.azure.com"})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/providers", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create provider status=%d body=%s", response.Code, response.Body.String())
	}
	request = httptest.NewRequest(http.MethodGet, "/api/v1/providers?organizationId=org_topoai&q=azure", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte("Azure Test")) {
		t.Fatalf("list providers status=%d body=%s", response.Code, response.Body.String())
	}
}
