package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/Mehrbod2002/lcp/internal/adapter/graphql"
	"github.com/Mehrbod2002/lcp/internal/adapter/repository/lcp"
	"github.com/Mehrbod2002/lcp/internal/adapter/rest"
	"github.com/Mehrbod2002/lcp/internal/auth"
	"github.com/Mehrbod2002/lcp/internal/config"
	lcpencrypt "github.com/Mehrbod2002/lcp/internal/lcp/encrypt"
	lcplicense "github.com/Mehrbod2002/lcp/internal/lcp/license"
	"github.com/Mehrbod2002/lcp/internal/usecase/lcp/license"
	"github.com/Mehrbod2002/lcp/internal/usecase/lcp/publication"
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
		defer db.Close()
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
	pubUsecase := publication.NewPublicationUsecase(pubRepo, lcpEnc)
	publicBaseURL := buildBaseURL(cfg)
	licUsecase := license.NewLicenseUsecase(licRepo, lcpSrv, publicBaseURL)
	authn := auth.New(cfg.JWT.Secret, cfg.JWT.Admin2FACode)
	restHandler := rest.NewHandler(pubRepo, pubUsecase)
	authHandler := rest.NewAuthHandler(cfg.JWT.Secret, cfg.Admin.Username, cfg.Admin.Password, cfg.Publisher.Username, cfg.Publisher.Password, cfg.JWT.Admin2FACode)
	publicationHandler := rest.NewPublicationHandler(pubRepo, pubUsecase)
	userStore := rest.NewAdminUserStore(cfg.DataDir)
	adminUsersHandler := rest.NewAdminUsersHandler(userStore)

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
	mux.HandleFunc("/api/v1/licenses/", rest.LicenseUserData(licUsecase))
	mux.Handle("/api/v1/publications", authn.RequireRole("admin", "publisher", "user", "guest")(publicationHandler))
	mux.Handle("/api/v1/publications/", authn.RequireRole("admin", "publisher", "user", "guest")(publicationHandler))

	mux.Handle("/api/v1/lcp/process", authn.RequireRole("admin", "publisher", "user")(http.HandlerFunc(restHandler.Process)))
	mux.Handle("/api/v1/lcp/status", authn.RequireRole("admin", "publisher", "user", "guest")(http.HandlerFunc(restHandler.Status)))
	mux.Handle("/api/v1/admin/metrics", authn.RequireRole("admin")(http.HandlerFunc(restHandler.Metrics)))
	mux.Handle("/api/v1/admin/users", authn.RequireRole("admin")(adminUsersHandler))
	mux.Handle("/api/v1/admin/users/", authn.RequireRole("admin")(adminUsersHandler))

	gqlHandler := graphql.NewHandler(&graphql.Resolver{
		PublicationUsecase: pubUsecase,
		LicenseUsecase:     licUsecase,
		PublicBaseURL:      publicBaseURL,
	})
	mux.Handle("/graphql", authn.RequireRole("admin", "publisher", "user")(gqlHandler))
	mux.Handle("/publications/", authn.RequireRole("admin", "publisher", "user", "guest")(publicationDownloadHandler(pubUsecase)))

	port := cfg.Server.Port
	if port == "" {
		port = ":8080"
	}

	log.Printf("lcp server listening on %s", port)
	if err := http.ListenAndServe(port, mux); err != nil {
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

func publicationDownloadHandler(pubUsecase publication.PublicationUsecase) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) < 3 || parts[0] != "publications" || parts[2] != "content" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		pubID := parts[1]
		pub, err := pubUsecase.GetByID(context.Background(), pubID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if pub == nil || pub.EncryptedPath == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		http.ServeFile(w, r, pub.EncryptedPath)
	})
}
