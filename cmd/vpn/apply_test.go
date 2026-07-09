package main

import "testing"

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
	idByTag := map[string]int{"VLESS_REALITY": 3, "HYSTERIA2": 1, "TROJAN": 2}

	all, err := resolveInbounds(ServiceSpec{Name: "all", Inbounds: []string{"*"}}, idByTag)
	if err != nil {
		t.Fatalf("wildcard: unexpected error: %v", err)
	}
	if !sameInts(all, []int{1, 2, 3}) {
		t.Errorf("wildcard: got %v want sorted [1 2 3]", all)
	}

	named, err := resolveInbounds(ServiceSpec{Name: "s", Inbounds: []string{"TROJAN", "VLESS_REALITY"}}, idByTag)
	if err != nil {
		t.Fatalf("named: unexpected error: %v", err)
	}
	if !sameInts(named, []int{2, 3}) {
		t.Errorf("named: got %v want [2 3]", named)
	}

	if _, err := resolveInbounds(ServiceSpec{Name: "s", Inbounds: []string{"NOPE"}}, idByTag); err == nil {
		t.Error("unknown tag: expected an error, got nil")
	}
}
