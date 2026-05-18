package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/amirhdev/ebook-lcp-server/internal/config"
	"github.com/amirhdev/ebook-lcp-server/internal/domain/lcp"
)

type fakePublicationUsecase struct {
	pub *lcp.Publication
}

func (u fakePublicationUsecase) UploadAndEncrypt(context.Context, string, io.Reader) (*lcp.Publication, error) {
	return nil, nil
}

func (u fakePublicationUsecase) GetAll(context.Context) ([]*lcp.Publication, error) {
	return []*lcp.Publication{u.pub}, nil
}

func (u fakePublicationUsecase) GetByID(context.Context, string) (*lcp.Publication, error) {
	return u.pub, nil
}

type fakeSignedStorage struct{}

func (fakeSignedStorage) StoreEncrypted(context.Context, string, string) (string, error) {
	return "", nil
}

func (fakeSignedStorage) OpenEncrypted(context.Context, string) (io.ReadCloser, error) {
	return nil, nil
}

func (fakeSignedStorage) SignedURL(context.Context, string, time.Duration) (string, bool, error) {
	return "http://localhost:9000/books/publications/book.epub?signature=ok", true, nil
}

func (fakeSignedStorage) Ready(context.Context) error {
	return nil
}

func TestPublicationDownloadHandlerRedirectsToSignedURL(t *testing.T) {
	cfg := &config.Config{}
	cfg.LCP.Storage.S3.SignedURLTTLSecs = 900
	handler := publicationDownloadHandler(fakePublicationUsecase{
		pub: &lcp.Publication{
			ID:           "book",
			EncryptedURI: "s3://books/publications/book.epub",
		},
	}, fakeSignedStorage{}, cfg)

	req := httptest.NewRequest(http.MethodGet, "/publications/book/content", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected %d, got %d", http.StatusTemporaryRedirect, rec.Code)
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "signature=ok") {
		t.Fatalf("unexpected redirect location: %s", location)
	}
}

func TestPublicationDownloadHandlerAcceptsEpubPathFromLCPL(t *testing.T) {
	cfg := &config.Config{}
	cfg.LCP.Storage.S3.SignedURLTTLSecs = 900
	handler := publicationDownloadHandler(fakePublicationUsecase{
		pub: &lcp.Publication{
			ID:           "book",
			EncryptedURI: "s3://books/publications/book.epub",
		},
	}, fakeSignedStorage{}, cfg)

	req := httptest.NewRequest(http.MethodGet, "/publications/book.epub", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected %d, got %d", http.StatusTemporaryRedirect, rec.Code)
	}
}
