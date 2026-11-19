package auth

import "time"

const (
	defaultCookieName = "access_token"
	defaultTokenTTL   = 8 * time.Hour
)

// Config is the explicit authentication configuration. Values come from
// `auth.*` in the bootstrap config; there is deliberately no environment
// fallback here. A missing secret must fail configuration validation rather
// than silently signing tokens with a predictable value.
type Config struct {
	Secret       string
	CookieName   string
	CookieDomain string
	CookieSecure bool
	TokenTTL     time.Duration
}

func (c Config) cookieName() string {
	if c.CookieName != "" {
		return c.CookieName
	}
	return defaultCookieName
}

func (c Config) tokenTTL() time.Duration {
	if c.TokenTTL > 0 {
		return c.TokenTTL
	}
	return defaultTokenTTL
}

// TokenLifetime returns the configured token TTL, falling back to the package
// default when the config left it unset.
func (c Config) TokenLifetime() time.Duration {
	return c.tokenTTL()
}
