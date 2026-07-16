package httpapi

import (
	"net/http"

	"github.com/topoai/aethergate/apps/api/internal/platform"
)

type vaultHandler struct {
	service   *platform.VaultService
	source    string
	algorithm string
}

func registerVaultRoutes(mux *http.ServeMux, repository platform.Repository, source string) {
	repo, ok := repository.(platform.VaultRepository)
	if !ok {
		return
	}
	protector := platform.NewVaultProtector(source)
	handler := &vaultHandler{service: platform.NewVaultService(repo, protector), source: source, algorithm: protector.Algorithm()}
	mux.HandleFunc("GET /api/v1/vault/secrets", handler.list)
	mux.HandleFunc("POST /api/v1/vault/secrets", handler.create)
	mux.HandleFunc("GET /api/v1/vault/secrets/{secretID}", handler.get)
	mux.HandleFunc("POST /api/v1/vault/secrets/{secretID}/rotate", handler.rotate)
	mux.HandleFunc("POST /api/v1/vault/secrets/{secretID}/disable", handler.disable)
	mux.HandleFunc("GET /api/v1/vault/access-events", handler.accessEvents)
}

func (h *vaultHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context(), platform.VaultSecretFilter{
		OrganizationID: r.URL.Query().Get("organizationId"), Query: r.URL.Query().Get("q"),
		Kind: r.URL.Query().Get("kind"), ScopeType: r.URL.Query().Get("scopeType"),
		Status: r.URL.Query().Get("status"), Rotation: r.URL.Query().Get("rotation"),
	})
	if err != nil {
		writePlatformError(w, err, "vault_secrets_unavailable", "Vault secret metadata could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, listPayload(items, h.source))
}

func (h *vaultHandler) get(w http.ResponseWriter, r *http.Request) {
	secret, err := h.service.Get(r.Context(), r.URL.Query().Get("organizationId"), r.PathValue("secretID"))
	if err != nil {
		writePlatformError(w, err, "vault_secret_not_found", "Vault secret metadata does not exist.")
		return
	}
	h.writeSecret(w, http.StatusOK, secret, "metadata-only")
}

func (h *vaultHandler) create(w http.ResponseWriter, r *http.Request) {
	var input platform.CreateVaultSecretInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	secret, err := h.service.Create(r.Context(), input)
	if err != nil {
		writePlatformError(w, err, "vault_secret_create_failed", "Vault secret could not be encrypted and stored.")
		return
	}
	h.writeSecret(w, http.StatusCreated, secret, "encrypted")
}

func (h *vaultHandler) rotate(w http.ResponseWriter, r *http.Request) {
	var input platform.RotateVaultSecretInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	secret, err := h.service.Rotate(r.Context(), r.PathValue("secretID"), input)
	if err != nil {
		writePlatformError(w, err, "vault_secret_rotate_failed", "Vault secret could not be rotated.")
		return
	}
	h.writeSecret(w, http.StatusOK, secret, "rotated")
}

func (h *vaultHandler) disable(w http.ResponseWriter, r *http.Request) {
	var input platform.DisableVaultSecretInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body must be valid JSON.")
		return
	}
	secret, err := h.service.Disable(r.Context(), r.PathValue("secretID"), input)
	if err != nil {
		writePlatformError(w, err, "vault_secret_disable_failed", "Vault secret could not be disabled.")
		return
	}
	h.writeSecret(w, http.StatusOK, secret, "disabled")
}

func (h *vaultHandler) accessEvents(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListAccessEvents(r.Context(), platform.VaultAccessFilter{
		OrganizationID: r.URL.Query().Get("organizationId"), SecretID: r.URL.Query().Get("secretId"),
		Outcome: r.URL.Query().Get("outcome"), Actor: r.URL.Query().Get("actor"),
	})
	if err != nil {
		writePlatformError(w, err, "vault_access_unavailable", "Vault access evidence could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, listPayload(items, h.source))
}

func (h *vaultHandler) writeSecret(w http.ResponseWriter, status int, secret platform.VaultSecret, operation string) {
	writeJSON(w, status, map[string]any{
		"data": secret,
		"meta": map[string]any{
			"source": h.source, "operation": operation, "plaintextReturned": false,
			"encryption": h.algorithm, "resolutionBoundary": "internal-workers-only",
		},
	})
}
