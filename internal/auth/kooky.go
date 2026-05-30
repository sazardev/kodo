package auth

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/browserutils/kooky"
	_ "github.com/browserutils/kooky/browser/all"
)

var ErrCookieNotFound = errors.New("opencode.ai session cookie not found")

type KookyAuth struct {
	browser string
}

func (k *KookyAuth) Authenticate() (string, error) {
	devNull, err := os.Open(os.DevNull)
	if err == nil {
		oldStderr := os.Stderr
		os.Stderr = devNull
		defer func() {
			os.Stderr = oldStderr
			devNull.Close()
		}()
	}

	browsers := []string{"chrome", "chromium", "firefox"}
	if k.browser != "" && k.browser != "all" {
		browsers = []string{k.browser}
	}

	for _, browser := range browsers {
		if cookie := k.findInBrowser(browser); cookie != "" {
			return cookie, nil
		}
	}

	return "", ErrCookieNotFound
}

func (k *KookyAuth) findInBrowser(browser string) string {
	ctx := context.Background()

	var cookies []*kooky.Cookie
	var err error

	switch browser {
	case "chrome":
		cookies, err = k.readChromeCookies(ctx)
	case "chromium":
		cookies, err = k.readChromiumCookies(ctx)
	case "firefox":
		cookies, err = k.readFirefoxCookies(ctx)
	default:
		cookies, err = kooky.ReadCookies(ctx, kooky.FilterFunc(func(c *kooky.Cookie) bool {
			return c.Browser != nil && c.Browser.Browser() == browser
		}))
	}

	if err != nil {
		return ""
	}

	for _, c := range cookies {
		if c.Domain == "opencode.ai" && c.Name == "__session" {
			return c.Value
		}
	}

	return ""
}

func (k *KookyAuth) readChromeCookies(ctx context.Context) ([]*kooky.Cookie, error) {
	paths := []string{
		os.ExpandEnv("$HOME/.config/google-chrome/Default/Network/Cookies"),
		os.ExpandEnv("$HOME/.config/chromium/Default/Cookies"),
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return kooky.ReadCookies(ctx, kooky.FilterFunc(func(c *kooky.Cookie) bool {
				return c.Browser != nil && c.Browser.Browser() == "chrome"
			}))
		}
	}

	return nil, errors.New("chrome cookies not found")
}

func (k *KookyAuth) readChromiumCookies(ctx context.Context) ([]*kooky.Cookie, error) {
	path := os.ExpandEnv("$HOME/.config/chromium/Default/Cookies")
	if _, err := os.Stat(path); err == nil {
		return kooky.ReadCookies(ctx, kooky.FilterFunc(func(c *kooky.Cookie) bool {
			return c.Browser != nil && c.Browser.Browser() == "chromium"
		}))
	}
	return nil, errors.New("chromium cookies not found")
}

func (k *KookyAuth) readFirefoxCookies(ctx context.Context) ([]*kooky.Cookie, error) {
	paths := []string{
		os.ExpandEnv("$HOME/.mozilla/firefox"),
		os.ExpandEnv("$HOME/.cache/mozilla/firefox"),
	}

	for _, basePath := range paths {
		if info, err := os.Stat(basePath); err == nil && info.IsDir() {
			return kooky.ReadCookies(ctx, kooky.FilterFunc(func(c *kooky.Cookie) bool {
				return c.Browser != nil && c.Browser.Browser() == "firefox"
			}))
		}
	}

	return nil, errors.New("firefox cookies not found")
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
