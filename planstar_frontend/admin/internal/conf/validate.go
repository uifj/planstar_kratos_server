package conf

import (
	"errors"
	"fmt"
	"strings"
)

// Validate checks the invariants the service cannot start without. It runs
// after scanning, so environment overrides are already applied.
func Validate(c *Bootstrap) error {
	if c == nil {
		return errors.New("config: bootstrap is nil")
	}

	var errs []error
	prod := isProduction(c.GetServer().GetEnv())
	db := c.GetData().GetDatabase()

	if driver := strings.ToLower(strings.TrimSpace(db.GetDriver())); driver != "mysql" {
		errs = append(errs, fmt.Errorf("data.database.driver must be %q, got %q", "mysql", db.GetDriver()))
	}
	if strings.TrimSpace(db.GetSource()) == "" {
		errs = append(errs, errors.New("data.database.source is required"))
	}

	secret := strings.TrimSpace(c.GetAuth().GetJwtSecret())
	switch {
	case secret == "":
		errs = append(errs, errors.New("auth.jwt_secret is required"))
	case prod && len(secret) < 32:
		errs = append(errs, errors.New("auth.jwt_secret must be at least 32 characters when server.env is prod"))
	}

	if prod {
		if !c.GetAuth().GetCookieSecure() {
			errs = append(errs, errors.New("auth.cookie_secure must be true when server.env is prod"))
		}
		if db.GetAutoMigrate() {
			errs = append(errs, errors.New("data.database.auto_migrate must be false when server.env is prod"))
		}
		if strings.TrimSpace(c.GetPlanstar().GetOpsToken()) == "" {
			errs = append(errs, errors.New("planstar.ops_token is required when server.env is prod"))
		}
	}

	return errors.Join(errs...)
}

// isProduction accepts both prod and production so a deployment naming
// difference cannot silently disable the production-only checks.
func isProduction(env string) bool {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "prod", "production":
		return true
	default:
		return false
	}
}
