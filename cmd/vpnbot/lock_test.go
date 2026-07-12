package main

import (
	"sync"
	"testing"
	"time"
)

// The per-user lock must serialise concurrent operations on one username: run under
// -race, an unsynchronised counter++ across goroutines would race or lose updates.
func TestLockUserSerialisesSameUser(t *testing.T) {
	a := &app{}
	const goroutines = 50
	counter := 0
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock := a.lockUser("alice")
			defer unlock()
			counter++
		}()
	}
	wg.Wait()
	if counter != goroutines {
		t.Fatalf("counter = %d, want %d (lost updates: lock not serialising)", counter, goroutines)
	}
}

// Different users must not block each other, so one slow node can't stall the bot.
func TestLockUserDifferentUsersDoNotBlock(t *testing.T) {
	a := &app{}
	unlock := a.lockUser("alice")
	defer unlock()
	done := make(chan struct{})
	go func() {
		release := a.lockUser("bob")
		release()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("lockUser(bob) blocked while alice was held")
	}
}
