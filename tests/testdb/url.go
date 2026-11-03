// Package testdb prevents tests from falling back to application credentials.
package testdb

import (
	"errors"
	"net/url"
	"regexp"
)

// ParseLocalURL accepts only explicitly named disposable databases on loopback.
// It rejects alternate pgx connection targets supplied through query parameters.
func ParseLocalURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		return nil, errors.New("TEST_DATABASE_URL must be an explicit local PostgreSQL URL")
	}
	switch u.Hostname() {
	case "localhost", "127.0.0.1", "::1":
	default:
		return nil, errors.New("test database host must be loopback")
	}
	if !regexp.MustCompile(`^/splitter_repair[a-zA-Z0-9_]*$`).MatchString(u.Path) {
		return nil, errors.New("test database name must begin with splitter_repair")
	}
	for key := range u.Query() {
		if key != "sslmode" && key != "connect_timeout" {
			return nil, errors.New("unsupported test database connection parameter")
		}
	}
	if u.Fragment != "" {
		return nil, errors.New("test database URL must not have a fragment")
	}
	return u, nil
}
