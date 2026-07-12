// Package ledger is the vpnbot's durable record of which panel user each Telegram
// chat has claimed, so a subscription is delivered to a person exactly once. It is
// owned by a single bot process: access is serialised with a mutex and every change
// is written atomically (temp file + rename) so a crash never leaves a torn file.
package ledger

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Status is the delivery state of a claim.
type Status string

const (
	// Pending means the claim token has been issued but not yet redeemed.
	Pending Status = "pending"
	// Delivered means the subscription has been handed to a bound chat.
	Delivered Status = "delivered"
)

// ErrNotFound is returned when no entry matches a lookup key.
var ErrNotFound = errors.New("ledger: entry not found")

// Entry links one panel user to the Telegram chat that claimed it.
type Entry struct {
	Username string `json:"username"`
	Token    string `json:"token"`
	ChatID   int64  `json:"chat_id,omitempty"`
	Status   Status `json:"status"`
}

// AWGPeer is a user's AmneziaWG peer at one location, kept so re-requesting the
// config re-sends the same keys and address instead of piling up peers.
type AWGPeer struct {
	Username   string `json:"username"`
	Location   string `json:"location"`
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
	Address    string `json:"address"`
}

type state struct {
	Entries  []Entry          `json:"entries"`
	Langs    map[int64]string `json:"langs,omitempty"`
	AWGPeers []AWGPeer        `json:"awg_peers,omitempty"`
}

// Ledger is a set of claim entries backed by a JSON file.
type Ledger struct {
	path string
	mu   sync.Mutex
	data state
}

// Open loads the ledger at path, creating its parent directory if needed. A
// missing file is treated as an empty ledger.
func Open(path string) (*Ledger, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	ledger := &Ledger{path: path}
	raw, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return ledger, nil
	case err != nil:
		return nil, err
	}
	if err := json.Unmarshal(raw, &ledger.data); err != nil {
		return nil, fmt.Errorf("ledger: parse %s: %w", path, err)
	}
	return ledger, nil
}

// Add records a new pending claim. It rejects a duplicate username or token.
func (l *Ledger) Add(username, token string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, entry := range l.data.Entries {
		if entry.Username == username {
			return fmt.Errorf("ledger: user %q already tracked", username)
		}
		if entry.Token == token {
			return fmt.Errorf("ledger: token collision")
		}
	}
	l.data.Entries = append(l.data.Entries, Entry{Username: username, Token: token, Status: Pending})
	if err := l.save(); err != nil {
		l.data.Entries = l.data.Entries[:len(l.data.Entries)-1]
		return err
	}
	return nil
}

// Claim binds chatID to the entry named by token and marks it delivered. The
// bool result is true only on the first redemption (Pending → Delivered), so a
// caller sends the one-time delivery exactly once; a repeat claim returns false.
func (l *Ledger) Claim(token string, chatID int64) (Entry, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i := range l.data.Entries {
		if l.data.Entries[i].Token != token {
			continue
		}
		first := l.data.Entries[i].Status == Pending
		if first {
			prevChat := l.data.Entries[i].ChatID
			l.data.Entries[i].ChatID = chatID
			l.data.Entries[i].Status = Delivered
			if err := l.save(); err != nil {
				l.data.Entries[i].Status = Pending
				l.data.Entries[i].ChatID = prevChat
				return Entry{}, false, err
			}
		}
		return l.data.Entries[i], first, nil
	}
	return Entry{}, false, ErrNotFound
}

// Lang returns a chat's saved language override, if one was set with /lang.
func (l *Ledger) Lang(chatID int64) (string, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	code, ok := l.data.Langs[chatID]
	return code, ok
}

// SetLang records a chat's language override.
func (l *Ledger) SetLang(chatID int64, code string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.data.Langs == nil {
		l.data.Langs = make(map[int64]string)
	}
	prev, had := l.data.Langs[chatID]
	l.data.Langs[chatID] = code
	if err := l.save(); err != nil {
		if had {
			l.data.Langs[chatID] = prev
		} else {
			delete(l.data.Langs, chatID)
		}
		return err
	}
	return nil
}

