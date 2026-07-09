package ledger

import (
	"path/filepath"
	"testing"
)

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
		t.Fatal("second claim reported a first delivery — exactly-once violated")
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
