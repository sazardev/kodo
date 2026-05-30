package auth

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
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

func PromptForCookie() (string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter your OpenCode cookie (__session token): ")
	cookie, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	cookie = strings.TrimSpace(cookie)
	if cookie == "" {
		cookie = os.Getenv("OPENCODE_COOKIE")
	}

	if cookie == "" {
		return "", ErrNoCookie
	}

	return cookie, nil
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
