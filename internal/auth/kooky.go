package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/browserutils/kooky"
	_ "github.com/browserutils/kooky/browser/all"
)

var ErrCookieNotFound = errors.New("opencode.ai session cookie not found")

type KookyAuth struct {
	browser string
}

func (k *KookyAuth) Authenticate() (string, error) {
	ctx := context.Background()

	var cookies []*kooky.Cookie
	var err error

	if k.browser != "" && k.browser != "all" {
		cookies, err = kooky.ReadCookies(ctx, kooky.FilterFunc(func(c *kooky.Cookie) bool {
			return c.Browser != nil && c.Browser.Browser() == k.browser
		}))
	} else {
		cookies, err = kooky.ReadCookies(ctx)
	}

	if err != nil {
		return "", err
	}

	for _, c := range cookies {
		if c.Domain == "opencode.ai" && c.Name == "__session" {
			return c.Value, nil
		}
	}

	return "", ErrCookieNotFound
}

func (k *KookyAuth) GetCookie() (string, error) {
	return k.Authenticate()
}

func ExtractFromHeader(header string) string {
	parts := strings.Split(header, ";")
	for _, part := range parts {
		if strings.Contains(part, "__session=") {
			return strings.TrimSpace(strings.Split(part, "__session=")[1])
		}
	}
	return ""
}
