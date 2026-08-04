package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Database_url         string
	App_key              string
	Port                 string
	App_env              string
	App_version          string
	Allowed_origins      []string
	Rate_limit_rps       int
	Delta_lt             float64
	Google_client_id     string
	Google_client_secret string
	Google_redirect_url  string
	App_url              string
	SMTP_host            string
	SMTP_port            int
	SMTP_user            string
	SMTP_password        string
	SMTP_from            string
	SMTP_tls             bool
	Imagekit_private_key string
	Imagekit_public_key  string
	Imagekit_url_endpoint string
	Audit_hmac_secret    string
	Audit_s3_endpoint    string
	Audit_s3_bucket      string
	Audit_s3_region      string
	Audit_s3_access_key  string
	Audit_s3_secret_key  string
	Audit_archive_dir    string
}

func Load() Config {
	return Config{
		Database_url:          envOr("DATABASE_URL", ""),
		App_key:               envOr("APP_KEY", ""),
		Port:                  envOr("PORT", "8080"),
		App_env:               envOr("APP_ENV", "development"),
		App_version:           envOr("APP_VERSION", "0.1.0"),
		Allowed_origins:       parseOrigins(envOr("ALLOWED_ORIGINS", "")),
		Rate_limit_rps:        envInt("RATE_LIMIT_RPS", 100),
		Delta_lt:              envFloat("DELTA_LT", 0.97),
		Google_client_id:      envOr("GOOGLE_CLIENT_ID", ""),
		Google_client_secret:  envOr("GOOGLE_CLIENT_SECRET", ""),
		Google_redirect_url:   envOr("GOOGLE_REDIRECT_URL", "http://localhost:8080/v1/auth/google/callback"),
		App_url:               envOr("APP_URL", "http://localhost:3000"),
		SMTP_host:             envOr("SMTP_HOST", ""),
		SMTP_port:             envInt("SMTP_PORT", 1025),
		SMTP_user:             envOr("SMTP_USER", ""),
		SMTP_password:         envOr("SMTP_PASSWORD", ""),
		SMTP_from:             envOr("SMTP_FROM", "Scaloo <noreply@scaloo.local>"),
		SMTP_tls:              envBool("SMTP_TLS", false),
		Imagekit_private_key:  envOr("IMAGEKIT_PRIVATE_KEY", ""),
		Imagekit_public_key:   envOr("IMAGEKIT_PUBLIC_KEY", ""),
		Imagekit_url_endpoint: envOr("IMAGEKIT_URL_ENDPOINT", ""),
		Audit_hmac_secret:     envOr("AUDIT_HMAC_SECRET", envOr("APP_KEY", "")),
		Audit_s3_endpoint:     envOr("AUDIT_S3_ENDPOINT", ""),
		Audit_s3_bucket:       envOr("AUDIT_S3_BUCKET", ""),
		Audit_s3_region:       envOr("AUDIT_S3_REGION", "auto"),
		Audit_s3_access_key:   envOr("AUDIT_S3_ACCESS_KEY", ""),
		Audit_s3_secret_key:   envOr("AUDIT_S3_SECRET_KEY", ""),
		Audit_archive_dir:     envOr("AUDIT_ARCHIVE_DIR", "./var/audit"),
	}
}

func (c Config) IsProduction() bool {
	return c.App_env == "production"
}

// parseOrigins splits a comma-separated ALLOWED_ORIGINS value, trimming
// whitespace and dropping empty entries.
func parseOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if o := strings.TrimSpace(p); o != "" {
			out = append(out, o)
		}
	}
	return out
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}
