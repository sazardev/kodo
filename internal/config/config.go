package config

import (
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"
)

type Config struct {
	Session SessionConfig `json:"session"`
	Auth   AuthConfig    `json:"auth"`
	Paths  PathsConfig   `json:"paths"`
}

type SessionConfig struct {
	Cookie    string `json:"cookie,omitempty"`
	Workspace string `json:"workspace,omitempty"`
}

type AuthConfig struct {
	Mode    string `json:"mode"`
	Browser string `json:"browser,omitempty"`
}

type PathsConfig struct {
	LocalDB     string `json:"local_db"`
	MessageLogs string `json:"message_logs"`
}

var DefaultConfig = Config{
	Auth: AuthConfig{
		Mode:    "auto",
		Browser: "all",
	},
	Paths: PathsConfig{
		LocalDB:     "~/.local/share/opencode/opencode.db",
		MessageLogs: "~/.local/share/opencode/",
	},
}

func ConfigPath() string {
	usr, _ := user.Current()
	return filepath.Join(usr.HomeDir, ".config", "octa", "config.json")
}

func DefaultConfigPath() string {
	return ConfigPath()
}

func Load() (*Config, error) {
	cfg := DefaultConfig
	path := ConfigPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &cfg, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Save() error {
	path := ConfigPath()
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return err
	}

	return nil
}
