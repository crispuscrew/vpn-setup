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

func TestRemoteCommandJoinsSafeArgs(t *testing.T) {
	got, err := remoteCommand("/usr/local/sbin/awg-peer", []string{"add-peer", "alice-USA", "abc+def/ghi="})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "/usr/local/sbin/awg-peer add-peer alice-USA abc+def/ghi="
	if got != want {
		t.Errorf("remoteCommand = %q, want %q", got, want)
	}
}

func TestRemoteCommandRejectsUnsafeArgs(t *testing.T) {
	for _, bad := range []string{"a b", "x;rm -rf /", "it's", "$(whoami)", "a`b`", "a|b"} {
		if _, err := remoteCommand("/usr/local/sbin/awg-peer", []string{"add-peer", bad, "key"}); err == nil {
			t.Errorf("remoteCommand should reject %q", bad)
		}
	}
}
