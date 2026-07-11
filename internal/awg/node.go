package awg

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// Default SSH wiring. The bot runs on the Ansible control box, so it reuses the
// same key and root login already used to deploy the nodes; when the bot moves to
// its own host this should become a forced-command-restricted key.
const (
	DefaultUser   = "root"
	DefaultScript = "/usr/local/sbin/awg-peer"
)

var (
	// peerNameSafe is the charset the node agent accepts for a peer name; the bot
	// sanitises to it so a username can never inject shell metacharacters.
	peerNameSafe = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)
	shellUnsafe  = regexp.MustCompile(`[^A-Za-z0-9_./:=@%+-]`)
)

// NodeAgent runs the node-side awg-peer script over SSH.
type NodeAgent struct {
	keyPath string
	user    string
	script  string
}

// NewNodeAgent returns an agent that logs in as user with keyPath and runs script
// on the target node. Empty user/script fall back to the defaults.
func NewNodeAgent(keyPath, user, script string) *NodeAgent {
	if user == "" {
		user = DefaultUser
	}
	if script == "" {
		script = DefaultScript
	}
	return &NodeAgent{keyPath: keyPath, user: user, script: script}
}

// PeerName builds a stable, node-safe peer name for a user at a location.
func PeerName(username, location string) string {
	name := peerNameSafe.ReplaceAllString(username+"-"+location, "-")
	name = strings.Trim(name, "-.")
	if len(name) > 64 {
		name = name[:64]
	}
	return name
}

// ServerProfile fetches the node's server identity and obfuscation profile.
func (n *NodeAgent) ServerProfile(ctx context.Context, host string) (ServerProfile, error) {
	out, err := n.run(ctx, host, "server-profile")
	if err != nil {
		return ServerProfile{}, err
	}
	var profile ServerProfile
	if err := json.Unmarshal(out, &profile); err != nil {
		return ServerProfile{}, fmt.Errorf("parse server profile: %w", err)
	}
	return profile, nil
}

// AddPeer provisions pubkey on the node (idempotent per key) and returns the host
// address the node assigned the peer.
func (n *NodeAgent) AddPeer(ctx context.Context, host, name, pubkey string) (string, error) {
	out, err := n.run(ctx, host, "add-peer", name, pubkey)
	if err != nil {
		return "", err
	}
	var res struct {
		Address string `json:"address"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		return "", fmt.Errorf("parse add-peer result: %w", err)
	}
	if res.Address == "" {
		return "", fmt.Errorf("node returned no address")
	}
	return res.Address, nil
}

// DelPeer removes a peer from the node by its public key.
func (n *NodeAgent) DelPeer(ctx context.Context, host, pubkey string) error {
	_, err := n.run(ctx, host, "del-peer", pubkey)
	return err
}

// run executes the awg-peer script with args on host over SSH and returns stdout.
// The remote command is assembled with shell quoting so an argument is never
// re-split or interpreted by the remote shell.
func (n *NodeAgent) run(ctx context.Context, host string, args ...string) ([]byte, error) {
	remote := shellJoin(append([]string{n.script}, args...))
	sshArgs := []string{
		"-i", n.keyPath,
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=15",
		"-o", "StrictHostKeyChecking=accept-new",
		n.user + "@" + host,
		remote,
	}
	cmd := exec.CommandContext(ctx, "ssh", sshArgs...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("awg-peer on %s: %s", host, msg)
	}
	return []byte(stdout.String()), nil
}

// shellJoin single-quotes each token so the remote shell treats it literally.
func shellJoin(tokens []string) string {
	quoted := make([]string, len(tokens))
	for i, tok := range tokens {
		if tok != "" && !shellUnsafe.MatchString(tok) {
			quoted[i] = tok
			continue
		}
		quoted[i] = "'" + strings.ReplaceAll(tok, "'", `'\''`) + "'"
	}
	return strings.Join(quoted, " ")
}
