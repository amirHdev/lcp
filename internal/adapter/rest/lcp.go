package rest

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/amirhdev/ebook-lcp-server/internal/domain/lcp"
	"github.com/amirhdev/ebook-lcp-server/internal/pkg/id"
	usecasePublication "github.com/amirhdev/ebook-lcp-server/internal/usecase/lcp/publication"
)

type Handler struct {
	publicationRepo lcp.PublicationRepository
	publications    usecasePublication.PublicationUsecase
	startedAt       time.Time

	mu        sync.RWMutex
	processes map[string]*ProcessStatus
	metrics   Metrics
}

type ProcessRequest struct {
	Title string          `json:"title"`
	File  string          `json:"file,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
}

type ProcessStatus struct {
	ID            string    `json:"id"`
	Status        string    `json:"status"`
	PublicationID string    `json:"publicationId,omitempty"`
	Error         string    `json:"error,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type Metrics struct {
	RequestsTotal int64 `json:"requestsTotal"`
	ProcessesOK   int64 `json:"processesOk"`
	ProcessesFail int64 `json:"processesFail"`
}

func NewHandler(repo lcp.PublicationRepository, publications usecasePublication.PublicationUsecase) *Handler {
	return &Handler{
		publicationRepo: repo,
		publications:    publications,
		startedAt:       time.Now(),
		processes:       map[string]*ProcessStatus{},
	}
}

func (h *Handler) Process(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	h.countRequest()
	var req ProcessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if req.Title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title is required"})
		return
	}

	status := &ProcessStatus{
		ID:        id.New(),
		Status:    "completed",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if req.File != "" {
		raw, err := base64.StdEncoding.DecodeString(req.File)
		if err != nil {
			status.Status = "failed"
			status.Error = "file must be base64 encoded"
			h.saveProcess(status)
			h.countProcess(false)
			writeJSON(w, http.StatusBadRequest, status)
			return
		}
		pub, err := h.publications.UploadAndEncrypt(r.Context(), req.Title, bytes.NewReader(raw))
		if err != nil {
			status.Status = "failed"
			status.Error = err.Error()
			h.saveProcess(status)
			h.countProcess(false)
			writeJSON(w, http.StatusInternalServerError, status)
			return
		}
		status.PublicationID = pub.ID
	}

	h.saveProcess(status)
	h.countProcess(true)
	writeJSON(w, http.StatusOK, status)
}

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	h.countRequest()
	processID := r.URL.Query().Get("id")
	if processID == "" {
		h.mu.RLock()
		defer h.mu.RUnlock()
		items := make([]*ProcessStatus, 0, len(h.processes))
		for _, status := range h.processes {
			items = append(items, status)
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":    "ok",
			"processes": items,
			"uptimeSec": int64(time.Since(h.startedAt).Seconds()),
		})
		return
	}

	h.mu.RLock()
	status := h.processes[processID]
	h.mu.RUnlock()
	if status == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "process not found"})
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *Handler) Metrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	h.countRequest()
	h.mu.RLock()
	metrics := h.metrics
	processes := len(h.processes)
	h.mu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"uptimeSec": int64(time.Since(h.startedAt).Seconds()),
		"processes": processes,
		"metrics":   metrics,
	})
}

func (h *Handler) PrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	metrics := h.metrics
	processes := len(h.processes)
	uptime := int64(time.Since(h.startedAt).Seconds())
	h.mu.RUnlock()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = fmt.Fprintf(w, "# HELP lcp_uptime_seconds Service uptime in seconds\n")
	_, _ = fmt.Fprintf(w, "# TYPE lcp_uptime_seconds gauge\n")
	_, _ = fmt.Fprintf(w, "lcp_uptime_seconds %d\n", uptime)
	_, _ = fmt.Fprintf(w, "# HELP lcp_processes_total Known LCP processes\n")
	_, _ = fmt.Fprintf(w, "# TYPE lcp_processes_total gauge\n")
	_, _ = fmt.Fprintf(w, "lcp_processes_total %d\n", processes)
	_, _ = fmt.Fprintf(w, "# HELP lcp_requests_total HTTP requests handled by contract endpoints\n")
	_, _ = fmt.Fprintf(w, "# TYPE lcp_requests_total counter\n")
	_, _ = fmt.Fprintf(w, "lcp_requests_total %d\n", metrics.RequestsTotal)
	_, _ = fmt.Fprintf(w, "# HELP lcp_process_results_total LCP process results by status\n")
	_, _ = fmt.Fprintf(w, "# TYPE lcp_process_results_total counter\n")
	_, _ = fmt.Fprintf(w, "lcp_process_results_total{status=\"ok\"} %d\n", metrics.ProcessesOK)
	_, _ = fmt.Fprintf(w, "lcp_process_results_total{status=\"failed\"} %d\n", metrics.ProcessesFail)
}

func (h *Handler) Healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Readyz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *Handler) saveProcess(status *ProcessStatus) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.processes[status.ID] = status
}

func (h *Handler) countRequest() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.metrics.RequestsTotal++
}

func (h *Handler) countProcess(ok bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if ok {
		h.metrics.ProcessesOK++
		return
	}
	h.metrics.ProcessesFail++
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
