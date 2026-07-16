package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFoundationHTTPResourceCreation(t *testing.T) {
	handler := NewHandler(discardLogger())
	workspace := postJSON(t, handler, "/api/v1/workspaces", `{"organizationId":"org_topoai","name":"AI Delivery","environment":"production"}`)
	workspaceID := dataID(t, workspace)
	project := postJSON(t, handler, "/api/v1/projects", `{"organizationId":"org_topoai","workspaceId":"`+workspaceID+`","name":"Document Intelligence","budgetUsd":4000}`)
	if dataID(t, project) == "" {
		t.Fatal("project ID was empty")
	}
	member := postJSON(t, handler, "/api/v1/members", `{"organizationId":"org_topoai","email":"new.member@example.com","role":"developer"}`)
	if dataID(t, member) == "" {
		t.Fatal("member ID was empty")
	}

	for _, path := range []string{"/api/v1/workspaces?organizationId=org_topoai", "/api/v1/projects?organizationId=org_topoai", "/api/v1/members?organizationId=org_topoai", "/api/v1/models"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s returned %d: %s", path, response.Code, response.Body.String())
		}
	}
}

func postJSON(t *testing.T, handler http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code < 200 || response.Code >= 300 {
		t.Fatalf("POST %s returned %d: %s", path, response.Code, response.Body.String())
	}
	return response
}

func dataID(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()
	var payload struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return payload.Data.ID
}
