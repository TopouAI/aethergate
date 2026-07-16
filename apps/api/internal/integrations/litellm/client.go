package litellm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var ErrNotConfigured = errors.New("LiteLLM integration is not configured")

type ProbeResult struct {
	Path       string `json:"path"`
	Healthy    bool   `json:"healthy"`
	StatusCode int    `json:"statusCode"`
	LatencyMS  int64  `json:"latencyMs"`
	ErrorCode  string `json:"errorCode"`
}

type Status struct {
	Configured          bool         `json:"configured"`
	BaseURL             string       `json:"baseUrl"`
	MasterKeyConfigured bool         `json:"masterKeyConfigured"`
	Overall             string       `json:"overall"`
	Liveness            *ProbeResult `json:"liveness"`
	Readiness           *ProbeResult `json:"readiness"`
	CheckedAt           *time.Time   `json:"checkedAt"`
}

type Client struct {
	baseURL   *url.URL
	masterKey string
	http      *http.Client
	now       func() time.Time
}

func New(baseURL, masterKey string, timeout time.Duration) (*Client, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return &Client{http: secureHTTPClient(timeout), now: time.Now}, nil
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("LiteLLM base URL must be an absolute HTTP(S) URL without credentials, query, or fragment")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return &Client{baseURL: parsed, masterKey: strings.TrimSpace(masterKey), http: secureHTTPClient(timeout), now: time.Now}, nil
}

func NewFromEnvironment() *Client {
	client, err := New(os.Getenv("LITELLM_BASE_URL"), os.Getenv("LITELLM_MASTER_KEY"), 5*time.Second)
	if err != nil {
		return &Client{http: secureHTTPClient(5 * time.Second), now: time.Now}
	}
	return client
}

func secureHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 || timeout > 30*time.Second {
		timeout = 5 * time.Second
	}
	return &http.Client{
		Timeout:       timeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}
}

func (c *Client) ConfigurationStatus() Status {
	status := Status{Overall: "not_configured"}
	if c.baseURL == nil {
		return status
	}
	status.Configured = true
	status.BaseURL = c.baseURL.String()
	status.MasterKeyConfigured = c.masterKey != ""
	status.Overall = "configured"
	return status
}

func (c *Client) Verify(ctx context.Context) (Status, error) {
	status := c.ConfigurationStatus()
	if c.baseURL == nil {
		return status, ErrNotConfigured
	}
	liveness := c.probe(ctx, "/health/liveliness")
	readiness := c.probe(ctx, "/health/readiness")
	checkedAt := c.now().UTC()
	status.Liveness = &liveness
	status.Readiness = &readiness
	status.CheckedAt = &checkedAt
	if liveness.Healthy && readiness.Healthy {
		status.Overall = "ready"
	} else if liveness.Healthy {
		status.Overall = "not_ready"
	} else {
		status.Overall = "unavailable"
	}
	return status, nil
}

func (c *Client) probe(ctx context.Context, path string) ProbeResult {
	result := ProbeResult{Path: path}
	target := *c.baseURL
	target.Path = strings.TrimRight(target.Path, "/") + path
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		result.ErrorCode = "request_invalid"
		return result
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "AetherGate-LiteLLM-Diagnostics/1.0")
	if c.masterKey != "" {
		request.Header.Set("Authorization", "Bearer "+c.masterKey)
	}
	started := time.Now()
	response, err := c.http.Do(request)
	result.LatencyMS = time.Since(started).Milliseconds()
	if err != nil {
		var networkError net.Error
		if errors.As(err, &networkError) && networkError.Timeout() {
			result.ErrorCode = "timeout"
		} else {
			result.ErrorCode = "unreachable"
		}
		return result
	}
	defer response.Body.Close()
	result.StatusCode = response.StatusCode
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 32*1024))
	if response.StatusCode == http.StatusOK {
		result.Healthy = true
		return result
	}
	if response.StatusCode >= 300 && response.StatusCode < 400 {
		result.ErrorCode = "redirect_rejected"
	} else if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		result.ErrorCode = "authentication_failed"
	} else {
		result.ErrorCode = "unexpected_status"
	}
	return result
}
