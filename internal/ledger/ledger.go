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

type state struct {
	Entries []Entry          `json:"entries"`
	Langs   map[int64]string `json:"langs,omitempty"`
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
	return l.save()
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
			l.data.Entries[i].ChatID = chatID
			l.data.Entries[i].Status = Delivered
			if err := l.save(); err != nil {
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
	l.data.Langs[chatID] = code
	return l.save()
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
		l.data.Entries = append(l.data.Entries[:i], l.data.Entries[i+1:]...)
		return true, l.save()
	}
	return false, nil
}

// save writes the ledger atomically: a temp file in the same directory, then a
// rename over the target so readers never observe a partial write.
func (l *Ledger) save() error {
	raw, err := json.MarshalIndent(l.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := l.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, l.path)
}
