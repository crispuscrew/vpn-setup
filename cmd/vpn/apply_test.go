package main

import (
	"testing"

	"github.com/crispuscrew/vpn-setup/internal/panel"
)

func TestSameInts(t *testing.T) {
	cases := []struct {
		name string
		a, b []int
		want bool
	}{
		{"equal same order", []int{1, 2, 3}, []int{1, 2, 3}, true},
		{"equal reordered", []int{3, 1, 2}, []int{1, 2, 3}, true},
		{"different length", []int{1, 2}, []int{1, 2, 3}, false},
		{"different members", []int{1, 2, 4}, []int{1, 2, 3}, false},
		{"both empty", nil, []int{}, true},
	}
	for _, tc := range cases {
		if got := sameInts(tc.a, tc.b); got != tc.want {
			t.Errorf("%s: sameInts(%v,%v)=%v want %v", tc.name, tc.a, tc.b, got, tc.want)
		}
	}
}

func TestResolveInbounds(t *testing.T) {
	// Same tag "VLESS_REALITY" on two nodes (ids 3 and 4) must not collapse.
	inbounds := []panel.Inbound{
		{ID: 1, Tag: "HYSTERIA2"},
		{ID: 2, Tag: "TROJAN"},
		{ID: 3, Tag: "VLESS_REALITY"},
		{ID: 4, Tag: "VLESS_REALITY"},
	}

	all, err := resolveInbounds(ServiceSpec{Name: "all", Inbounds: []string{"*"}}, inbounds)
	if err != nil {
		t.Fatalf("wildcard: unexpected error: %v", err)
	}
	if !sameInts(all, []int{1, 2, 3, 4}) {
		t.Errorf("wildcard: got %v want [1 2 3 4] (must include both same-tag nodes)", all)
	}

	named, err := resolveInbounds(ServiceSpec{Name: "s", Inbounds: []string{"VLESS_REALITY"}}, inbounds)
	if err != nil {
		t.Fatalf("named: unexpected error: %v", err)
	}
	if !sameInts(named, []int{3, 4}) {
		t.Errorf("named: got %v want [3 4] (both nodes' VLESS_REALITY)", named)
	}

	if _, err := resolveInbounds(ServiceSpec{Name: "s", Inbounds: []string{"NOPE"}}, inbounds); err == nil {
		t.Error("unknown tag: expected an error, got nil")
	}
}

func TestResolveInboundsByNode(t *testing.T) {
	// Every node runs the same VLESS_REALITY tag; a per-location service must
	// select by node name, not tag.
	inbounds := []panel.Inbound{
		{ID: 1, Tag: "VLESS_REALITY", Node: panel.InboundNode{ID: 10, Name: "Estonia"}},
		{ID: 2, Tag: "VLESS_REALITY", Node: panel.InboundNode{ID: 20, Name: "Russia"}},
		{ID: 3, Tag: "VLESS_REALITY", Node: panel.InboundNode{ID: 30, Name: "Serbia"}},
	}

	ee, err := resolveInbounds(ServiceSpec{Name: "Estonia", Nodes: []string{"Estonia"}}, inbounds)
	if err != nil {
		t.Fatalf("one node: unexpected error: %v", err)
	}
	if !sameInts(ee, []int{1}) {
		t.Errorf("one node: got %v want [1]", ee)
	}

	multi, err := resolveInbounds(ServiceSpec{Name: "eu", Nodes: []string{"Estonia", "Serbia"}}, inbounds)
	if err != nil {
		t.Fatalf("two nodes: unexpected error: %v", err)
	}
	if !sameInts(multi, []int{1, 3}) {
		t.Errorf("two nodes: got %v want [1 3]", multi)
	}

	if _, err := resolveInbounds(ServiceSpec{Name: "x", Nodes: []string{"Mars"}}, inbounds); err == nil {
		t.Error("unknown node: expected an error, got nil")
	}
}
