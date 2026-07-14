package main

import (
	"testing"

	"github.com/crispuscrew/vpn-setup/internal/panel"
)

func TestAWGLocationsFrom(t *testing.T) {
	awgNodes := map[string]string{"Estonia": "h", "Serbia": "h", "USA": "h", "Russia": "h"}
	// service 1 = "all" (exit nodes only), 2 = Estonia, 4 = Serbia, 5 = USA, 3 = Russia.
	inbounds := []panel.Inbound{
		{ID: 1, ServiceIDs: []int{1, 2}, Node: panel.InboundNode{Name: "Estonia"}},
		{ID: 2, ServiceIDs: []int{1, 2}, Node: panel.InboundNode{Name: "Estonia"}}, // 2nd inbound, same node
		{ID: 3, ServiceIDs: []int{1, 4}, Node: panel.InboundNode{Name: "Serbia"}},
		{ID: 4, ServiceIDs: []int{1, 5}, Node: panel.InboundNode{Name: "USA"}},
		{ID: 5, ServiceIDs: []int{3}, Node: panel.InboundNode{Name: "Russia"}},
	}
	cases := []struct {
		name    string
		granted []int
		want    []string
	}{
		{"all expands to exit nodes, excludes Russia", []int{1}, []string{"Estonia", "Serbia", "USA"}},
		{"per-location Estonia dedups its two inbounds", []int{2}, []string{"Estonia"}},
		{"Russia service", []int{3}, []string{"Russia"}},
		{"all plus Russia", []int{1, 3}, []string{"Estonia", "Russia", "Serbia", "USA"}},
		{"no grants", nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := awgLocationsFrom(tc.granted, inbounds, awgNodes); !equalStrings(got, tc.want) {
				t.Errorf("awgLocationsFrom(%v) = %v, want %v", tc.granted, got, tc.want)
			}
		})
	}
	// A node with no AWG agent is excluded even when the user is granted it.
	noUSA := map[string]string{"Estonia": "h", "Serbia": "h", "Russia": "h"}
	if got := awgLocationsFrom([]int{1}, inbounds, noUSA); !equalStrings(got, []string{"Estonia", "Serbia"}) {
		t.Errorf("no-AWG-node: got %v want [Estonia Serbia]", got)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

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

func TestResolveAWGTarget(t *testing.T) {
	const bound = "crispuscrew"
	cases := []struct {
		name     string
		args     []string
		isAdmin  bool
		hasBound bool
		want     string
		wantOK   bool
	}{
		// A tapped button is a callback: Args() is [""]. It must resolve to the
		// caller's own account, not an empty username (the reported bug).
		{"admin taps button", []string{""}, true, true, bound, true},
		{"non-admin taps button", []string{""}, false, true, bound, true},
		{"admin bare command", nil, true, true, bound, true},
		{"admin targets a user", []string{"alice"}, true, true, "alice", true},
		{"admin arg is lowercased", []string{"Alice"}, true, true, "alice", true},
		{"non-admin arg ignored, uses bound", []string{"alice"}, false, true, bound, true},
		{"button but no bound account", []string{""}, true, false, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := resolveAWGTarget(tc.args, tc.isAdmin, bound, tc.hasBound)
			if got != tc.want || ok != tc.wantOK {
				t.Errorf("resolveAWGTarget(%q, admin=%v, bound=%v) = (%q, %v), want (%q, %v)",
					tc.args, tc.isAdmin, tc.hasBound, got, ok, tc.want, tc.wantOK)
			}
		})
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
