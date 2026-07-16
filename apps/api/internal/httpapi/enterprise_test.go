package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateOrganizationAndRejectDuplicateSlug(t *testing.T) {
	handler := NewHandler(discardLogger())
	body := []byte(`{"name":"Example Industries","slug":"example-industries","region":"Singapore","plan":"Evaluation"}`)
	create := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", bytes.NewReader(body))
	create.Header.Set("Content-Type", "application/json")
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, create)
	if created.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, created.Code, created.Body.String())
	}

	duplicate := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", bytes.NewReader(body))
	duplicate.Header.Set("Content-Type", "application/json")
	conflict := httptest.NewRecorder()
	handler.ServeHTTP(conflict, duplicate)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("expected duplicate status %d, got %d", http.StatusConflict, conflict.Code)
	}
}

func TestAPIKeySecretIsReturnedOnceAndNeverListed(t *testing.T) {
	handler := NewHandler(discardLogger())
	body := []byte(`{"name":"Production application","project":"Engineering Copilot","models":["gpt-5-mini"],"rpm":120,"createdBy":"owner@example.com"}`)
	create := httptest.NewRequest(http.MethodPost, "/api/v1/api-keys", bytes.NewReader(body))
	create.Header.Set("Content-Type", "application/json")
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, create)
	if created.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, created.Code, created.Body.String())
	}

	var createdPayload struct {
		Data   apiKeyRecord `json:"data"`
		Secret string       `json:"secret"`
	}
	decodeResponse(t, created, &createdPayload)
	if !strings.HasPrefix(createdPayload.Secret, "ag_live_") || len(createdPayload.Secret) < 32 {
		t.Fatalf("expected a strong one-time secret, got prefix %q", createdPayload.Secret)
	}

	list := httptest.NewRequest(http.MethodGet, "/api/v1/api-keys", nil)
	listed := httptest.NewRecorder()
	handler.ServeHTTP(listed, list)
	if strings.Contains(listed.Body.String(), createdPayload.Secret) || strings.Contains(listed.Body.String(), "secretHash") {
		t.Fatal("list response exposed secret material")
	}

	revoke := httptest.NewRequest(http.MethodPost, "/api/v1/api-keys/"+createdPayload.Data.ID+"/revoke", nil)
	revoked := httptest.NewRecorder()
	handler.ServeHTTP(revoked, revoke)
	if revoked.Code != http.StatusOK {
		t.Fatalf("expected revoke status %d, got %d: %s", http.StatusOK, revoked.Code, revoked.Body.String())
	}
	var revokedPayload struct {
		Data apiKeyRecord `json:"data"`
	}
	if err := json.NewDecoder(revoked.Body).Decode(&revokedPayload); err != nil {
		t.Fatalf("decode revoked key: %v", err)
	}
	if revokedPayload.Data.Status != "revoked" {
		t.Fatalf("expected revoked status, got %q", revokedPayload.Data.Status)
	}
}
