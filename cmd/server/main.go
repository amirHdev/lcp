package main

import (
	"context"
	"database/sql"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/amirhdev/ebook-lcp-server/internal/adapter/graphql"
	auditrepo "github.com/amirhdev/ebook-lcp-server/internal/adapter/repository/audit"
	"github.com/amirhdev/ebook-lcp-server/internal/adapter/repository/lcp"
	"github.com/amirhdev/ebook-lcp-server/internal/adapter/rest"
	auditservice "github.com/amirhdev/ebook-lcp-server/internal/audit"
	"github.com/amirhdev/ebook-lcp-server/internal/auth"
	"github.com/amirhdev/ebook-lcp-server/internal/config"
	userdomain "github.com/amirhdev/ebook-lcp-server/internal/domain"
	lcpencrypt "github.com/amirhdev/ebook-lcp-server/internal/lcp/encrypt"
	lcplicense "github.com/amirhdev/ebook-lcp-server/internal/lcp/license"
	"github.com/amirhdev/ebook-lcp-server/internal/ratelimit"
	"github.com/amirhdev/ebook-lcp-server/internal/requestmeta"
	publicationstorage "github.com/amirhdev/ebook-lcp-server/internal/storage"
	"github.com/amirhdev/ebook-lcp-server/internal/usecase/lcp/license"
	"github.com/amirhdev/ebook-lcp-server/internal/usecase/lcp/publication"
	"github.com/amirhdev/ebook-lcp-server/internal/webhook"
)

// @title LCP License Server API
// @version 1.0
// @description API for managing LCP licenses and publications
// @host localhost:8080
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}

	lcpEnc := lcpencrypt.NewReadiumEncrypter(cfg)
	lcpSrv := lcplicense.NewService(
		cfg.LCP.CoreURL,
		cfg.LCP.CoreUser,
		cfg.LCP.CorePass,
		buildStatusBaseURL(cfg),
		cfg.LCP.CoreUser,
		cfg.LCP.CorePass,
		buildProviderURI(cfg),
	)
	db, err := buildDatabase(cfg)
	if err != nil {
		panic(err)
	}
	if db != nil {
		defer func() {
			if err := db.Close(); err != nil {
				log.Printf("close rows: %v", err)
			}
		}()
		if err := lcp.EnsurePostgresSchema(context.Background(), db); err != nil {
			panic(err)
		}
	}
	pubRepo, err := buildPublicationRepository(cfg, db)
	if err != nil {
		panic(err)
	}
	licRepo, err := buildLicenseRepository(cfg, db)
	if err != nil {
		panic(err)
	}
	auditRepo, err := buildAuditRepository(cfg, db)
	if err != nil {
		panic(err)
	}
	tenantStore := rest.NewTenantStore(cfg.DataDir, cfg.Tenant.DefaultID)
	publicationStorage, err := buildPublicationStorage(cfg, tenantStore)
	if err != nil {
		panic(err)
	}
	webhookPublisher := buildWebhookPublisher(cfg, tenantStore)
	auditSvc := auditservice.NewService(auditRepo)
	pubUsecase := publication.NewPublicationUsecase(pubRepo, lcpEnc, publicationStorage, webhookPublisher, auditSvc)
	publicBaseURL := buildBaseURL(cfg)
	licUsecase := license.NewLicenseUsecase(licRepo, pubRepo, buildUserRepository(cfg, db), lcpEnc, lcpSrv, publicBaseURL, webhookPublisher, auditSvc)
	authn := auth.New(cfg.JWT.Secret, cfg.JWT.Admin2FACode, ratelimit.New(cfg.Server.RateLimitRPM, time.Minute)).
		WithAPIKeys(buildAPIKeyResolver(tenantStore)).
		WithTenantRateLimits(buildTenantRateLimitResolver(tenantStore))
	restHandler := rest.NewHandler(pubRepo, pubUsecase, buildReadyCheck(db, publicationStorage))
	authHandler := rest.NewAuthHandler(cfg.JWT.Secret, cfg.Admin.Username, cfg.Admin.Password, cfg.Publisher.Username, cfg.Publisher.Password, cfg.JWT.Admin2FACode, cfg.Tenant.DefaultID)
	publicationHandler := rest.NewPublicationHandler(pubRepo, pubUsecase)
	userStore := rest.NewAdminUserStore(cfg.DataDir)
	adminUsersHandler := rest.NewAdminUsersHandler(userStore)
	adminTenantsHandler := rest.NewAdminTenantsHandler(tenantStore)
	auditHandler := rest.NewAuditHandler(auditRepo)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("/api/v1/auth/ping", authHandler.Ping)
	mux.HandleFunc("/swagger.yaml", rest.SwaggerYAML())
	mux.HandleFunc("/swagger.json", rest.SwaggerJSON())
	mux.HandleFunc("/docs/openapi.yaml", rest.OpenAPIYAML())
	mux.HandleFunc("/docs/swagger.json", rest.SwaggerJSON())
	mux.HandleFunc("/healthz", restHandler.Healthz)
	mux.HandleFunc("/readyz", restHandler.Readyz)
	mux.HandleFunc("/metrics", restHandler.PrometheusMetrics)
	licenseDownloadHandler := rest.NewLicenseDownloadHandler(licUsecase, lcpSrv)
	licenseStatusHandler := rest.LicenseStatusDocument(licUsecase)
	mux.HandleFunc("/licenses/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/status") {
			licenseStatusHandler(w, r)
			return
		}
		licenseDownloadHandler.ServeHTTP(w, r)
	})
	mux.HandleFunc("/api/v1/licenses/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/user") {
			rest.LicenseUserData(licUsecase)(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/lcpl") {
			licenseDownloadHandler.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	})
	mux.Handle("/api/v1/publications", authn.RequireRole("admin", "publisher", "user", "guest")(publicationHandler))
	mux.Handle("/api/v1/publications/", authn.RequireRole("admin", "publisher", "user", "guest")(publicationHandler))

	mux.Handle("/api/v1/lcp/process", authn.RequireRole("admin", "publisher", "user")(http.HandlerFunc(restHandler.Process)))
	mux.Handle("/api/v1/lcp/status", authn.RequireRole("admin", "publisher", "user", "guest")(http.HandlerFunc(restHandler.Status)))
	mux.Handle("/api/v1/admin/metrics", authn.RequireRole("admin")(http.HandlerFunc(restHandler.Metrics)))
	mux.Handle("/api/v1/admin/users", authn.RequireRole("admin")(adminUsersHandler))
	mux.Handle("/api/v1/admin/users/", authn.RequireRole("admin")(adminUsersHandler))
	mux.Handle("/api/v1/admin/tenants", authn.RequireRole("admin")(adminTenantsHandler))
	mux.Handle("/api/v1/admin/tenants/", authn.RequireRole("admin")(adminTenantsHandler))
	mux.Handle("/api/v1/admin/audit", authn.RequireRole("admin")(auditHandler))

	gqlHandler := graphql.NewHandler(&graphql.Resolver{
		PublicationUsecase: pubUsecase,
		LicenseUsecase:     licUsecase,
		PublicBaseURL:      publicBaseURL,
	})
	mux.Handle("/graphql", authn.RequireRole("admin", "publisher", "user")(gqlHandler))
	mux.Handle("/publications/", publicationDownloadHandler(pubUsecase, publicationStorage, cfg))

	port := cfg.Server.Port
	if port == "" {
		port = ":8080"
	}

	slog.Info("server_listening", "port", port)
	if err := http.ListenAndServe(port, requestmeta.Middleware(logger)(mux)); err != nil {
		panic(err)
	}
}

