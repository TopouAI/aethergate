package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReportRoutes(t *testing.T) {
	handler := NewHandler(discardLogger())
	body, _ := json.Marshal(map[string]any{
		"organizationId": "org_topoai", "name": "HTTP weekly reliability", "template": "reliability",
		"status": "active", "frequency": "weekly", "dayOfWeek": "monday", "localTime": "10:00",
		"timezone": "Asia/Shanghai", "formats": []string{"xlsx", "csv"},
		"recipients": []map[string]any{{"channel": "email", "target": "ops@topoai.dev"}},
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/reports", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || !bytes.Contains(response.Body.Bytes(), []byte(`"nextRunAt"`)) {
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

	body, _ = json.Marshal(map[string]any{"organizationId": "org_topoai", "requestedBy": "http-test@topoai.dev"})
	request = httptest.NewRequest(http.MethodPost, "/api/v1/reports/"+created.Data.ID+"/run", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted || !bytes.Contains(response.Body.Bytes(), []byte(`"status":"queued"`)) || !bytes.Contains(response.Body.Bytes(), []byte(`"dispatchBoundary":"reports-worker"`)) {
		t.Fatalf("run=%d %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/reports/"+created.Data.ID+"/pause?organizationId=org_topoai", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"status":"paused"`)) {
		t.Fatalf("pause=%d %s", response.Code, response.Body.String())
	}

	body, _ = json.Marshal(map[string]any{"requestedBy": "http-test@topoai.dev"})
	request = httptest.NewRequest(http.MethodPost, "/api/v1/report-runs/rrun_exec_failed/retry?organizationId=org_topoai", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted || !bytes.Contains(response.Body.Bytes(), []byte(`"trigger":"retry"`)) || !bytes.Contains(response.Body.Bytes(), []byte(`"attempt":2`)) {
		t.Fatalf("retry=%d %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/report-runs?organizationId=org_topoai&reportId="+created.Data.ID, nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(created.Data.ID)) {
		t.Fatalf("list runs=%d %s", response.Code, response.Body.String())
	}
}
