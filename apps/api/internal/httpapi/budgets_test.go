package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBudgetRoutes(t *testing.T) {
	h := NewHandler(discardLogger())
	body, _ := json.Marshal(map[string]any{"organizationId": "org_topoai", "name": "HTTP block budget", "scopeType": "project", "scopeId": "project_engineering_copilot", "period": "monthly", "limitUsd": 1000, "warningPercent": 70, "criticalPercent": 90, "action": "block"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/budgets", bytes.NewReader(body))
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
	r = httptest.NewRequest(http.MethodPost, "/api/v1/budgets/"+p.Data.ID+"/activate?organizationId=org_topoai", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("activate=%d", w.Code)
	}
	body, _ = json.Marshal(map[string]any{"organizationId": "org_topoai", "projectId": "project_engineering_copilot", "currentSpendUsd": 950, "proposedSpendUsd": 100, "elapsedPercent": 50})
	r = httptest.NewRequest(http.MethodPost, "/api/v1/budgets/evaluate", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte(`"allowed":false`)) {
		t.Fatalf("decision=%d %s", w.Code, w.Body.String())
	}
}