func buildDatabase(cfg *config.Config) (*sql.DB, error) {
	if strings.TrimSpace(cfg.Database.DSN) == "" {
		return nil, nil
	}
	return lcp.OpenPostgres(context.Background(), cfg.Database.DSN)
}

func buildPublicationRepository(cfg *config.Config, db *sql.DB) (lcp.PublicationRepository, error) {
	if db != nil {
		return lcp.NewPostgresPublicationRepository(db), nil
	}
	if cfg.DataDir == "" {
		return lcp.NewPublicationRepository(), nil
	}
	return lcp.NewPersistentPublicationRepository(filepath.Join(cfg.DataDir, "publications.json"))
}

func buildLicenseRepository(cfg *config.Config, db *sql.DB) (lcp.LicenseRepository, error) {
	if db != nil {
		return lcp.NewPostgresLicenseRepository(db), nil
	}
	if cfg.DataDir == "" {
		return lcp.NewLicenseRepository(), nil
	}
	return lcp.NewPersistentLicenseRepository(filepath.Join(cfg.DataDir, "licenses.json"))
}

func buildAuditRepository(cfg *config.Config, db *sql.DB) (auditrepo.Repository, error) {
	if db != nil {
		return lcp.NewPostgresAuditRepository(db), nil
	}
	if cfg.DataDir == "" {
		return auditrepo.NewRepository(), nil
	}
	return auditrepo.NewPersistentRepository(filepath.Join(cfg.DataDir, "audit.json"))
}

func buildUserRepository(cfg *config.Config, db *sql.DB) userdomain.UserRepository {
	if db != nil {
		return lcp.NewPostgresUserRepository(db)
	}
	return nil
}

func buildPublicationStorage(cfg *config.Config, tenants *rest.TenantStore) (publicationstorage.PublicationStorage, error) {
	if strings.EqualFold(strings.TrimSpace(cfg.LCP.Storage.Mode), "s3") {
		store, err := publicationstorage.NewS3PublicationStorage(cfg)
		if err != nil {
			return nil, err
		}
		return store.WithPrefixResolver(func(ctx context.Context) string {
			if tenant, ok := tenants.Get(tenantIDFromContext(ctx)); ok {
				return tenant.StoragePrefix
			}
			return ""
		}), nil
	}
	return publicationstorage.NewFilesystemPublicationStorage(), nil
}

