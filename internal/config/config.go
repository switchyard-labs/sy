// Package config manages sy's configuration: hosts (Switchyard installations),
// the active host, and per-host session tokens. Config follows XDG on Linux.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Host struct {
	Token string `json:"token"`
	User  string `json:"user,omitempty"`
}

type Config struct {
	Hosts      map[string]Host `json:"hosts"`
	ActiveHost string          `json:"active_host"`
	path       string
}

// Load reads the config from the platform config dir, returning an empty
// config if none exists yet.
func Load() (*Config, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	c := &Config{Hosts: map[string]Host{}, path: p}
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(b, c); err != nil {
		return nil, fmt.Errorf("config %s: %w", p, err)
	}
	if c.Hosts == nil {
		c.Hosts = map[string]Host{}
	}
	c.path = p
	return c, nil
}

// Path returns the config file path (XDG_CONFIG_HOME or ~/.config).
func Path() (string, error) {
	// SY_CONFIG_DIR overrides the whole config directory (useful for testing,
	// CI and self-hosted setups).
	if dir := os.Getenv("SY_CONFIG_DIR"); dir != "" {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return "", err
		}
		return filepath.Join(dir, "config.json"), nil
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	dir := filepath.Join(base, "switchyard")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Save writes the config with restrictive permissions.
func (c *Config) Save() error {
	if c.path == "" {
		p, err := Path()
		if err != nil {
			return err
		}
		c.path = p
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(c.path), ".config-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), c.path)
}

// Host returns the active host config, or nil.
func (c *Config) Host() (*Host, string) {
	if c.ActiveHost == "" {
		return nil, ""
	}
	h, ok := c.Hosts[c.ActiveHost]
	if !ok {
		return nil, c.ActiveHost
	}
	return &h, c.ActiveHost
}

// SetActive records the active host and returns the previous one (if any).
func (c *Config) SetActive(host string) string {
	prev := c.ActiveHost
	c.ActiveHost = host
	return prev
}

// SetHost stores (or updates) a host profile.
func (c *Config) SetHost(host string, h Host) {
	c.Hosts[host] = h
}

// HostList returns sorted host keys.
func (c *Config) HostList() []string {
	keys := make([]string, 0, len(c.Hosts))
	for k := range c.Hosts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// NormalizeHost preserves the transport scheme so HTTPS is never downgraded.
func NormalizeHost(u string) string {
	u = strings.TrimSpace(u)
	u = strings.TrimSuffix(u, "/")
	return u
}
