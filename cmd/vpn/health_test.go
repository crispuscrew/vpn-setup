package main

import (
	"strings"
	"testing"

	"github.com/crispuscrew/vpn-setup/internal/panel"
)

func TestHealthReport(t *testing.T) {
	nodes := []panel.Node{
		{Name: "local", Status: panel.NodeHealthy, Backends: []panel.Backend{{Name: "xray", Version: "25.2.21"}}},
		{Name: "de", Status: "unhealthy"},
	}
	report, unhealthy := healthReport(nodes)
	if unhealthy != 1 {
		t.Fatalf("unhealthy count = %d, want 1", unhealthy)
	}
	if !strings.Contains(report, "1 of 2 node(s) healthy") {
		t.Errorf("summary line missing:\n%s", report)
	}
	if !strings.Contains(report, "DOWN") || !strings.Contains(report, "xray/25.2.21") {
		t.Errorf("report body wrong:\n%s", report)
	}
}

func TestHealthReportNoNodes(t *testing.T) {
	report, unhealthy := healthReport(nil)
	if unhealthy != 0 {
		t.Fatalf("unhealthy = %d, want 0", unhealthy)
	}
	if !strings.Contains(report, "none registered") {
		t.Errorf("want 'none registered':\n%s", report)
	}
}
