package litellm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestVerifyUsesOfficialHealthEndpointsWithoutCredentialExposure(t *testing.T) {
	const masterKey = "sk-litellm-master-test"
	seen := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+masterKey {
			t.Errorf("missing server-side authorization")
		}
		seen[r.URL.Path] = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()
	client, err := New(server.URL, masterKey, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	status, err := client.Verify(context.Background())
	if err != nil || status.Overall != "ready" || !seen["/health/liveliness"] || !seen["/health/readiness"] {
		t.Fatalf("verify status=%#v seen=%#v err=%v", status, seen, err)
	}
	encoded, _ := json.Marshal(status)
	if strings.Contains(string(encoded), masterKey) {
		t.Fatal("LiteLLM master key leaked into status")
	}
}

func TestVerifyRejectsRedirectsAndReportsReadiness(t *testing.T) {
	redirectTargetHit := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		redirectTargetHit = true
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health/liveliness" {
			w.Header().Set("Location", target.URL)
			w.WriteHeader(http.StatusTemporaryRedirect)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	client, _ := New(server.URL, "secret", time.Second)
	status, err := client.Verify(context.Background())
	if err != nil || status.Overall != "unavailable" || status.Liveness.ErrorCode != "redirect_rejected" || status.Readiness.ErrorCode != "unexpected_status" || redirectTargetHit {
		t.Fatalf("redirect/readiness boundary failed: %#v target=%v err=%v", status, redirectTargetHit, err)
	}
}

func TestClientConfigurationValidation(t *testing.T) {
	for _, value := range []string{"ftp://litellm:4000", "http://user:pass@litellm:4000", "http://litellm:4000?token=secret"} {
		if _, err := New(value, "", time.Second); err == nil {
			t.Fatalf("expected invalid base URL: %s", value)
		}
	}
	client, err := New("", "", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Verify(context.Background()); err != ErrNotConfigured {
		t.Fatalf("expected not configured, got %v", err)
	}
}
