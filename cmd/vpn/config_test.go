package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTemp(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "vpn.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}

func TestLoadConfigValid(t *testing.T) {
	path := writeTemp(t, `
services:
  - name: all
    inbounds: ["*"]
users:
  - username: testuser
    services: ["all"]
    expire_strategy: never
`)
	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if len(cfg.Services) != 1 || cfg.Services[0].Name != "all" {
		t.Fatalf("services parsed wrong: %+v", cfg.Services)
	}
	if len(cfg.Users) != 1 || cfg.Users[0].Username != "testuser" {
		t.Fatalf("users parsed wrong: %+v", cfg.Users)
	}
}

func TestLoadConfigRejectsEmptyService(t *testing.T) {
	path := writeTemp(t, `
services:
  - name: broken
    inbounds: []
`)
	if _, err := loadConfig(path); err == nil {
		t.Error("expected an error for a service with no inbounds, got nil")
	}
}

func TestLoadConfigRejectsUserWithoutServices(t *testing.T) {
	path := writeTemp(t, `
users:
  - username: orphan
    services: []
`)
	if _, err := loadConfig(path); err == nil {
		t.Error("expected an error for a user with no services, got nil")
	}
}

func TestLoadConfigNodesService(t *testing.T) {
	path := writeTemp(t, `
services:
  - name: Estonia
    nodes: ["Estonia"]
`)
	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if len(cfg.Services) != 1 || len(cfg.Services[0].Nodes) != 1 || cfg.Services[0].Nodes[0] != "Estonia" {
		t.Fatalf("nodes service parsed wrong: %+v", cfg.Services)
	}
}

func TestLoadConfigRejectsBothSelectors(t *testing.T) {
	path := writeTemp(t, `
services:
  - name: both
    inbounds: ["*"]
    nodes: ["Estonia"]
`)
	if _, err := loadConfig(path); err == nil {
		t.Error("expected an error for a service setting both inbounds and nodes, got nil")
	}
}
