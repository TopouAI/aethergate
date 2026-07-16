package httpapi

import (
	"errors"
	"net/http"

	"github.com/topoai/aethergate/apps/api/internal/integrations/litellm"
)

type liteLLMHandler struct{ client *litellm.Client }

func registerLiteLLMRoutes(mux *http.ServeMux) {
	handler := &liteLLMHandler{client: litellm.NewFromEnvironment()}
	mux.HandleFunc("GET /api/v1/integrations/litellm/status", handler.status)
	mux.HandleFunc("POST /api/v1/integrations/litellm/verify", handler.verify)
}

func (h *liteLLMHandler) status(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"data": h.client.ConfigurationStatus(), "meta": map[string]string{"source": "server-configuration", "credentialExposure": "none"}})
}

func (h *liteLLMHandler) verify(w http.ResponseWriter, r *http.Request) {
	status, err := h.client.Verify(r.Context())
	if errors.Is(err, litellm.ErrNotConfigured) {
		writeError(w, http.StatusUnprocessableEntity, "litellm_not_configured", "LITELLM_BASE_URL is required before verification.")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, "litellm_verification_failed", "LiteLLM verification could not be completed.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": status, "meta": map[string]string{"source": "live-probe", "credentialExposure": "none", "databaseAccess": "none"}})
}
