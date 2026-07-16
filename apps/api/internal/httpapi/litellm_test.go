package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLiteLLMIntegrationRoutes(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer upstream.Close()
	t.Setenv("LITELLM_BASE_URL", upstream.URL)
	t.Setenv("LITELLM_MASTER_KEY", "server-only-master")
	handler := NewHandler(discardLogger())

	request := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/litellm/status", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"configured":true`)) || bytes.Contains(response.Body.Bytes(), []byte("server-only-master")) {
		t.Fatalf("status=%d %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/integrations/litellm/verify", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"overall":"ready"`)) || !bytes.Contains(response.Body.Bytes(), []byte(`"databaseAccess":"none"`)) || bytes.Contains(response.Body.Bytes(), []byte("server-only-master")) {
		t.Fatalf("verify=%d %s", response.Code, response.Body.String())
	}
}

func TestLiteLLMVerifyRequiresConfiguration(t *testing.T) {
	t.Setenv("LITELLM_BASE_URL", "")
	t.Setenv("LITELLM_MASTER_KEY", "")
	handler := NewHandler(discardLogger())
	request := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/litellm/verify", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity || !bytes.Contains(response.Body.Bytes(), []byte(`"code":"litellm_not_configured"`)) {
		t.Fatalf("verify=%d %s", response.Code, response.Body.String())
	}
}
