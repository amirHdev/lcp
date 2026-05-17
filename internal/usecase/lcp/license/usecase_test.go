// internal/usecase/lcp/license/usecase_test.go
package license

import (
	"context"
	"testing"

	userdomain "github.com/amirhdev/ebook-lcp-server/internal/domain"
	domain "github.com/amirhdev/ebook-lcp-server/internal/domain/lcp"
)

type fakeLicenseRepo struct {
	saved *domain.License
}

func (r *fakeLicenseRepo) Save(_ context.Context, lic *domain.License) error {
	r.saved = lic
	return nil
}

func (r *fakeLicenseRepo) FindByID(_ context.Context, id string) (*domain.License, error) {
	if r.saved != nil && r.saved.ID == id {
		return r.saved, nil
	}
	return nil, nil
}

func (r *fakeLicenseRepo) FindByPublication(_ context.Context, publicationID *string) ([]*domain.License, error) {
	if r.saved == nil {
		return nil, nil
	}
	return []*domain.License{r.saved}, nil
}

func (r *fakeLicenseRepo) Delete(_ context.Context, id string) error {
	r.saved = nil
	return nil
}

type fakePublicationRepo struct {
	pub *domain.Publication
}

func (r *fakePublicationRepo) Save(_ context.Context, pub *domain.Publication) error {
	r.pub = pub
	return nil
}

func (r *fakePublicationRepo) FindAll(_ context.Context) ([]*domain.Publication, error) {
	if r.pub == nil {
		return nil, nil
	}
	return []*domain.Publication{r.pub}, nil
}

func (r *fakePublicationRepo) FindByID(_ context.Context, id string) (*domain.Publication, error) {
	if r.pub != nil && r.pub.ID == id {
		return r.pub, nil
	}
	return nil, nil
}

type fakeUserRepo struct{}

func (r *fakeUserRepo) Ensure(_ context.Context, _ *userdomain.User) error {
	return nil
}

func TestCreateRejectsInvalidInput(t *testing.T) {
	uc := &licenseUsecase{}

	_, err := uc.Create(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetByIDAndPublication(t *testing.T) {
	repo := &fakeLicenseRepo{
		saved: &domain.License{
			ID:            "lic1",
			PublicationID: "pub1",
		},
	}

	uc := &licenseUsecase{
		repo: repo,
	}

	lic, err := uc.GetByID(context.Background(), "lic1")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if lic == nil {
		t.Fatal("expected license")
	}

	pubID := "pub1"

	list, err := uc.GetByPublication(context.Background(), &pubID)
	if err != nil {
		t.Fatalf("GetByPublication failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 license, got %d", len(list))
	}
}

type fakeLCPService struct {
	generated *domain.License
	revoked   string
}

func (s *fakeLCPService) GenerateLicense(_ context.Context, lic *domain.License) error {
	s.generated = lic
	lic.LCPL = "fake-lcpl"
	return nil
}

func (s *fakeLCPService) RevokeLicense(_ context.Context, id string) error {
	s.revoked = id
	return nil
}

func TestCreateGeneratesAndSavesLicense(t *testing.T) {
	printRight := 10
	copyRight := 5

	licenseRepo := &fakeLicenseRepo{}
	pubRepo := &fakePublicationRepo{
		pub: &domain.Publication{
			ID:         "pub1",
			Title:      "Book",
			Status:     "active",
			RightPrint: &printRight,
			RightCopy:  &copyRight,
		},
	}
	lcpSvc := &fakeLCPService{}

	uc := &licenseUsecase{
		repo:    licenseRepo,
		pubs:    pubRepo,
		users:   &fakeUserRepo{},
		lcp:     lcpSvc,
		baseURL: "http://localhost",
	}

	lic, err := uc.Create(context.Background(), &domain.LicenseInput{
		PublicationID: "pub1",
		UserID:        "user1",
		Passphrase:    "secret",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if lic.ID == "" {
		t.Fatal("expected license ID")
	}
	if lic.PublicationURL != "http://localhost/publications/pub1/content" {
		t.Fatalf("unexpected publication URL: %s", lic.PublicationURL)
	}
	if lic.RightPrint == nil || *lic.RightPrint != printRight {
		t.Fatal("expected right print inherited from publication")
	}
	if lic.RightCopy == nil || *lic.RightCopy != copyRight {
		t.Fatal("expected right copy inherited from publication")
	}
	if licenseRepo.saved == nil {
		t.Fatal("expected license to be saved")
	}
	if lcpSvc.generated == nil {
		t.Fatal("expected LCP service to generate license")
	}
}

func TestRevokeRevokesAndDeletesLicense(t *testing.T) {
	repo := &fakeLicenseRepo{
		saved: &domain.License{ID: "lic1"},
	}
	lcpSvc := &fakeLCPService{}

	uc := &licenseUsecase{
		repo: repo,
		lcp:  lcpSvc,
	}

	err := uc.Revoke(context.Background(), "lic1")
	if err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}

	if lcpSvc.revoked != "lic1" {
		t.Fatalf("expected revoked lic1, got %s", lcpSvc.revoked)
	}
	if repo.saved != nil {
		t.Fatal("expected license to be deleted")
	}
}
