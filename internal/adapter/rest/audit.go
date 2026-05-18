package rest

import (
	"net/http"
	"strconv"

	repo "github.com/amirhdev/ebook-lcp-server/internal/adapter/repository/audit"
	"github.com/amirhdev/ebook-lcp-server/internal/tenant"
)

type AuditHandler struct {
	repo repo.Repository
}

func NewAuditHandler(repo repo.Repository) *AuditHandler {
	return &AuditHandler{repo: repo}
}

func (h *AuditHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 100
	}
	entries, err := h.repo.FindRecentByTenant(r.Context(), tenant.IDFromContext(r.Context()), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries})
}
