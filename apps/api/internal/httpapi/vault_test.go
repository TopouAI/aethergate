package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVaultRoutesNeverReturnPlaintext(t *testing.T) {
	handler := NewHandler(discardLogger())
	plaintext := "sk-http-vault-secret-123456"
	body, _ := json.Marshal(map[string]any{
		"organizationId": "org_topoai", "name": "HTTP Provider Key", "kind": "provider_api_key",
		"scopeType": "provider", "scopeId": "provider_http", "secretValue": plaintext,
		"rotationIntervalDays": 45, "createdBy": "security@topoai.dev", "requestId": "req_http_create", "sourceIp": "10.0.0.8",
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/vault/secrets", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || bytes.Contains(response.Body.Bytes(), []byte(plaintext)) || !bytes.Contains(response.Body.Bytes(), []byte(`"plaintextReturned":false`)) || !bytes.Contains(response.Body.Bytes(), []byte(`"encryption":"AES-256-GCM"`)) {
		t.Fatalf("create=%d %s", response.Code, response.Body.String())
	}
	var created struct {
		Data struct {
			ID             string `json:"id"`
			CurrentVersion int    `json:"currentVersion"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil || created.Data.ID == "" || created.Data.CurrentVersion != 1 {
		t.Fatalf("decode create response: %#v %v", created, err)
	}

	body, _ = json.Marshal(map[string]any{
		"organizationId": "org_topoai", "secretValue": "sk-http-vault-rotated-654321", "reason": "scheduled rotation",
		"rotatedBy": "security@topoai.dev", "requestId": "req_http_rotate", "sourceIp": "10.0.0.8",
	})
	request = httptest.NewRequest(http.MethodPost, "/api/v1/vault/secrets/"+created.Data.ID+"/rotate", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"currentVersion":2`)) || bytes.Contains(response.Body.Bytes(), []byte("sk-http-vault-rotated")) {
		t.Fatalf("rotate=%d %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/vault/access-events?organizationId=org_topoai&secretId="+created.Data.ID, nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"purpose":"create secret"`)) || !bytes.Contains(response.Body.Bytes(), []byte(`"purpose":"rotate secret: scheduled rotation"`)) {
		t.Fatalf("access=%d %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/vault/secrets/"+created.Data.ID+"/resolve", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code == http.StatusOK || bytes.Contains(response.Body.Bytes(), []byte(plaintext)) {
		t.Fatalf("public resolve boundary must not exist: %d %s", response.Code, response.Body.String())
	}
}

func TestVaultRouteValidation(t *testing.T) {
	handler := NewHandler(discardLogger())
	body, _ := json.Marshal(map[string]any{
		"organizationId": "org_topoai", "name": "Weak", "kind": "provider_api_key", "scopeType": "provider",
		"scopeId": "provider_weak", "secretValue": "short", "rotationIntervalDays": 90,
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/vault/secrets", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity || !bytes.Contains(response.Body.Bytes(), []byte(`"code":"vault_secret_invalid"`)) {
		t.Fatalf("validation=%d %s", response.Code, response.Body.String())
	}
}
