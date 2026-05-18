package audit

import (
	"context"
	"time"

	repo "github.com/amirhdev/ebook-lcp-server/internal/adapter/repository/audit"
	"github.com/amirhdev/ebook-lcp-server/internal/auth"
	domain "github.com/amirhdev/ebook-lcp-server/internal/domain"
	"github.com/amirhdev/ebook-lcp-server/internal/pkg/id"
	"github.com/amirhdev/ebook-lcp-server/internal/tenant"
)

type Recorder interface {
	Record(ctx context.Context, action, resource, resourceID string) error
}

type Service struct {
	repo repo.Repository
}

func NewService(repo repo.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Record(ctx context.Context, action, resource, resourceID string) error {
	actor := "system"
	if claims, ok := auth.FromContext(ctx); ok && claims.Subject != "" {
		actor = claims.Subject
	}
	return s.repo.Save(ctx, &domain.AuditEntry{
		ID:         id.New(),
		TenantID:   tenant.IDFromContext(ctx),
		Action:     action,
		Actor:      actor,
		Resource:   resource,
		ResourceID: resourceID,
		CreatedAt:  time.Now(),
	})
}
