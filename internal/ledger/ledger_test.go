package ledger

import (
	"os"
	"path/filepath"
	"testing"
)

// breakSaves makes the next save() fail by removing the ledger's directory; restore
// re-creates it so a later save() succeeds again.
func breakSaves(t *testing.T, led *Ledger) {
	t.Helper()
	if err := os.RemoveAll(filepath.Dir(led.path)); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}
}

func restoreSaves(t *testing.T, led *Ledger) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(led.path), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
}

func TestClaimRollsBackOnSaveFailure(t *testing.T) {
	led := openTemp(t)
	if err := led.Add("alice", "tok-a"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	breakSaves(t, led)
	if _, _, err := led.Claim("tok-a", 42); err == nil {
		t.Fatal("Claim: expected save error, got nil")
	}
	// The claim must remain Pending in memory so a retry can still deliver.
	entry, ok := led.ByUsername("alice")
	if !ok {
		t.Fatal("entry vanished after failed Claim")
	}
	if entry.Status != Pending || entry.ChatID != 0 {
		t.Fatalf("after failed Claim: status=%q chat=%d, want pending/0", entry.Status, entry.ChatID)
	}
	restoreSaves(t, led)
	got, first, err := led.Claim("tok-a", 42)
	if err != nil || !first {
		t.Fatalf("retry Claim = (%+v, first=%v, %v), want first=true nil", got, first, err)
	}
}

func TestAddRollsBackOnSaveFailure(t *testing.T) {
	led := openTemp(t)
	if err := led.Add("alice", "tok-a"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	breakSaves(t, led)
	if err := led.Add("bob", "tok-b"); err == nil {
		t.Fatal("Add: expected save error, got nil")
	}
	if _, ok := led.ByUsername("bob"); ok {
		t.Fatal("bob still tracked after failed Add (not rolled back)")
	}
	if _, ok := led.ByUsername("alice"); !ok {
		t.Fatal("alice lost after failed Add of bob")
	}
	// The rolled-back append must not leave a lingering 'already tracked' state.
	restoreSaves(t, led)
	if err := led.Add("bob", "tok-b"); err != nil {
		t.Fatalf("retry Add bob: %v", err)
	}
}

func TestSaveAWGPeerRollsBackOnSaveFailure(t *testing.T) {
	led := openTemp(t)
	if err := led.SaveAWGPeer(AWGPeer{Username: "alice", Location: "Estonia", Address: "10.8.0.2"}); err != nil {
		t.Fatalf("SaveAWGPeer: %v", err)
	}
	breakSaves(t, led)
	if err := led.SaveAWGPeer(AWGPeer{Username: "alice", Location: "Russia", Address: "10.8.0.3"}); err == nil {
		t.Fatal("SaveAWGPeer: expected save error, got nil")
	}
	if _, ok := led.AWGPeer("alice", "Russia"); ok {
		t.Fatal("Russia peer present after failed append (not rolled back)")
	}
	if _, ok := led.AWGPeer("alice", "Estonia"); !ok {
		t.Fatal("Estonia peer lost after failed append of Russia")
	}
}

func TestAWGPeersForAndDelete(t *testing.T) {
	led := openTemp(t)
	peers := []AWGPeer{
		{Username: "alice", Location: "Estonia", Address: "10.8.0.2"},
		{Username: "alice", Location: "Russia", Address: "10.8.0.3"},
		{Username: "bob", Location: "Estonia", Address: "10.8.0.4"},
	}
	for _, peer := range peers {
		if err := led.SaveAWGPeer(peer); err != nil {
			t.Fatalf("SaveAWGPeer: %v", err)
		}
	}
	if got := led.AWGPeersFor("alice"); len(got) != 2 {
		t.Fatalf("AWGPeersFor(alice) = %d peers, want 2", len(got))
	}
	if got := led.AWGPeersFor("carol"); len(got) != 0 {
		t.Fatalf("AWGPeersFor(carol) = %d peers, want 0", len(got))
	}
	removed, err := led.DeleteAWGPeer("alice", "Russia")
	if err != nil || !removed {
		t.Fatalf("DeleteAWGPeer = (%v, %v), want (true, nil)", removed, err)
	}
	if _, ok := led.AWGPeer("alice", "Russia"); ok {
		t.Fatal("Russia peer still present after delete")
	}
	if _, ok := led.AWGPeer("alice", "Estonia"); !ok {
		t.Fatal("Estonia peer removed by unrelated delete")
	}
	if _, ok := led.AWGPeer("bob", "Estonia"); !ok {
		t.Fatal("bob's peer removed by alice's delete")
	}
	if removed, _ := led.DeleteAWGPeer("alice", "Russia"); removed {
		t.Fatal("DeleteAWGPeer reported removing an absent peer")
	}
}

func TestRemoveRollsBackOnSaveFailure(t *testing.T) {
	led := openTemp(t)
	for _, u := range [][2]string{{"a", "t-a"}, {"b", "t-b"}, {"c", "t-c"}} {
		if err := led.Add(u[0], u[1]); err != nil {
			t.Fatalf("Add %s: %v", u[0], err)
		}
	}
	breakSaves(t, led)
	if _, err := led.Remove("b"); err == nil {
		t.Fatal("Remove: expected save error, got nil")
	}
	got := led.List()
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("entries = %d after failed Remove, want %d", len(got), len(want))
	}
	for i, e := range got {
		if e.Username != want[i] {
			t.Fatalf("entry %d = %q, want %q (order not restored)", i, e.Username, want[i])
		}
	}
}

func openTemp(t *testing.T) *Ledger {
	t.Helper()
	led, err := Open(filepath.Join(t.TempDir(), "state", "ledger.json"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return led
}

func TestClaimIsExactlyOnce(t *testing.T) {
	led := openTemp(t)
	if err := led.Add("alice", "tok-a"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	entry, first, err := led.Claim("tok-a", 111)
	if err != nil || !first {
		t.Fatalf("first claim: entry=%+v first=%v err=%v", entry, first, err)
	}
	if entry.Username != "alice" || entry.ChatID != 111 || entry.Status != Delivered {
		t.Fatalf("first claim bound wrong: %+v", entry)
	}

	// A second claim of the same token must not re-deliver.
	_, first, err = led.Claim("tok-a", 111)
	if err != nil {
		t.Fatalf("second claim err: %v", err)
	}
	if first {
		t.Fatal("second claim reported a first delivery - exactly-once violated")
	}
}

func TestClaimUnknownToken(t *testing.T) {
	led := openTemp(t)
	if _, _, err := led.Claim("nope", 1); err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestAddRejectsDuplicates(t *testing.T) {
	led := openTemp(t)
	if err := led.Add("bob", "tok-b"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := led.Add("bob", "tok-c"); err == nil {
		t.Error("duplicate username accepted")
	}
	if err := led.Add("carol", "tok-b"); err == nil {
		t.Error("duplicate token accepted")
	}
}

func TestByChatAfterClaim(t *testing.T) {
	led := openTemp(t)
	_ = led.Add("dave", "tok-d")
	if _, ok := led.ByChat(222); ok {
		t.Fatal("unbound chat matched")
	}
	if _, _, err := led.Claim("tok-d", 222); err != nil {
		t.Fatalf("Claim: %v", err)
	}
	entry, ok := led.ByChat(222)
	if !ok || entry.Username != "dave" {
		t.Fatalf("ByChat after claim: entry=%+v ok=%v", entry, ok)
	}
}

func TestPersistenceAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ledger.json")

	led, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	_ = led.Add("erin", "tok-e")
	if _, _, err := led.Claim("tok-e", 333); err != nil {
		t.Fatalf("Claim: %v", err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	entry, ok := reopened.ByChat(333)
	if !ok || entry.Username != "erin" || entry.Status != Delivered {
		t.Fatalf("state not persisted: entry=%+v ok=%v", entry, ok)
	}
}

func TestRemove(t *testing.T) {
	led := openTemp(t)
	_ = led.Add("frank", "tok-f")
	removed, err := led.Remove("frank")
	if err != nil || !removed {
		t.Fatalf("Remove: removed=%v err=%v", removed, err)
	}
	if _, err := led.Remove("frank"); err != nil {
		t.Fatalf("second Remove err: %v", err)
	}
	if len(led.List()) != 0 {
		t.Fatalf("entries remain: %+v", led.List())
	}
}
