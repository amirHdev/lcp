package publication

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/amirhdev/ebook-lcp-server/internal/domain/lcp"
	"github.com/amirhdev/ebook-lcp-server/internal/lcp/encrypt"
	"github.com/amirhdev/ebook-lcp-server/internal/pkg/id"
)

type PublicationUsecase interface {
	UploadAndEncrypt(ctx context.Context, title string, file io.Reader) (*lcp.Publication, error)
	GetAll(ctx context.Context) ([]*lcp.Publication, error)
	GetByID(ctx context.Context, id string) (*lcp.Publication, error)
}

type publicationUsecase struct {
	repo lcp.PublicationRepository
	enc  encrypt.Encrypter
}

func NewPublicationUsecase(repo lcp.PublicationRepository, enc encrypt.Encrypter) PublicationUsecase {
	return &publicationUsecase{repo, enc}
}

func (u *publicationUsecase) UploadAndEncrypt(ctx context.Context, title string, file io.Reader) (*lcp.Publication, error) {
	if title == "" || file == nil {
		return nil, fmt.Errorf("title and file are required")
	}

	pubID := id.New()
	raw, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	tempExt := detectPublicationExt(raw)
	tempPath := filepath.Join(os.TempDir(), pubID+tempExt)
	out, err := os.Create(tempPath)
	if err != nil {
		return nil, err
	}
	if _, err := out.Write(raw); err != nil {
		_ = out.Close()
		return nil, err
	}
	if err := out.Close(); err != nil {
		return nil, err
	}

	// Encrypt using lcpencrypt
	encryptedPath, err := u.enc.Encrypt(tempPath, pubID, title)
	if err != nil {
		return nil, err
	}
	sourcePath := filepath.Join(filepath.Dir(encryptedPath), pubID+tempExt)
	if err := os.WriteFile(sourcePath, raw, 0o644); err != nil {
		return nil, err
	}

	// Store publication metadata
	pub := &lcp.Publication{
		ID:            pubID,
		Title:         title,
		Status:        "active",
		FilePath:      sourcePath,
		EncryptedPath: encryptedPath,
		EncryptedURI:  encryptedPath,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	err = u.repo.Save(ctx, pub)
	if err != nil {
		return nil, err
	}

	return pub, nil
}

func detectPublicationExt(raw []byte) string {
	if bytes.HasPrefix(raw, []byte("%PDF-")) {
		return ".pdf"
	}
	if strings.HasPrefix(httpDetectContentType(raw), "application/zip") {
		if ext := detectZipPublicationExt(raw); ext != "" {
			return ext
		}
		return ".epub"
	}
	return ".pdf"
}

func detectZipPublicationExt(raw []byte) string {
	readerAt := bytes.NewReader(raw)
	zr, err := zip.NewReader(readerAt, int64(len(raw)))
	if err != nil {
		return ""
	}
	for _, f := range zr.File {
		if f.Name == "mimetype" {
			rc, err := f.Open()
			if err != nil {
				return ""
			}
			defer func() {
				if err := rc.Close(); err != nil {
					log.Printf("close rows: %v", err)
				}
			}()
			mimeBytes, err := io.ReadAll(rc)
			if err != nil {
				return ""
			}
			if string(mimeBytes) == "application/epub+zip" {
				return ".epub"
			}
		}
	}
	return ".epub"
}

func httpDetectContentType(raw []byte) string {
	if len(raw) > 512 {
		raw = raw[:512]
	}
	return http.DetectContentType(raw)
}

func (u *publicationUsecase) GetAll(ctx context.Context) ([]*lcp.Publication, error) {
	return u.repo.FindAll(ctx)
}

func (u *publicationUsecase) GetByID(ctx context.Context, id string) (*lcp.Publication, error) {
	return u.repo.FindByID(ctx, id)
}
