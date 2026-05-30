package auth

import (
	"errors"
	"os"
)

var ErrNoCookie = errors.New("no cookie found")

type ManualAuth struct {
	cookie string
}

func (m *ManualAuth) Authenticate() (string, error) {
	if m.cookie != "" {
		return m.cookie, nil
	}

	cookie := os.Getenv("OPENCODE_COOKIE")
	if cookie != "" {
		return cookie, nil
	}

	return "", ErrNoCookie
}

func (m *ManualAuth) GetCookie() (string, error) {
	return m.Authenticate()
}
