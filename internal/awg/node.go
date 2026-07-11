package awg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// Default SSH wiring. On the panel host the bot uses a dedicated key that each node
// restricts to the awg-peer command with a forced command, so it can do nothing
// else there.
const (
	DefaultUser   = "root"
	DefaultScript = "/usr/local/sbin/awg-peer"
	dialTimeout   = 15 * time.Second
)

var (
	// peerNameSafe is the charset the node agent accepts for a peer name; the bot
	// sanitises to it so a username can never inject shell metacharacters.
	peerNameSafe = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)
	// argSafe is the charset allowed in a remote command argument. Every argument
	// (fixed subcommands, sanitised names, base64 keys) fits it, so the remote
	// command is a plain space-joined string that both a login shell and the
	// forced-command wrapper's word-split parse identically - no quoting needed.
	argSafe = regexp.MustCompile(`^[A-Za-z0-9_.:/=+@%-]+$`)
)

// NodeAgent runs the node-side awg-peer script over SSH (pure Go, so it works in a
// scratch container).
type NodeAgent struct {
	user      string
	script    string
	signer    ssh.Signer
	hostKeyCB ssh.HostKeyCallback
}

// NewNodeAgent returns an agent that logs in as user with the private key at keyPath
// and runs script on the target node. Empty user/script fall back to the defaults.
// When knownHostsPath is set the node's host key is verified against it; otherwise
// host keys are not verified.
func NewNodeAgent(keyPath, user, script, knownHostsPath string) (*NodeAgent, error) {
	if user == "" {
		user = DefaultUser
	}
	if script == "" {
		script = DefaultScript
	}
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read ssh key: %w", err)
	}
	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("parse ssh key: %w", err)
	}
	hostKeyCB := ssh.InsecureIgnoreHostKey()
	if knownHostsPath != "" {
		hostKeyCB, err = knownhosts.New(knownHostsPath)
		if err != nil {
			return nil, fmt.Errorf("load known_hosts: %w", err)
		}
	}
	return &NodeAgent{user: user, script: script, signer: signer, hostKeyCB: hostKeyCB}, nil
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

// run dials host over SSH, runs the awg-peer script with args, and returns stdout.
func (n *NodeAgent) run(ctx context.Context, host string, args ...string) ([]byte, error) {
	cmd, err := remoteCommand(n.script, args)
	if err != nil {
		return nil, err
	}
	addr := net.JoinHostPort(host, "22")
	dialer := net.Dialer{Timeout: dialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", host, err)
	}
	cfg := &ssh.ClientConfig{
		User:            n.user,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(n.signer)},
		HostKeyCallback: n.hostKeyCB,
		Timeout:         dialTimeout,
	}
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, cfg)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("ssh handshake %s: %w", host, err)
	}
	client := ssh.NewClient(sshConn, chans, reqs)
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()
	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	done := make(chan error, 1)
	go func() { done <- session.Run(cmd) }()
	select {
	case <-ctx.Done():
		session.Close()
		return nil, ctx.Err()
	case err := <-done:
		if err != nil {
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				msg = err.Error()
			}
			return nil, fmt.Errorf("awg-peer on %s: %s", host, msg)
		}
	}
	return stdout.Bytes(), nil
}

// remoteCommand joins the script and args into the command string sent over SSH.
// Every argument must be shell-safe so the string needs no quoting - the node's
// forced-command wrapper word-splits it, which quoting would defeat.
func remoteCommand(script string, args []string) (string, error) {
	parts := append([]string{script}, args...)
	for _, part := range parts {
		if !argSafe.MatchString(part) {
			return "", fmt.Errorf("unsafe remote argument %q", part)
		}
	}
	return strings.Join(parts, " "), nil
}
