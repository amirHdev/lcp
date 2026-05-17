package publication

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	domain "github.com/amirhdev/ebook-lcp-server/internal/domain/lcp"
)

type fakeRepo struct {
	saved *domain.Publication
}

func (r *fakeRepo) Save(_ context.Context, pub *domain.Publication) error {
	r.saved = pub
	return nil
}

func (r *fakeRepo) FindAll(_ context.Context) ([]*domain.Publication, error) {
	return []*domain.Publication{r.saved}, nil
}

func (r *fakeRepo) FindByID(_ context.Context, id string) (*domain.Publication, error) {
	if r.saved != nil && r.saved.ID == id {
		return r.saved, nil
	}
	return nil, nil
}

type fakeEncrypter struct {
	dir string
}

func (e fakeEncrypter) Encrypt(inputPath, contentID, filename string) (string, error) {
	out := filepath.Join(e.dir, contentID+".lcpl")
	return out, os.WriteFile(out, []byte("encrypted"), 0o644)
}

func TestUploadAndEncryptCreatesPublication(t *testing.T) {
	dir := t.TempDir()

	repo := &fakeRepo{}
	uc := NewPublicationUsecase(repo, fakeEncrypter{dir: dir})

	pub, err := uc.UploadAndEncrypt(
		context.Background(),
		"Book",
		strings.NewReader("%PDF- fake pdf"),
	)
	if err != nil {
		t.Fatalf("UploadAndEncrypt failed: %v", err)
	}

	if pub.ID == "" {
		t.Fatal("expected publication ID")
	}
	if pub.Title != "Book" {
		t.Fatalf("unexpected title: %s", pub.Title)
	}
	if pub.Status != "active" {
		t.Fatalf("unexpected status: %s", pub.Status)
	}
	if repo.saved == nil {
		t.Fatal("expected publication to be saved")
	}
	if pub.FilePath == "" || pub.EncryptedPath == "" {
		t.Fatal("expected file paths to be set")
	}
}

func TestUploadAndEncryptRejectsMissingInput(t *testing.T) {
	uc := NewPublicationUsecase(&fakeRepo{}, fakeEncrypter{dir: t.TempDir()})

	_, err := uc.UploadAndEncrypt(context.Background(), "", strings.NewReader("x"))
	if err == nil {
		t.Fatal("expected error for missing title")
	}

	_, err = uc.UploadAndEncrypt(context.Background(), "Book", nil)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestGetAllAndGetByID(t *testing.T) {
	repo := &fakeRepo{
		saved: &domain.Publication{
			ID:    "pub1",
			Title: "Book",
		},
	}

	uc := NewPublicationUsecase(repo, fakeEncrypter{dir: t.TempDir()})

	all, err := uc.GetAll(context.Background())
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 publication, got %d", len(all))
	}

	found, err := uc.GetByID(context.Background(), "pub1")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if found == nil {
		t.Fatal("expected publication")
	}
}
