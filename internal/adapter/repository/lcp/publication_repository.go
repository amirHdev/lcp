package lcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/amirhdev/ebook-lcp-server/internal/domain/lcp"
)

type PublicationRepository interface {
	Save(ctx context.Context, pub *lcp.Publication) error
	FindAll(ctx context.Context) ([]*lcp.Publication, error)
	FindByID(ctx context.Context, id string) (*lcp.Publication, error)
}

type publicationRepository struct {
	mu           sync.RWMutex
	publications []*lcp.Publication
	path         string
}

func NewPublicationRepository() PublicationRepository {
	return &publicationRepository{}
}

func NewPersistentPublicationRepository(path string) (PublicationRepository, error) {
	repo := &publicationRepository{path: path}
	if err := repo.load(); err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *publicationRepository) Save(_ context.Context, pub *lcp.Publication) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	pub.UpdatedAt = time.Now()
	replaced := false
	for i, existing := range r.publications {
		if existing.ID == pub.ID {
			r.publications[i] = pub
			replaced = true
			break
		}
	}
	if !replaced {
		r.publications = append(r.publications, pub)
	}
	return r.persistLocked()
}

func (r *publicationRepository) FindAll(_ context.Context) ([]*lcp.Publication, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	pubs := make([]*lcp.Publication, len(r.publications))
	copy(pubs, r.publications)
	return pubs, nil
}

func (r *publicationRepository) FindByID(_ context.Context, id string) (*lcp.Publication, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, pub := range r.publications {
		if pub.ID == id {
			return pub, nil
		}
	}

	return nil, nil
}

func (r *publicationRepository) load() error {
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
	return json.Unmarshal(data, &r.publications)
}

func (r *publicationRepository) persistLocked() error {
	if r.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r.publications, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, data, 0o600)
}
