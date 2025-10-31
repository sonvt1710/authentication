package config

import (
	"context"
	"os"
	"strings"
	"time"

	coreConfig "github.com/lee-tech/core/config"
	"github.com/lee-tech/core/secret"
)

// AuthConfig extends the core configuration with auth-specific settings
type AuthConfig struct {
	*coreConfig.Config

	// Auth specific settings
	TokenExpiration   time.Duration `env:"TOKEN_EXPIRATION" envDefault:"15m"`
	RefreshExpiration time.Duration `env:"REFRESH_EXPIRATION" envDefault:"7d"`
	PasswordMinLength int           `env:"PASSWORD_MIN_LENGTH" envDefault:"8"`
	MaxLoginAttempts  int           `env:"MAX_LOGIN_ATTEMPTS" envDefault:"5"`
	LockoutDuration   time.Duration `env:"LOCKOUT_DURATION" envDefault:"15m"`
	BCryptCost        int           `env:"BCRYPT_COST" envDefault:"10"`

	// OAuth settings (optional)
	OAuthEnabled       bool   `env:"OAUTH_ENABLED" envDefault:"false"`
	GoogleClientID     string `env:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret string `env:"GOOGLE_CLIENT_SECRET"`

	// MFA settings
	MFAEnabled bool   `env:"MFA_ENABLED" envDefault:"false"`
	TOTPIssuer string `env:"TOTP_ISSUER" envDefault:"Lee-Tech"`

	// Bootstrap settings
	BootstrapOrganizationName        string
	BootstrapOrganizationDescription string
	BootstrapOrganizationDomain      string
	BootstrapAdminEmail              string
	BootstrapAdminUsername           string
	BootstrapAdminPassword           string
	BootstrapAdminFirstName          string
	BootstrapAdminLastName           string
}

// Load loads the configuration from environment variables
func Load() (*AuthConfig, error) {
	// Load core configuration
	coreConfig, err := coreConfig.Load()
	if err != nil {
		return nil, err
	}

	// Create auth config with core config embedded
	authConfig := &AuthConfig{
		Config: coreConfig,
	}

	// Load secrets from Vault if configured
	if coreConfig.VaultAddr != "" && coreConfig.VaultToken != "" {
		provider, err := secret.NewVaultProvider(coreConfig.VaultAddr, coreConfig.VaultToken)
		if err == nil {
			ctx := context.Background()
			secrets, err := provider.GetSecrets(ctx, []string{
				"JWT_SECRET",
				"GOOGLE_CLIENT_SECRET",
			})
			if err == nil {
				if jwtSecret, ok := secrets["JWT_SECRET"]; ok {
					authConfig.JWTSecret = jwtSecret
				}
				if googleSecret, ok := secrets["GOOGLE_CLIENT_SECRET"]; ok {
					authConfig.GoogleClientSecret = googleSecret
				}
			}
		}
	}

	applyBootstrapDefaults(authConfig)

	return authConfig, nil
}

// NewWatcher creates a configuration watcher for the auth service
func NewWatcher(cfg *coreConfig.Config) (*coreConfig.Watcher, error) {
	return coreConfig.NewWatcher(cfg)
}

func applyBootstrapDefaults(cfg *AuthConfig) {
	if cfg == nil {
		return
	}

	cfg.BootstrapOrganizationName = getEnvDefault("BOOTSTRAP_ORG_NAME", "Root Organization")
	cfg.BootstrapOrganizationDescription = getEnvDefault("BOOTSTRAP_ORG_DESCRIPTION", "System root organization")
	cfg.BootstrapOrganizationDomain = getEnvDefault("BOOTSTRAP_ORG_DOMAIN", "root.local")
	cfg.BootstrapAdminEmail = getEnvDefault("BOOTSTRAP_ADMIN_EMAIL", "admin@root.local")
	cfg.BootstrapAdminUsername = getEnvDefault("BOOTSTRAP_ADMIN_USERNAME", "root-admin")
	cfg.BootstrapAdminPassword = getEnvDefault("BOOTSTRAP_ADMIN_PASSWORD", "ChangeMe123!")
	cfg.BootstrapAdminFirstName = getEnvDefault("BOOTSTRAP_ADMIN_FIRST_NAME", "System")
	cfg.BootstrapAdminLastName = getEnvDefault("BOOTSTRAP_ADMIN_LAST_NAME", "Administrator")
}

func getEnvDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
