package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/topoai/aethergate/apps/api/internal/platform"
)

type requestRecord struct {
	ID           string  `json:"id"`
	Timestamp    string  `json:"timestamp"`
	Model        string  `json:"model"`
	Provider     string  `json:"provider"`
	Project      string  `json:"project"`
	User         string  `json:"user"`
	Status       string  `json:"status"`
	LatencyMS    int     `json:"latencyMs"`
	InputTokens  int     `json:"inputTokens"`
	OutputTokens int     `json:"outputTokens"`
	CostUSD      float64 `json:"costUsd"`
	Cached       bool    `json:"cached"`
	Prompt       string  `json:"prompt"`
	Response     string  `json:"response"`
}

type metric struct {
	Label  string  `json:"label"`
	Value  string  `json:"value"`
	Change float64 `json:"change"`
	Hint   string  `json:"hint"`
	Tone   string  `json:"tone"`
}

type server struct {
	logger   *slog.Logger
	requests []requestRecord
}

func NewHandler(logger *slog.Logger) http.Handler {
	return NewHandlerWithRepository(logger, platform.NewMemoryRepository(), "development-memory")
}

func NewHandlerWithRepository(logger *slog.Logger, repository platform.Repository, source string) http.Handler {
	s := &server{logger: logger, requests: developmentRequests()}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /readyz", s.health)
	mux.HandleFunc("GET /api/v1/overview", s.overview)
	mux.HandleFunc("GET /api/v1/requests", s.listRequests)
	mux.HandleFunc("GET /api/v1/requests/{requestID}", s.getRequest)
	registerEnterpriseRoutes(mux, logger, repository, source)
	registerFoundationRoutes(mux, repository, source)
	registerProviderRoutes(mux, repository, source)
	registerProviderHealthRoutes(mux, repository, source)
	registerRoutingRoutes(mux, repository, source)
	registerRateLimitRoutes(mux, repository, source)
	registerBudgetRoutes(mux, repository, source)
	registerAlertRoutes(mux, repository, source)
	registerWebhookRoutes(mux, repository, source)
	registerReportRoutes(mux, repository, source)
	registerNotificationRoutes(mux, repository, source)
	registerAuditRoutes(mux, repository, source)
	registerVaultRoutes(mux, repository, source)
	registerLiteLLMRoutes(mux)
	return s.recoverPanic(s.accessLog(s.cors(mux)))
}

func (s *server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "aethergate-api", "time": time.Now().UTC().Format(time.RFC3339)})
}

func (s *server) overview(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{
		"metrics": []metric{
			{Label: "Requests", Value: "1.28M", Change: 18.4, Hint: "Across 42 active projects", Tone: "accent"},
			{Label: "Total cost", Value: "$18,420", Change: 7.2, Hint: "14.3% below budget", Tone: "success"},
			{Label: "P95 latency", Value: "1.84s", Change: -12.6, Hint: "246ms faster this period", Tone: "success"},
			{Label: "Error rate", Value: "0.73%", Change: -3.1, Hint: "Within the 1% objective", Tone: "warning"},
		}, "source": "development-seed",
	}})
}

func (s *server) listRequests(w http.ResponseWriter, r *http.Request) {
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	status := r.URL.Query().Get("status")
	project := r.URL.Query().Get("project")
	filtered := make([]requestRecord, 0, len(s.requests))
	for _, record := range s.requests {
		if status != "" && status != "all" && record.Status != status {
			continue
		}
		if project != "" && project != "all" && record.Project != project {
			continue
		}
		searchFields := []string{record.ID, record.Model, record.Provider, record.Project, record.User, record.Prompt}
		if query != "" && !slices.ContainsFunc(searchFields, func(value string) bool { return strings.Contains(strings.ToLower(value), query) }) {
			continue
		}
		filtered = append(filtered, record)
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": filtered, "meta": map[string]any{"count": len(filtered), "total": len(s.requests), "source": "development-seed"}})
}

func (s *server) getRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("requestID")
	for _, record := range s.requests {
		if record.ID == id {
			writeJSON(w, http.StatusOK, map[string]any{"data": record})
			return
		}
	}
	writeError(w, http.StatusNotFound, "request_not_found", "The requested model request does not exist.")
}

func (s *server) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Info("http request", "method", r.Method, "path", r.URL.Path, "duration_ms", time.Since(started).Milliseconds())
	})
}

func (s *server) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				s.logger.Error("http panic", "error", recovered, "path", r.URL.Path)
				writeError(w, http.StatusInternalServerError, "internal_error", "An unexpected error occurred.")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID, X-AetherGate-Actor")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func developmentRequests() []requestRecord {
	return []requestRecord{
		{ID: "req_01JY8E8F9T", Timestamp: "2026-07-14T13:42:18+08:00", Model: "claude-sonnet-4", Provider: "Anthropic", Project: "Engineering Copilot", User: "li.ming@topoai.dev", Status: "success", LatencyMS: 1240, InputTokens: 18234, OutputTokens: 2841, CostUSD: 0.1284, Cached: true, Prompt: "Review the release candidate and identify migration risks.", Response: "The release candidate has three migration risks: schema compatibility, key rotation, and worker replay behavior."},
		{ID: "req_01JY8E54QH", Timestamp: "2026-07-14T13:40:51+08:00", Model: "gpt-5-mini", Provider: "OpenAI", Project: "Customer Support", User: "support-bot@acme.cn", Status: "success", LatencyMS: 842, InputTokens: 4520, OutputTokens: 932, CostUSD: 0.0216, Prompt: "Summarize the customer's open incidents and draft a response.", Response: "Two incidents remain open. The database timeout is mitigated and the export issue is awaiting validation."},
		{ID: "req_01JY8DZYCP", Timestamp: "2026-07-14T13:36:05+08:00", Model: "gemini-2.5-pro", Provider: "Google", Project: "Contract Intelligence", User: "chen.yu@acme.cn", Status: "error", LatencyMS: 10012, InputTokens: 24411, CostUSD: 0.0812, Prompt: "Compare these two contract versions and highlight financial changes.", Response: "Provider timeout before a complete response was received."},
		{ID: "req_01JY8DVF6A", Timestamp: "2026-07-14T13:31:43+08:00", Model: "deepseek-v3", Provider: "DeepSeek", Project: "Code Modernization", User: "wang.lei@topoai.dev", Status: "success", LatencyMS: 2168, InputTokens: 31900, OutputTokens: 4650, CostUSD: 0.0342, Cached: true, Prompt: "Convert this legacy data access layer to a repository abstraction.", Response: "Created repository interfaces, transaction boundaries, and a migration plan for existing callers."},
		{ID: "req_01JY8DR1MB", Timestamp: "2026-07-14T13:27:22+08:00", Model: "claude-sonnet-4", Provider: "Anthropic", Project: "Finance Analyst", User: "finance-agent@acme.cn", Status: "rate_limited", LatencyMS: 124, Prompt: "Build the weekly cost variance explanation.", Response: "Project RPM limit exceeded. Retry after 24 seconds."},
		{ID: "req_01JY8DMN8R", Timestamp: "2026-07-14T13:22:14+08:00", Model: "gpt-5-mini", Provider: "OpenAI", Project: "Knowledge Search", User: "zhao.xin@acme.cn", Status: "success", LatencyMS: 734, InputTokens: 8120, OutputTokens: 651, CostUSD: 0.0189, Cached: true, Prompt: "Find the deployment policy for regulated customer environments.", Response: "Regulated deployments require private networking, customer-managed keys, and quarterly restore validation."},
	}
}
