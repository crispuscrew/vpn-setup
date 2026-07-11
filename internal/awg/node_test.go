package awg

import "testing"

func TestPeerName(t *testing.T) {
	cases := []struct {
		user, location, want string
	}{
		{"alice", "Estonia", "alice-Estonia"},
		{"bob smith", "Serbia", "bob-smith-Serbia"},
		{"weird/name", "USA", "weird-name-USA"},
		{"drop;table", "X", "drop-table-X"},
	}
	for _, tc := range cases {
		if got := PeerName(tc.user, tc.location); got != tc.want {
			t.Errorf("PeerName(%q, %q) = %q, want %q", tc.user, tc.location, got, tc.want)
		}
	}
}

func TestPeerNameTruncates(t *testing.T) {
	long := ""
	for i := 0; i < 100; i++ {
		long += "a"
	}
	if got := PeerName(long, "USA"); len(got) != 64 {
		t.Errorf("PeerName length = %d, want 64", len(got))
	}
}

func TestShellJoinQuotesUnsafe(t *testing.T) {
	cases := map[string]string{
		"plain":             "plain",
		"10.8.0.5/32":       "10.8.0.5/32",
		"a b":               "'a b'",
		"x;rm -rf /":        "'x;rm -rf /'",
		"it's":              `'it'\''s'`,
		"base64+key/value=": "base64+key/value=",
	}
	for in, want := range cases {
		if got := shellJoin([]string{in}); got != want {
			t.Errorf("shellJoin(%q) = %q, want %q", in, got, want)
		}
	}
}
