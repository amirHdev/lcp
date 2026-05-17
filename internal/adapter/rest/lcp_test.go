package rest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/amirhdev/ebook-lcp-server/internal/domain/lcp"
)

type fakePublicationUsecase struct{}

type fakePublicationRepo struct{}

func (fakePublicationUsecase) UploadAndEncrypt(_ context.Context, title string, file io.Reader) (*lcp.Publication, error) {
	return &lcp.Publication{
		ID:            "pub-1",
		Title:         title,
		FilePath:      "/tmp/source",
		EncryptedPath: "/tmp/encrypted",
		CreatedAt:     time.Now(),
	}, nil
}

func (fakePublicationUsecase) GetAll(_ context.Context) ([]*lcp.Publication, error) {
	return nil, nil
}

func (fakePublicationUsecase) GetByID(_ context.Context, id string) (*lcp.Publication, error) {
	return nil, nil
}

func (fakePublicationRepo) Save(ctx context.Context, pub *lcp.Publication) error {
	return nil
}

func (fakePublicationRepo) FindAll(ctx context.Context) ([]*lcp.Publication, error) {
	return nil, nil
}

func (fakePublicationRepo) FindByID(ctx context.Context, id string) (*lcp.Publication, error) {
	return nil, nil
}

func TestProcessCreatesPublication(t *testing.T) {
	handler := NewHandler(fakePublicationRepo{}, fakePublicationUsecase{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lcp/process", strings.NewReader(`{"title":"Book","file":"aGVsbG8="}`))
	rec := httptest.NewRecorder()

	handler.Process(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
	}
	var status ProcessStatus
	if err := json.NewDecoder(rec.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if status.Status != "completed" || status.PublicationID != "pub-1" {
		t.Fatalf("unexpected process status: %#v", status)
	}
}

func TestProcessRejectsInvalidPayload(t *testing.T) {
	handler := NewHandler(fakePublicationRepo{}, fakePublicationUsecase{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lcp/process", strings.NewReader(`{"file":"aGVsbG8="}`))
	rec := httptest.NewRecorder()

	handler.Process(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status %d", rec.Code)
	}
}
