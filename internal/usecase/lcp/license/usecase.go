package license

import (
	"context"
	"fmt"
	"time"

	auditservice "github.com/amirhdev/ebook-lcp-server/internal/audit"
	userdomain "github.com/amirhdev/ebook-lcp-server/internal/domain"
	"github.com/amirhdev/ebook-lcp-server/internal/domain/lcp"
	lcpencrypt "github.com/amirhdev/ebook-lcp-server/internal/lcp/encrypt"
	lcplicense "github.com/amirhdev/ebook-lcp-server/internal/lcp/license"
	"github.com/amirhdev/ebook-lcp-server/internal/observability"
	"github.com/amirhdev/ebook-lcp-server/internal/pkg/id"
	"github.com/amirhdev/ebook-lcp-server/internal/tenant"
	"github.com/amirhdev/ebook-lcp-server/internal/webhook"
)

type LicenseUsecase interface {
	Create(ctx context.Context, input *lcp.LicenseInput) (*lcp.License, error)
	GetByID(ctx context.Context, id string) (*lcp.License, error)
	GetByPublication(ctx context.Context, publicationID *string) ([]*lcp.License, error)
	Revoke(ctx context.Context, id string) error
}

type lcpService interface {
	GenerateLicense(ctx context.Context, license *lcp.License) error
	RevokeLicense(ctx context.Context, id string) error
}

type licenseUsecase struct {
	repo    lcp.LicenseRepository
	pubs    lcp.PublicationRepository
	users   userdomain.UserRepository
	enc     lcpencrypt.Encrypter
	lcp     lcpService
	baseURL string
	hooks   webhook.Publisher
	audit   auditservice.Recorder
}

func NewLicenseUsecase(repo lcp.LicenseRepository, pubs lcp.PublicationRepository, users userdomain.UserRepository, enc lcpencrypt.Encrypter, lcp *lcplicense.Service, baseURL string, hooks webhook.Publisher, audit auditservice.Recorder) LicenseUsecase {
	if hooks == nil {
		hooks = webhook.NopPublisher{}
	}
	return &licenseUsecase{repo: repo, pubs: pubs, users: users, enc: enc, lcp: lcp, baseURL: baseURL, hooks: hooks, audit: audit}
}

func (u *licenseUsecase) Create(ctx context.Context, input *lcp.LicenseInput) (*lcp.License, error) {
	if input == nil || input.PublicationID == "" || input.UserID == "" || input.Passphrase == "" {
		return nil, fmt.Errorf("publicationID, userID, and passphrase are required")
	}
	if input.StartDate != nil && input.EndDate != nil && input.EndDate.Before(*input.StartDate) {
		return nil, fmt.Errorf("endDate must be after startDate")
	}
	if u.pubs != nil {
		if pub, err := u.pubs.FindByID(ctx, input.PublicationID); err == nil && pub != nil {
			if input.RightPrint == nil {
				input.RightPrint = pub.RightPrint
			}
			if input.RightCopy == nil {
				input.RightCopy = pub.RightCopy
			}
		} else {
			if err := u.pubs.Save(ctx, &lcp.Publication{
				ID:           input.PublicationID,
				Title:        input.PublicationID,
				Status:       "active",
				EncryptedURI: u.baseURL + "/publications/" + input.PublicationID + "/content",
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}); err != nil {
				return nil, err
			}
		}
	}
	if u.users != nil {
		if err := u.users.Ensure(ctx, &userdomain.User{
			ID:    input.UserID,
			Email: input.UserID + "@local",
			Name:  input.UserID,
		}); err != nil {
			return nil, err
		}
	}

	license := &lcp.License{
		ID:             id.New(),
		TenantID:       tenant.IDFromContext(ctx),
		PublicationID:  input.PublicationID,
		UserID:         input.UserID,
		Passphrase:     input.Passphrase,
		Hint:           input.Hint,
		PublicationURL: u.baseURL + "/publications/" + input.PublicationID + "/content",
		RightPrint:     input.RightPrint,
		RightCopy:      input.RightCopy,
		StartDate:      input.StartDate,
		EndDate:        input.EndDate,
		CreatedAt:      time.Now(),
	}

	err := u.lcp.GenerateLicense(ctx, license)
	if err == lcplicense.ErrContentNotFound && u.enc != nil && u.pubs != nil {
		if pub, findErr := u.pubs.FindByID(ctx, input.PublicationID); findErr == nil && pub != nil && pub.FilePath != "" {
			if _, encErr := u.enc.Encrypt(pub.FilePath, pub.ID, pub.Title); encErr == nil {
				err = u.lcp.GenerateLicense(ctx, license)
			} else {
				err = encErr
			}
		}
	}
	if err != nil {
		observability.IncLicensesFailed()
		return nil, err
	}
	if license.ID == "" {
		license.ID = id.New()
	}

	// Save license to database
	err = u.repo.Save(ctx, license)
	if err != nil {
		observability.IncLicensesFailed()
		return nil, err
	}
	observability.IncLicensesOK()
	_ = u.hooks.Publish(ctx, webhook.Event{
		Type:      webhook.EventLicenseCreated,
		CreatedAt: time.Now(),
		Data: map[string]string{
			"id":            license.ID,
			"publicationID": license.PublicationID,
			"userID":        license.UserID,
		},
	})
	if u.audit != nil {
		_ = u.audit.Record(ctx, "license.created", "license", license.ID)
	}

	return license, nil
}

func (u *licenseUsecase) GetByID(ctx context.Context, id string) (*lcp.License, error) {
	lic, err := u.repo.FindByID(ctx, id)
	if err != nil || lic == nil {
		return lic, err
	}
	if lic.TenantID != "" && lic.TenantID != tenant.IDFromContext(ctx) {
		return nil, nil
	}
	return lic, nil
}

func (u *licenseUsecase) GetByPublication(ctx context.Context, publicationID *string) ([]*lcp.License, error) {
	licenses, err := u.repo.FindByPublication(ctx, publicationID)
	if err != nil {
		return nil, err
	}
	tenantID := tenant.IDFromContext(ctx)
	filtered := make([]*lcp.License, 0, len(licenses))
	for _, lic := range licenses {
		if lic.TenantID == "" || lic.TenantID == tenantID {
			filtered = append(filtered, lic)
		}
	}
	return filtered, nil
}

func (u *licenseUsecase) Revoke(ctx context.Context, id string) error {
	if err := u.lcp.RevokeLicense(ctx, id); err != nil {
		return err
	}
	if err := u.repo.Delete(ctx, id); err != nil {
		return err
	}
	_ = u.hooks.Publish(ctx, webhook.Event{
		Type:      webhook.EventLicenseRevoked,
		CreatedAt: time.Now(),
		Data:      map[string]string{"id": id},
	})
	if u.audit != nil {
		_ = u.audit.Record(ctx, "license.revoked", "license", id)
	}
	return nil
}
