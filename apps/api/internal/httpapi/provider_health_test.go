package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestProviderHealthRoutes(t *testing.T) {
	handler := NewHandler(discardLogger())
	body, _ := json.Marshal(map[string]any{"organizationId": "org_topoai", "region": "apac", "model": "gpt-5-mini", "requestedBy": "test@topoai.dev"})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/providers/provider_openai_primary/health/probes", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted || !bytes.Contains(response.Body.Bytes(), []byte(`"status":"queued"`)) {
		t.Fatalf("probe=%d %s", response.Code, response.Body.String())
	}

	body, _ = json.Marshal(map[string]any{"organizationId": "org_topoai", "source": "passive_telemetry", "success": false, "requestCount": 100, "errorCount": 6, "averageLatencyMs": 1400, "p95LatencyMs": 6200})
	request = httptest.NewRequest(http.MethodPost, "/api/v1/providers/provider_openai_primary/health/observations", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || !bytes.Contains(response.Body.Bytes(), []byte(`"status":"degraded"`)) || !bytes.Contains(response.Body.Bytes(), []byte(`"routingEligible":false`)) {
		t.Fatalf("observation=%d %s", response.Code, response.Body.String())
	}

	until := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	body, _ = json.Marshal(map[string]any{"organizationId": "org_topoai", "enabled": true, "until": until, "reason": "HTTP maintenance test"})
	request = httptest.NewRequest(http.MethodPost, "/api/v1/providers/provider_openai_primary/maintenance", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"status":"maintenance"`)) {
		t.Fatalf("maintenance=%d %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/provider-health-events?organizationId=org_topoai&providerId=provider_openai_primary", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"source":"passive_telemetry"`)) {
		t.Fatalf("events=%d %s", response.Code, response.Body.String())
	}
}
