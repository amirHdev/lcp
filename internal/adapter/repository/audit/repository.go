package audit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	domain "github.com/amirhdev/ebook-lcp-server/internal/domain"
)

type Repository interface {
	Save(ctx context.Context, entry *domain.AuditEntry) error
	FindRecent(ctx context.Context, limit int) ([]*domain.AuditEntry, error)
	FindRecentByTenant(ctx context.Context, tenantID string, limit int) ([]*domain.AuditEntry, error)
}

type repository struct {
	mu      sync.RWMutex
	entries []*domain.AuditEntry
	path    string
}

func NewRepository() Repository {
	return &repository{}
}

func NewPersistentRepository(path string) (Repository, error) {
	repo := &repository{path: path}
	if err := repo.load(); err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *repository) Save(_ context.Context, entry *domain.AuditEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, entry)
	return r.persistLocked()
}

func (r *repository) FindRecent(_ context.Context, limit int) ([]*domain.AuditEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 || limit > len(r.entries) {
		limit = len(r.entries)
	}
	result := make([]*domain.AuditEntry, 0, limit)
	for i := len(r.entries) - 1; i >= 0 && len(result) < limit; i-- {
		result = append(result, r.entries[i])
	}
	return result, nil
}

func (r *repository) FindRecentByTenant(_ context.Context, tenantID string, limit int) ([]*domain.AuditEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*domain.AuditEntry, 0, limit)
	for i := len(r.entries) - 1; i >= 0 && (limit <= 0 || len(result) < limit); i-- {
		if r.entries[i].TenantID == tenantID {
			result = append(result, r.entries[i])
		}
	}
	return result, nil
}

func (r *repository) load() error {
	if r.path == "" {
		return nil
	}
	data, err := os.ReadFile(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &r.entries)
}

func (r *repository) persistLocked() error {
	if r.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r.entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, data, 0o600)
}
