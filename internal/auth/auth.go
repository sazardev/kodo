package auth

import (
	"kodo/internal/config"
)

type Authenticator interface {
	Authenticate() (string, error)
	GetCookie() (string, error)
}

func NewAuthenticator(cfg *config.Config) Authenticator {
	if cfg.Auth.Mode == "manual" {
		return &ManualAuth{
			cookie: cfg.Session.Cookie,
		}
	}
	return &KookyAuth{
		browser: cfg.Auth.Browser,
	}
}
