package license

import (
	"context"
	"fmt"
	"time"

	userdomain "github.com/Mehrbod2002/lcp/internal/domain"
	"github.com/Mehrbod2002/lcp/internal/domain/lcp"
	lcpencrypt "github.com/Mehrbod2002/lcp/internal/lcp/encrypt"
	lcplicense "github.com/Mehrbod2002/lcp/internal/lcp/license"
	"github.com/Mehrbod2002/lcp/internal/pkg/id"
)

type LicenseUsecase interface {
	Create(ctx context.Context, input *lcp.LicenseInput) (*lcp.License, error)
	GetByID(ctx context.Context, id string) (*lcp.License, error)
	GetByPublication(ctx context.Context, publicationID *string) ([]*lcp.License, error)
	Revoke(ctx context.Context, id string) error
}

type licenseUsecase struct {
	repo    lcp.LicenseRepository
	pubs    lcp.PublicationRepository
	users   userdomain.UserRepository
	enc     lcpencrypt.Encrypter
	lcp     *lcplicense.Service
	baseURL string
}

func NewLicenseUsecase(repo lcp.LicenseRepository, pubs lcp.PublicationRepository, users userdomain.UserRepository, enc lcpencrypt.Encrypter, lcp *lcplicense.Service, baseURL string) LicenseUsecase {
	return &licenseUsecase{repo: repo, pubs: pubs, users: users, enc: enc, lcp: lcp, baseURL: baseURL}
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
		return nil, err
	}
	if license.ID == "" {
		license.ID = id.New()
	}

	// Save license to database
	err = u.repo.Save(ctx, license)
	if err != nil {
		return nil, err
	}

	return license, nil
}

func (u *licenseUsecase) GetByID(ctx context.Context, id string) (*lcp.License, error) {
	return u.repo.FindByID(ctx, id)
}

func (u *licenseUsecase) GetByPublication(ctx context.Context, publicationID *string) ([]*lcp.License, error) {
	return u.repo.FindByPublication(ctx, publicationID)
}

func (u *licenseUsecase) Revoke(ctx context.Context, id string) error {
	if err := u.lcp.RevokeLicense(ctx, id); err != nil {
		return err
	}
	return u.repo.Delete(ctx, id)
}
