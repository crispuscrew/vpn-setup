// Package awg provisions AmneziaWG peers. It generates WireGuard-format key pairs,
// renders a client config from a node's server profile, and drives the node-side
// awg-peer agent over SSH. AmneziaWG cannot ride the Marzneshin subscription, so a
// peer is delivered out of band as a .conf file the AmneziaVPN app imports.
package awg

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
)

// Keypair is a base64-encoded X25519 key pair in the format WireGuard and
// AmneziaWG use for PrivateKey/PublicKey fields.
type Keypair struct {
	PrivateKey string
	PublicKey  string
}

// GenerateKeypair returns a fresh X25519 key pair. GenerateKey clamps the private
// scalar exactly as WireGuard requires, so the keys are drop-in for an awg config.
func GenerateKeypair() (Keypair, error) {
	priv, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return Keypair{}, err
	}
	return Keypair{
		PrivateKey: base64.StdEncoding.EncodeToString(priv.Bytes()),
		PublicKey:  base64.StdEncoding.EncodeToString(priv.PublicKey().Bytes()),
	}, nil
}