func buildReadyCheck(db *sql.DB, store publicationstorage.PublicationStorage) func(context.Context) error {
	return func(ctx context.Context) error {
		if db != nil {
			if err := db.PingContext(ctx); err != nil {
				return err
			}
		}
		if store != nil {
			return store.Ready(ctx)
		}
		return nil
	}
}

func buildWebhookFailureRecorder(cfg *config.Config) webhook.FailureRecorder {
	if strings.TrimSpace(cfg.DataDir) == "" {
		return &webhook.MemoryFailureRecorder{}
	}
	return webhook.NewPersistentFailureRecorder(filepath.Join(cfg.DataDir, "webhook-failures.json"))
}

func buildWebhookPublisher(cfg *config.Config, tenants *rest.TenantStore) webhook.Publisher {
	publisher := webhook.NewHTTPPublisherWithOptions(
		cfg.Webhooks.URLs,
		cfg.Webhooks.Secret,
		cfg.Webhooks.MaxAttempts,
		time.Duration(cfg.Webhooks.RetryBackoffMS)*time.Millisecond,
		buildWebhookFailureRecorder(cfg),
	)
	httpPublisher, ok := publisher.(*webhook.HTTPPublisher)
	if !ok {
		return publisher
	}
	return httpPublisher.WithURLResolver(func(ctx context.Context) []string {
		if tenant, ok := tenants.Get(tenantIDFromContext(ctx)); ok {
			return tenant.WebhookURLs
		}
		return nil
	})
}

func buildAPIKeyResolver(tenants *rest.TenantStore) auth.APIKeyResolver {
	return func(key string) (*auth.Claims, bool) {
		for _, tenant := range tenants.List() {
			for _, apiKey := range tenant.APIKeys {
				if apiKey.Key == key {
					role := strings.TrimSpace(apiKey.Role)
					if role == "" {
						role = "publisher"
					}
					subject := strings.TrimSpace(apiKey.Subject)
					if subject == "" {
						subject = "api-key"
					}
					return &auth.Claims{
						Subject:  subject,
						TenantID: tenant.ID,
						Role:     role,
						Roles:    []string{role},
					}, true
				}
			}
		}
		return nil, false
	}
}

func buildTenantRateLimitResolver(tenants *rest.TenantStore) auth.RateLimitResolver {
	return func(tenantID string) int {
		if tenant, ok := tenants.Get(tenantID); ok {
			return tenant.RateLimitRPM
		}
		return 0
	}
}

func tenantIDFromContext(ctx context.Context) string {
	if claims, ok := auth.FromContext(ctx); ok && claims.TenantID != "" {
		return claims.TenantID
	}
	return "default"
}

func buildBaseURL(cfg *config.Config) string {
	baseURL := strings.TrimSpace(cfg.Server.PublicBaseURL)
	if baseURL != "" {
		return strings.TrimSuffix(baseURL, "/")
	}

	port := cfg.Server.Port
	if port == "" {
		port = ":8080"
	}

	return "http://localhost" + port
}

func buildProviderURI(cfg *config.Config) string {
	if uri := strings.TrimSpace(cfg.LCP.ProviderURI); uri != "" {
		return strings.TrimRight(uri, "/")
	}
	return strings.TrimRight(buildBaseURL(cfg), "/")
}

func buildStatusBaseURL(cfg *config.Config) string {
	if uri := strings.TrimSpace(cfg.Server.StatusBaseURL); uri != "" {
		return strings.TrimRight(uri, "/")
	}
	return ""
}

func publicationDownloadHandler(pubUsecase publication.PublicationUsecase, store publicationstorage.PublicationStorage, cfg *config.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

		pubID := ""
		if len(parts) == 3 && parts[0] == "publications" && parts[2] == "content" {
			pubID = parts[1]
		} else if len(parts) == 2 && parts[0] == "publications" && strings.HasSuffix(parts[1], ".lcpdf") {
			pubID = strings.TrimSuffix(parts[1], ".lcpdf")
		} else if len(parts) == 2 && parts[0] == "publications" && strings.HasSuffix(parts[1], ".epub") {
			pubID = strings.TrimSuffix(parts[1], ".epub")
		} else {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		pub, err := pubUsecase.GetByID(context.Background(), pubID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if pub == nil || pub.EncryptedURI == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if strings.HasPrefix(pub.EncryptedURI, "s3://") {
			expiry := time.Duration(cfg.LCP.Storage.S3.SignedURLTTLSecs) * time.Second
			if signedURL, ok, err := store.SignedURL(r.Context(), pub.EncryptedURI, expiry); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			} else if ok {
				http.Redirect(w, r, signedURL, http.StatusTemporaryRedirect)
				return
			}
			reader, err := store.OpenEncrypted(r.Context(), pub.EncryptedURI)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer func() {
				if err := reader.Close(); err != nil {
					log.Printf("close encrypted object: %v", err)
				}
			}()
			_, _ = io.Copy(w, reader)
			return
		}
		http.ServeFile(w, r, pub.EncryptedURI)
	})
}
