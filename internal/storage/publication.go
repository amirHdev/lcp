package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/amirhdev/ebook-lcp-server/internal/config"
	"github.com/amirhdev/ebook-lcp-server/internal/observability"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type PublicationStorage interface {
	StoreEncrypted(ctx context.Context, localPath, publicationID string) (string, error)
	OpenEncrypted(ctx context.Context, uri string) (io.ReadCloser, error)
	SignedURL(ctx context.Context, uri string, expiry time.Duration) (string, bool, error)
	Ready(ctx context.Context) error
}

type FilesystemPublicationStorage struct{}

func NewFilesystemPublicationStorage() *FilesystemPublicationStorage {
	return &FilesystemPublicationStorage{}
}

func (s *FilesystemPublicationStorage) StoreEncrypted(_ context.Context, localPath, _ string) (string, error) {
	return localPath, nil
}

func (s *FilesystemPublicationStorage) OpenEncrypted(_ context.Context, uri string) (io.ReadCloser, error) {
	return os.Open(uri)
}

func (s *FilesystemPublicationStorage) SignedURL(_ context.Context, _ string, _ time.Duration) (string, bool, error) {
	return "", false, nil
}

func (s *FilesystemPublicationStorage) Ready(context.Context) error {
	return nil
}

type S3PublicationStorage struct {
	client        *minio.Client
	signingClient *minio.Client
	bucket        string
	prefixFor     func(context.Context) string
}

func NewS3PublicationStorage(cfg *config.Config) (*S3PublicationStorage, error) {
	if cfg == nil {
		return nil, fmt.Errorf("missing storage config")
	}
	endpoint := strings.TrimSpace(cfg.LCP.Storage.S3.Endpoint)
	bucket := strings.TrimSpace(cfg.LCP.Storage.S3.Bucket)
	if endpoint == "" || bucket == "" {
		return nil, fmt.Errorf("s3 endpoint and bucket are required")
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.LCP.Storage.S3.AccessKey, cfg.LCP.Storage.S3.SecretKey, ""),
		Secure: cfg.LCP.Storage.S3.UseSSL,
		Region: cfg.LCP.Storage.S3.Region,
	})
	if err != nil {
		return nil, err
	}
	publicEndpoint := strings.TrimSpace(cfg.LCP.Storage.S3.PublicEndpoint)
	if publicEndpoint == "" {
		publicEndpoint = endpoint
	}
	signingClient, err := minio.New(publicEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.LCP.Storage.S3.AccessKey, cfg.LCP.Storage.S3.SecretKey, ""),
		Secure: cfg.LCP.Storage.S3.UseSSL,
		Region: cfg.LCP.Storage.S3.Region,
	})
	if err != nil {
		return nil, err
	}
	exists, err := client.BucketExists(context.Background(), bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := client.MakeBucket(context.Background(), bucket, minio.MakeBucketOptions{Region: cfg.LCP.Storage.S3.Region}); err != nil {
			return nil, err
		}
	}

	return &S3PublicationStorage{client: client, signingClient: signingClient, bucket: bucket}, nil
}

func (s *S3PublicationStorage) WithPrefixResolver(resolver func(context.Context) string) *S3PublicationStorage {
	s.prefixFor = resolver
	return s
}

func (s *S3PublicationStorage) StoreEncrypted(ctx context.Context, localPath, publicationID string) (string, error) {
	prefix := "publications"
	if s.prefixFor != nil {
		if configured := strings.Trim(strings.TrimSpace(s.prefixFor(ctx)), "/"); configured != "" {
			prefix = configured
		}
	}
	objectKey := filepath.ToSlash(filepath.Join(prefix, publicationID+filepath.Ext(localPath)))
	if _, err := s.client.FPutObject(ctx, s.bucket, objectKey, localPath, minio.PutObjectOptions{}); err != nil {
		observability.IncS3StoreFailed()
		return "", err
	}
	observability.IncS3StoreOK()
	return "s3://" + s.bucket + "/" + objectKey, nil
}

func (s *S3PublicationStorage) OpenEncrypted(ctx context.Context, uri string) (io.ReadCloser, error) {
	bucket, objectKey, err := parseS3URI(uri)
	if err != nil {
		observability.IncS3OpenFailed()
		return nil, err
	}
	object, err := s.client.GetObject(ctx, bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		observability.IncS3OpenFailed()
		return nil, err
	}
	observability.IncS3OpenOK()
	return object, nil
}

func (s *S3PublicationStorage) SignedURL(ctx context.Context, uri string, expiry time.Duration) (string, bool, error) {
	bucket, objectKey, err := parseS3URI(uri)
	if err != nil {
		observability.IncS3SignedURLFail()
		return "", false, err
	}
	url, err := s.signingClient.PresignedGetObject(ctx, bucket, objectKey, expiry, nil)
	if err != nil {
		observability.IncS3SignedURLFail()
		return "", false, err
	}
	observability.IncS3SignedURLOK()
	return url.String(), true, nil
}

func (s *S3PublicationStorage) Ready(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("s3 bucket %q is missing", s.bucket)
	}
	return nil
}

func parseS3URI(uri string) (string, string, error) {
	const prefix = "s3://"
	if !strings.HasPrefix(uri, prefix) {
		return "", "", fmt.Errorf("invalid s3 uri")
	}
	trimmed := strings.TrimPrefix(uri, prefix)
	bucket, objectKey, ok := strings.Cut(trimmed, "/")
	if !ok || bucket == "" || objectKey == "" {
		return "", "", fmt.Errorf("invalid s3 uri")
	}
	return bucket, objectKey, nil
}
