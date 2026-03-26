package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime configuration for vynilino.
type Config struct {
	// Server
	ListenAddr  string
	Environment string // "development" or "production"

	// Database
	DBPath string // SQLite file path; "postgres://..." switches to PostgreSQL

	// Media storage
	MediaDir string

	// Auth
	TokenKey       string // PASETO symmetric key (hex-encoded 32 bytes)
	TokenKeyNew    string // Optional: new key during rotation bridge period (VYNILINO_TOKEN_KEY_NEW)
	SingleOwner    bool   // When true, only one user account is allowed
	BootstrapToken string // VYNILINO_BOOTSTRAP_TOKEN — when set, required for first-user registration

	// OIDC (optional — disabled when OIDCIssuer is empty)
	OIDCIssuer       string // e.g. https://accounts.google.com
	OIDCClientID     string
	OIDCClientSecret string
	OIDCRedirectURL  string // e.g. http://localhost:8080/oidc/callback
	OIDCAutoRedirect bool   // When true, GET /login redirects directly to OIDC provider

	// Discogs (optional — enables higher API rate limits when set)
	DiscogsToken string

	// GraphQL
	AllowedOrigins []string
	Playground     bool
	Introspection  bool

	// Proxy / TLS
	// BehindProxy controls whether X-Forwarded-For / X-Real-IP headers are trusted.
	// Only set this when the server is running behind a known reverse proxy.
	BehindProxy bool   // VYNILINO_BEHIND_PROXY; default false
	TLSCertFile string // VYNILINO_TLS_CERT — path to TLS certificate (PEM)
	TLSKeyFile  string // VYNILINO_TLS_KEY  — path to TLS private key (PEM)
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		ListenAddr:       getEnv("VYNILINO_LISTEN_ADDR", ":8080"),
		Environment:      getEnv("VYNILINO_ENV", "production"),
		DBPath:           getEnv("VYNILINO_DB_PATH", "./data/vynilino.db"),
		MediaDir:         getEnv("VYNILINO_MEDIA_DIR", "./data/media"),
		TokenKey:         getEnv("VYNILINO_TOKEN_KEY", ""),
		TokenKeyNew:      getEnv("VYNILINO_TOKEN_KEY_NEW", ""),
		SingleOwner:      getBoolEnv("VYNILINO_SINGLE_OWNER", true),
		BootstrapToken:   getEnv("VYNILINO_BOOTSTRAP_TOKEN", ""),
		OIDCIssuer:       getEnv("VYNILINO_OIDC_ISSUER", ""),
		OIDCClientID:     getEnv("VYNILINO_OIDC_CLIENT_ID", ""),
		OIDCClientSecret: getEnv("VYNILINO_OIDC_CLIENT_SECRET", ""),
		OIDCRedirectURL:  getEnv("VYNILINO_OIDC_REDIRECT_URL", ""),
		OIDCAutoRedirect: getBoolEnv("VYNILINO_OIDC_AUTO_REDIRECT", false),
		DiscogsToken:     getEnv("VYNILINO_DISCOGS_TOKEN", ""),
		AllowedOrigins: splitCSV(
			getEnv("VYNILINO_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173"),
		),
		Playground:    getBoolEnv("VYNILINO_PLAYGROUND", false),
		Introspection: getBoolEnv("VYNILINO_INTROSPECTION", false),
		BehindProxy:   getBoolEnv("VYNILINO_BEHIND_PROXY", false),
		TLSCertFile:   getEnv("VYNILINO_TLS_CERT", ""),
		TLSKeyFile:    getEnv("VYNILINO_TLS_KEY", ""),
	}

	// Override Playground / Introspection defaults in development
	if cfg.IsDevelopment() {
		if os.Getenv("VYNILINO_PLAYGROUND") == "" {
			cfg.Playground = true
		}
		if os.Getenv("VYNILINO_INTROSPECTION") == "" {
			cfg.Introspection = true
		}
	}

	if cfg.TokenKey == "" {
		return nil, fmt.Errorf("VYNILINO_TOKEN_KEY must be set")
	}

	return cfg, nil
}

// IsDevelopment reports whether the server is running in development mode.
func (c *Config) IsDevelopment() bool {
	return strings.EqualFold(c.Environment, "development")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getBoolEnv(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
