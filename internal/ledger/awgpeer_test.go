package ledger

import (
	"path/filepath"
	"testing"
)

func TestAWGPeerSaveAndGet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	led, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, ok := led.AWGPeer("alice", "Estonia"); ok {
		t.Fatal("expected no peer before save")
	}
	peer := AWGPeer{Username: "alice", Location: "Estonia", PublicKey: "pub", PrivateKey: "priv", Address: "10.8.0.5"}
	if err := led.SaveAWGPeer(peer); err != nil {
		t.Fatalf("SaveAWGPeer: %v", err)
	}
	got, ok := led.AWGPeer("alice", "Estonia")
	if !ok || got != peer {
		t.Fatalf("AWGPeer = (%+v, %v), want %+v", got, ok, peer)
	}
}

func TestAWGPeerUpsertsByUserAndLocation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	led, _ := Open(path)
	_ = led.SaveAWGPeer(AWGPeer{Username: "alice", Location: "Estonia", Address: "10.8.0.5"})
	_ = led.SaveAWGPeer(AWGPeer{Username: "alice", Location: "Estonia", Address: "10.8.0.9"})
	_ = led.SaveAWGPeer(AWGPeer{Username: "alice", Location: "USA", Address: "10.8.0.2"})

	if got, _ := led.AWGPeer("alice", "Estonia"); got.Address != "10.8.0.9" {
		t.Errorf("Estonia address = %q, want 10.8.0.9 (upsert)", got.Address)
	}
	if got, _ := led.AWGPeer("alice", "USA"); got.Address != "10.8.0.2" {
		t.Errorf("USA address = %q, want 10.8.0.2", got.Address)
	}
}

func TestAWGPeerPersistsAcrossReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	led, _ := Open(path)
	_ = led.SaveAWGPeer(AWGPeer{Username: "bob", Location: "Serbia", PrivateKey: "priv", Address: "10.8.0.3"})

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	got, ok := reopened.AWGPeer("bob", "Serbia")
	if !ok || got.PrivateKey != "priv" || got.Address != "10.8.0.3" {
		t.Errorf("after reopen AWGPeer = (%+v, %v)", got, ok)
	}
}
