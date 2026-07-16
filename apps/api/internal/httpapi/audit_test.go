package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuditRoutes(t *testing.T) {
	handler := NewHandler(discardLogger())
	body, _ := json.Marshal(map[string]any{
		"organizationId": "org_topoai", "actorId": "user_holden", "actorEmail": "holden@topoai.dev",
		"action": "provider.maintenance_enabled", "resourceType": "provider", "resourceId": "provider_azure_east",
		"outcome": "success", "riskLevel": "high", "source": "control-plane", "reason": "Planned maintenance",
		"requestId": "req_http_audit", "ipAddress": "10.0.0.8", "beforeState": map[string]any{"maintenance": false}, "afterState": map[string]any{"maintenance": true},
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/audit-events", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || !bytes.Contains(response.Body.Bytes(), []byte(`"integrity":"sha256-chain"`)) {
		t.Fatalf("append=%d %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/audit-events/verify?organizationId=org_topoai", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"valid":true`)) || !bytes.Contains(response.Body.Bytes(), []byte(`"eventCount":4`)) {
		t.Fatalf("verify=%d %s", response.Code, response.Body.String())
	}

	body, _ = json.Marshal(map[string]any{"organizationId": "org_topoai", "retentionDays": 730, "legalHold": true, "exportFormat": "jsonl", "updatedBy": "security@topoai.dev"})
	request = httptest.NewRequest(http.MethodPut, "/api/v1/audit-retention", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"legalHold":true`)) {
		t.Fatalf("retention=%d %s", response.Code, response.Body.String())
	}

	body, _ = json.Marshal(map[string]any{"organizationId": "org_topoai", "requestedBy": "holden@topoai.dev", "format": "csv", "periodStart": "2026-07-01T00:00:00Z", "periodEnd": "2026-07-14T00:00:00Z", "filters": map[string]string{"riskLevel": "high"}})
	request = httptest.NewRequest(http.MethodPost, "/api/v1/audit-exports", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted || !bytes.Contains(response.Body.Bytes(), []byte(`"dispatchBoundary":"audit-export-worker"`)) {
		t.Fatalf("export=%d %s", response.Code, response.Body.String())
	}

	body, _ = json.Marshal(map[string]string{"requestedBy": "holden@topoai.dev"})
	request = httptest.NewRequest(http.MethodPost, "/api/v1/audit-exports/aexp_failed/retry?organizationId=org_topoai", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted || !bytes.Contains(response.Body.Bytes(), []byte(`"parentId":"aexp_failed"`)) {
		t.Fatalf("retry=%d %s", response.Code, response.Body.String())
	}
}
