package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAlertRoutes(t *testing.T) {
	h := NewHandler(discardLogger())
	b, _ := json.Marshal(map[string]any{"organizationId": "org_topoai", "name": "HTTP token spike", "metric": "tokens", "operator": "gte", "threshold": 1000, "window": "5m", "cooldownMinutes": 10, "severity": "warning", "channels": []string{"in_app"}, "filters": map[string]string{"project": "copilot"}})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/alerts", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 201 {
		t.Fatalf("create=%d %s", w.Code, w.Body.String())
	}
	var p struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &p)
	r = httptest.NewRequest(http.MethodPost, "/api/v1/alerts/"+p.Data.ID+"/enable?organizationId=org_topoai", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	b, _ = json.Marshal(map[string]any{"organizationId": "org_topoai", "metric": "tokens", "value": 1200, "dimensions": map[string]string{"project": "copilot"}})
	r = httptest.NewRequest(http.MethodPost, "/api/v1/alerts/evaluate", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte(`"triggered":true`)) {
		t.Fatalf("eval=%d %s", w.Code, w.Body.String())
	}
}
