package awg

import (
	"encoding/base64"
	"testing"
)

func TestGenerateKeypair(t *testing.T) {
	kp, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair: %v", err)
	}
	for name, key := range map[string]string{"private": kp.PrivateKey, "public": kp.PublicKey} {
		raw, err := base64.StdEncoding.DecodeString(key)
		if err != nil {
			t.Fatalf("%s key is not base64: %v", name, err)
		}
		if len(raw) != 32 {
			t.Errorf("%s key is %d bytes, want 32", name, len(raw))
		}
	}
	if kp.PrivateKey == kp.PublicKey {
		t.Error("private and public key must differ")
	}
}

func TestGenerateKeypairUnique(t *testing.T) {
	first, _ := GenerateKeypair()
	second, _ := GenerateKeypair()
	if first.PrivateKey == second.PrivateKey {
		t.Error("two generated private keys collided")
	}
}
