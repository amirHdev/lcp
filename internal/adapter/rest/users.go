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

type AdminUser struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	Verified  bool      `json:"verified"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type AdminUsersResponse struct {
	Users []*AdminUser `json:"users"`
}

type AdminUserStore struct {
	path  string
	mu    sync.Mutex
	users []*AdminUser
}

func NewAdminUserStore(dataDir string) *AdminUserStore {
	store := &AdminUserStore{path: filepath.Join(strings.TrimSpace(dataDir), "users.json")}
	store.load()
	return store
}

func (s *AdminUserStore) load() {
	if s.path == "" {
		s.users = defaultAdminUsers()
		return
	}
	b, err := os.ReadFile(s.path)
	if err != nil {
		s.users = defaultAdminUsers()
		return
	}
	if err := json.Unmarshal(b, &s.users); err != nil || len(s.users) == 0 {
		s.users = defaultAdminUsers()
	}
}

func (s *AdminUserStore) save() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s.users, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0o600)
}

func (s *AdminUserStore) List() []*AdminUser {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]*AdminUser, len(s.users))
	copy(items, s.users)
	return items
}

func (s *AdminUserStore) SetVerified(id string, verified bool) (*AdminUser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.users {
		if u.ID == id {
			u.Verified = verified
			u.UpdatedAt = time.Now().UTC()
			return u, s.save()
		}
	}
	return nil, nil
}

func defaultAdminUsers() []*AdminUser {
	now := time.Now().UTC()
	return []*AdminUser{
		{
			ID:        "publisher-01",
			Email:     "publisher01@testmedical.ir",
			Name:      "Publisher One",
			Role:      "publisher",
			Verified:  true,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "reader-01",
			Email:     "reader01@testmedical.ir",
			Name:      "Reader One",
			Role:      "user",
			Verified:  false,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

type AdminUsersHandler struct {
	store *AdminUserStore
}

func NewAdminUsersHandler(store *AdminUserStore) *AdminUsersHandler {
	return &AdminUsersHandler{store: store}
}

func (h *AdminUsersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/api/v1/admin/users") {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	trimmed := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/admin/users"), "/")
	switch {
	case trimmed == "":
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		writeJSON(w, http.StatusOK, AdminUsersResponse{Users: h.store.List()})
	case strings.HasSuffix(trimmed, "/verify") && r.Method == http.MethodPost:
		h.toggle(w, strings.TrimSuffix(trimmed, "/verify"), true)
	case strings.HasSuffix(trimmed, "/unverify") && r.Method == http.MethodPost:
		h.toggle(w, strings.TrimSuffix(trimmed, "/unverify"), false)
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	}
}

func (h *AdminUsersHandler) toggle(w http.ResponseWriter, id string, verified bool) {
	id = strings.Trim(id, "/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user id is required"})
		return
	}
	user, err := h.store.SetVerified(id, verified)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if user == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}
	writeJSON(w, http.StatusOK, user)
}
