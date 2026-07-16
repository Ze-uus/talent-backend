package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Database_url          string
	App_key               string
	Port                  string
	App_env               string
	App_version           string
	Allowed_origins       []string
	Rate_limit_rps        int
	Delta_lt              float64
	Google_client_id      string
	Google_client_secret  string
	Google_redirect_url   string
}

func Load() Config {
	return Config{
		Database_url:         envOr("DATABASE_URL", ""),
		App_key:              envOr("APP_KEY", ""),
		Port:                 envOr("PORT", "8080"),
		App_env:              envOr("APP_ENV", "development"),
		App_version:          envOr("APP_VERSION", "0.1.0"),
		Allowed_origins:      parseOrigins(envOr("ALLOWED_ORIGINS", "")),
		Rate_limit_rps:       envInt("RATE_LIMIT_RPS", 100),
		Delta_lt:             envFloat("DELTA_LT", 0.97),
		Google_client_id:     envOr("GOOGLE_CLIENT_ID", ""),
		Google_client_secret: envOr("GOOGLE_CLIENT_SECRET", ""),
		Google_redirect_url:  envOr("GOOGLE_REDIRECT_URL", "http://localhost:8080/v1/auth/google/callback"),
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
