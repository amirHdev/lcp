package rest

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type TenantRecord struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	APIKeys       []APIKey  `json:"apiKeys,omitempty"`
	WebhookURLs   []string  `json:"webhookUrls,omitempty"`
	StoragePrefix string    `json:"storagePrefix,omitempty"`
	RateLimitRPM  int       `json:"rateLimitRpm,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type APIKey struct {
	Key     string `json:"key"`
	Subject string `json:"subject"`
	Role    string `json:"role"`
}

type TenantStore struct {
	path    string
	mu      sync.Mutex
	tenants map[string]*TenantRecord
}

func NewTenantStore(dataDir, defaultTenantID string) *TenantStore {
	store := &TenantStore{
		path:    filepath.Join(strings.TrimSpace(dataDir), "tenants.json"),
		tenants: map[string]*TenantRecord{},
	}
	store.load(defaultTenantID)
	return store
}

func (s *TenantStore) load(defaultTenantID string) {
	if s.path != "" {
		if raw, err := os.ReadFile(s.path); err == nil {
			var items []*TenantRecord
			if json.Unmarshal(raw, &items) == nil {
				for _, tenant := range items {
					s.tenants[tenant.ID] = tenant
				}
			}
		}
	}
	if len(s.tenants) == 0 {
		now := time.Now().UTC()
		id := strings.TrimSpace(defaultTenantID)
		if id == "" {
			id = "default"
		}
		s.tenants[id] = &TenantRecord{ID: id, Name: id, CreatedAt: now, UpdatedAt: now}
	}
}

func (s *TenantStore) save() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	items := make([]*TenantRecord, 0, len(s.tenants))
	for _, tenant := range s.tenants {
		items = append(items, tenant)
	}
	raw, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0o600)
}

func (s *TenantStore) List() []*TenantRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]*TenantRecord, 0, len(s.tenants))
	for _, tenant := range s.tenants {
		copy := *tenant
		items = append(items, &copy)
	}
	return items
}

func (s *TenantStore) Get(id string) (*TenantRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tenant, ok := s.tenants[id]
	if !ok {
		return nil, false
	}
	copy := *tenant
	return &copy, true
}

func (s *TenantStore) Upsert(tenant *TenantRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if existing, ok := s.tenants[tenant.ID]; ok {
		tenant.CreatedAt = existing.CreatedAt
	} else if tenant.CreatedAt.IsZero() {
		tenant.CreatedAt = now
	}
	tenant.UpdatedAt = now
	copy := *tenant
	s.tenants[tenant.ID] = &copy
	return s.save()
}

func (s *TenantStore) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tenants[id]; !ok {
		return false
	}
	delete(s.tenants, id)
	return s.save() == nil
}

type AdminTenantsHandler struct {
	store *TenantStore
}

func NewAdminTenantsHandler(store *TenantStore) *AdminTenantsHandler {
	return &AdminTenantsHandler{store: store}
}

func (h *AdminTenantsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/admin/tenants"), "/")
	if trimmed == "" {
		h.collection(w, r)
		return
	}
	h.item(w, r, trimmed)
}

func (h *AdminTenantsHandler) collection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"tenants": h.store.List()})
	case http.MethodPost:
		var req TenantRecord
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.ID) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "tenant id is required"})
			return
		}
		req.ID = strings.TrimSpace(req.ID)
		if req.Name == "" {
			req.Name = req.ID
		}
		if err := h.store.Upsert(&req); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, req)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *AdminTenantsHandler) item(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodGet:
		tenant, ok := h.store.Get(id)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "tenant not found"})
			return
		}
		writeJSON(w, http.StatusOK, tenant)
	case http.MethodPut:
		var req TenantRecord
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		req.ID = id
		if req.Name == "" {
			req.Name = id
		}
		if err := h.store.Upsert(&req); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, req)
	case http.MethodDelete:
		if !h.store.Delete(id) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "tenant not found"})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}