// ByUsername returns the entry tracked for a panel username, if any.
func (l *Ledger) ByUsername(username string) (Entry, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, entry := range l.data.Entries {
		if entry.Username == username {
			return entry, true
		}
	}
	return Entry{}, false
}

// AWGPeer returns a user's stored AmneziaWG peer for a location, if one exists.
func (l *Ledger) AWGPeer(username, location string) (AWGPeer, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, peer := range l.data.AWGPeers {
		if peer.Username == username && peer.Location == location {
			return peer, true
		}
	}
	return AWGPeer{}, false
}

// SaveAWGPeer records or updates a user's peer for a location (keyed by both).
func (l *Ledger) SaveAWGPeer(peer AWGPeer) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i := range l.data.AWGPeers {
		if l.data.AWGPeers[i].Username == peer.Username && l.data.AWGPeers[i].Location == peer.Location {
			prev := l.data.AWGPeers[i]
			l.data.AWGPeers[i] = peer
			if err := l.save(); err != nil {
				l.data.AWGPeers[i] = prev
				return err
			}
			return nil
		}
	}
	l.data.AWGPeers = append(l.data.AWGPeers, peer)
	if err := l.save(); err != nil {
		l.data.AWGPeers = l.data.AWGPeers[:len(l.data.AWGPeers)-1]
		return err
	}
	return nil
}

// ByChat returns the entry a chat has already claimed, if any.
func (l *Ledger) ByChat(chatID int64) (Entry, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, entry := range l.data.Entries {
		if entry.Status == Delivered && entry.ChatID == chatID {
			return entry, true
		}
	}
	return Entry{}, false
}

// List returns a copy of all entries.
func (l *Ledger) List() []Entry {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]Entry, len(l.data.Entries))
	copy(out, l.data.Entries)
	return out
}

// Remove drops the entry for username, returning whether one was present.
func (l *Ledger) Remove(username string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, entry := range l.data.Entries {
		if entry.Username != username {
			continue
		}
		removed := entry
		l.data.Entries = append(l.data.Entries[:i], l.data.Entries[i+1:]...)
		if err := l.save(); err != nil {
			l.data.Entries = append(l.data.Entries, Entry{})
			copy(l.data.Entries[i+1:], l.data.Entries[i:])
			l.data.Entries[i] = removed
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// AWGPeersFor returns every stored AmneziaWG peer for a user, across locations.
func (l *Ledger) AWGPeersFor(username string) []AWGPeer {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []AWGPeer
	for _, peer := range l.data.AWGPeers {
		if peer.Username == username {
			out = append(out, peer)
		}
	}
	return out
}

// DeleteAWGPeer drops a user's stored peer for a location, returning whether one
// was present. Used when access is revoked so the peer is not silently reused.
func (l *Ledger) DeleteAWGPeer(username, location string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, peer := range l.data.AWGPeers {
		if peer.Username != username || peer.Location != location {
			continue
		}
		removed := peer
		l.data.AWGPeers = append(l.data.AWGPeers[:i], l.data.AWGPeers[i+1:]...)
		if err := l.save(); err != nil {
			l.data.AWGPeers = append(l.data.AWGPeers, AWGPeer{})
			copy(l.data.AWGPeers[i+1:], l.data.AWGPeers[i:])
			l.data.AWGPeers[i] = removed
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// save writes the ledger atomically and durably: a temp file in the same directory
// is written and fsync'd, renamed over the target so readers never observe a partial
// write, then the parent directory is fsync'd so the rename survives a power loss.
// Callers mutate l.data first and roll the change back if this returns an error, so a
// failed write never leaves memory ahead of disk.
func (l *Ledger) save() error {
	raw, err := json.MarshalIndent(l.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := l.path + ".tmp"
	file, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(raw); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, l.path); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(l.path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
