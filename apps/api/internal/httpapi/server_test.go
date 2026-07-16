package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()

	NewHandler(discardLogger()).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var payload struct {
		Status  string `json:"status"`
		Service string `json:"service"`
	}
	decodeResponse(t, response, &payload)
	if payload.Status != "ok" || payload.Service != "aethergate-api" {
		t.Fatalf("unexpected health response: %+v", payload)
	}
}

func TestListRequestsAppliesFilters(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/requests?status=success&project=Knowledge%20Search&q=deployment", nil)
	response := httptest.NewRecorder()

	NewHandler(discardLogger()).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var payload struct {
		Data []requestRecord `json:"data"`
		Meta struct {
			Count int `json:"count"`
			Total int `json:"total"`
		} `json:"meta"`
	}
	decodeResponse(t, response, &payload)
	if payload.Meta.Count != 1 || payload.Meta.Total != 6 || len(payload.Data) != 1 {
		t.Fatalf("unexpected filtered response: %+v", payload)
	}
	if payload.Data[0].ID != "req_01JY8DMN8R" {
		t.Fatalf("expected Knowledge Search request, got %s", payload.Data[0].ID)
	}
}

func TestGetRequestAndNotFound(t *testing.T) {
	handler := NewHandler(discardLogger())

	foundRequest := httptest.NewRequest(http.MethodGet, "/api/v1/requests/req_01JY8E8F9T", nil)
	foundResponse := httptest.NewRecorder()
	handler.ServeHTTP(foundResponse, foundRequest)
	if foundResponse.Code != http.StatusOK {
		t.Fatalf("expected existing request status %d, got %d", http.StatusOK, foundResponse.Code)
	}

	missingRequest := httptest.NewRequest(http.MethodGet, "/api/v1/requests/missing", nil)
	missingResponse := httptest.NewRecorder()
	handler.ServeHTTP(missingResponse, missingRequest)
	if missingResponse.Code != http.StatusNotFound {
		t.Fatalf("expected missing request status %d, got %d", http.StatusNotFound, missingResponse.Code)
	}

	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeResponse(t, missingResponse, &payload)
	if payload.Error.Code != "request_not_found" {
		t.Fatalf("unexpected error code %q", payload.Error.Code)
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func decodeResponse(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}
