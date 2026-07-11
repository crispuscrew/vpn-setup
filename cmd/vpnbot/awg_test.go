package main

import "testing"

func TestParseAWGNodes(t *testing.T) {
	nodes, err := parseAWGNodes("Estonia=150.0.0.1, Russia=31.0.0.2 , USA=8.8.8.8")
	if err != nil {
		t.Fatalf("parseAWGNodes: %v", err)
	}
	want := map[string]string{"Estonia": "150.0.0.1", "Russia": "31.0.0.2", "USA": "8.8.8.8"}
	if len(nodes) != len(want) {
		t.Fatalf("got %d nodes, want %d", len(nodes), len(want))
	}
	for name, host := range want {
		if nodes[name] != host {
			t.Errorf("nodes[%q] = %q, want %q", name, nodes[name], host)
		}
	}
}

func TestParseAWGNodesEmpty(t *testing.T) {
	nodes, err := parseAWGNodes("")
	if err != nil {
		t.Fatalf("parseAWGNodes empty: %v", err)
	}
	if len(nodes) != 0 {
		t.Errorf("empty input should yield no nodes, got %d", len(nodes))
	}
}

func TestParseAWGNodesMalformed(t *testing.T) {
	for _, in := range []string{"Estonia", "Estonia=", "=1.2.3.4"} {
		if _, err := parseAWGNodes(in); err == nil {
			t.Errorf("parseAWGNodes(%q) should have failed", in)
		}
	}
}

func TestParseUserLocation(t *testing.T) {
	user, loc, ok := parseUserLocation("alice|Estonia")
	if !ok || user != "alice" || loc != "Estonia" {
		t.Errorf("parseUserLocation = (%q, %q, %v)", user, loc, ok)
	}
	for _, bad := range []string{"", "alice", "|Estonia", "alice|"} {
		if _, _, ok := parseUserLocation(bad); ok {
			t.Errorf("parseUserLocation(%q) should be invalid", bad)
		}
	}
}
