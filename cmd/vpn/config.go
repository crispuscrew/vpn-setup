package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the declared panel state applied by `vpn apply`. Protocols/inbounds
// themselves live in each node's core config (Ansible); this file only groups them
// into services and grants users.
type Config struct {
	Services []ServiceSpec `yaml:"services"`
	Users    []UserSpec    `yaml:"users"`
}

// ServiceSpec declares a service and the inbounds it groups. Set exactly one of:
// Inbounds (tags, or a single "*" for every discovered inbound) or Nodes (node
// names, grouping every inbound on those nodes - used for per-location access).
type ServiceSpec struct {
	Name     string   `yaml:"name"`
	Inbounds []string `yaml:"inbounds"`
	Nodes    []string `yaml:"nodes"`
}

// UserSpec declares a user and the services (by name) granted to them.
type UserSpec struct {
	Username       string   `yaml:"username"`
	Services       []string `yaml:"services"`
	ExpireStrategy string   `yaml:"expire_strategy"`
	Note           string   `yaml:"note"`
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &cfg, nil
}

func (c *Config) validate() error {
	for _, service := range c.Services {
		if service.Name == "" {
			return fmt.Errorf("a service entry is missing its name")
		}
		if (len(service.Inbounds) > 0) == (len(service.Nodes) > 0) {
			return fmt.Errorf("service %q must set exactly one of 'inbounds' or 'nodes'", service.Name)
		}
	}
	for _, user := range c.Users {
		if user.Username == "" {
			return fmt.Errorf("a user entry is missing its username")
		}
		if len(user.Services) == 0 {
			return fmt.Errorf("user %q is granted no services", user.Username)
		}
	}
	return nil
}
