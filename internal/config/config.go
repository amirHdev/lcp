package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Database struct {
		DSN string
	}
	LCP struct {
		Profile     string // "basic" or "production"
		Certificate string // Path to X.509 certificate
		PrivateKey  string // Path to private key
		ProviderURI string
		CoreURL     string
		CoreUser    string
		CorePass    string
		Storage     struct {
			Mode string // "fs" or "s3"
			FS   struct {
				Directory string
			}
			S3 struct {
				Endpoint         string
				PublicEndpoint   string
				Region           string
				Bucket           string
				AccessKey        string
				SecretKey        string
				UseSSL           bool
				SignedURLTTLSecs int
			}
		}
	}
	JWT struct {
		Secret       string
		Admin2FACode string
	}
	Admin struct {
		Username string
		Password string
	}
	Publisher struct {
		Username string
		Password string
	}
	Tenant struct {
		DefaultID string
	}
	Server struct {
		Port          string
		PublicBaseURL string
		StatusBaseURL string
		RateLimitRPM  int
	}
	Webhooks struct {
		URLs           []string
		Secret         string
		MaxAttempts    int
		RetryBackoffMS int
	}
	DataDir string
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}
	cfg.Database.DSN = os.Getenv("DB_DSN")
	cfg.LCP.Profile = os.Getenv("LCP_PROFILE")
	cfg.LCP.Certificate = os.Getenv("LCP_CERTIFICATE")
	cfg.LCP.PrivateKey = os.Getenv("LCP_PRIVATE_KEY")
	cfg.LCP.ProviderURI = os.Getenv("LCP_PROVIDER_URI")
	cfg.LCP.CoreURL = os.Getenv("LCP_CORE_URL")
	cfg.LCP.CoreUser = os.Getenv("LCP_CORE_USER")
	cfg.LCP.CorePass = os.Getenv("LCP_CORE_PASSWORD")
	cfg.LCP.Storage.Mode = os.Getenv("LCP_STORAGE_MODE")
	cfg.LCP.Storage.FS.Directory = os.Getenv("LCP_STORAGE_FS_DIR")
	cfg.LCP.Storage.S3.Region = os.Getenv("LCP_S3_REGION")
	cfg.LCP.Storage.S3.Endpoint = os.Getenv("LCP_S3_ENDPOINT")
	cfg.LCP.Storage.S3.PublicEndpoint = os.Getenv("LCP_S3_PUBLIC_ENDPOINT")
	cfg.LCP.Storage.S3.Bucket = os.Getenv("LCP_S3_BUCKET")
	cfg.LCP.Storage.S3.AccessKey = os.Getenv("LCP_S3_ACCESS_KEY")
	cfg.LCP.Storage.S3.SecretKey = os.Getenv("LCP_S3_SECRET_KEY")
	cfg.LCP.Storage.S3.UseSSL = os.Getenv("LCP_S3_USE_SSL") == "true"
	cfg.LCP.Storage.S3.SignedURLTTLSecs = envInt("LCP_S3_SIGNED_URL_TTL_SECONDS", 900)
	cfg.JWT.Secret = os.Getenv("JWT_SECRET")
	cfg.JWT.Admin2FACode = os.Getenv("ADMIN_2FA_CODE")
	cfg.Admin.Username = os.Getenv("ADMIN_USERNAME")
	cfg.Admin.Password = os.Getenv("ADMIN_PASSWORD")
	cfg.Publisher.Username = os.Getenv("PUBLISHER_USERNAME")
	cfg.Publisher.Password = os.Getenv("PUBLISHER_PASSWORD")
	cfg.Tenant.DefaultID = defaultString(os.Getenv("DEFAULT_TENANT_ID"), "default")
	cfg.Server.Port = os.Getenv("SERVER_PORT")
	cfg.Server.PublicBaseURL = os.Getenv("PUBLIC_BASE_URL")
	cfg.Server.StatusBaseURL = os.Getenv("STATUS_BASE_URL")
	cfg.Server.RateLimitRPM = envInt("RATE_LIMIT_RPM", 600)
	cfg.Webhooks.URLs = splitCSV(os.Getenv("WEBHOOK_URLS"))
	cfg.Webhooks.Secret = os.Getenv("WEBHOOK_SECRET")
	cfg.Webhooks.MaxAttempts = envInt("WEBHOOK_MAX_ATTEMPTS", 3)
	cfg.Webhooks.RetryBackoffMS = envInt("WEBHOOK_RETRY_BACKOFF_MS", 250)
	cfg.DataDir = os.Getenv("DATA_DIR")
	return cfg, nil
}

func envInt(name string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(name))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		if item := strings.TrimSpace(part); item != "" {
			items = append(items, item)
		}
	}
	return items
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}
