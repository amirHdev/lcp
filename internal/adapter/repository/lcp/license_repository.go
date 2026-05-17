package lcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/amirhdev/ebook-lcp-server/internal/domain/lcp"
)

type LicenseRepository interface {
	Save(ctx context.Context, license *lcp.License) error
	FindByID(ctx context.Context, id string) (*lcp.License, error)
	FindByPublication(ctx context.Context, publicationID *string) ([]*lcp.License, error)
	Delete(ctx context.Context, id string) error
}

type licenseRepository struct {
	mu       sync.RWMutex
	licenses []*lcp.License
	path     string
}

func NewLicenseRepository() LicenseRepository {
	return &licenseRepository{}
}

func NewPersistentLicenseRepository(path string) (LicenseRepository, error) {
	repo := &licenseRepository{path: path}
	if err := repo.load(); err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *licenseRepository) Save(ctx context.Context, license *lcp.License) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	replaced := false
	for i, existing := range r.licenses {
		if existing.ID == license.ID {
			r.licenses[i] = license
			replaced = true
			break
		}
	}
	if !replaced {
		r.licenses = append(r.licenses, license)
	}
	return r.persistLocked()
}

func (r *licenseRepository) FindByID(ctx context.Context, id string) (*lcp.License, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, lic := range r.licenses {
		if lic.ID == id {
			return lic, nil
		}
	}
	return nil, nil
}

func (r *licenseRepository) FindByPublication(ctx context.Context, publicationID *string) ([]*lcp.License, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*lcp.License
	for _, lic := range r.licenses {
		if publicationID == nil || lic.PublicationID == *publicationID {
			result = append(result, lic)
		}
	}
	return result, nil
}

func (r *licenseRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	filtered := r.licenses[:0]
	for _, lic := range r.licenses {
		if lic.ID != id {
			filtered = append(filtered, lic)
		}
	}
	r.licenses = filtered
	return r.persistLocked()
}

func (r *licenseRepository) load() error {
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
	return json.Unmarshal(data, &r.licenses)
}

func (r *licenseRepository) persistLocked() error {
	if r.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r.licenses, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, data, 0o600)
}
