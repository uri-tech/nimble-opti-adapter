// internal/controller/named_mutex.go
package utils

import (
	"sync"
)

// NamedMutex provides a mutex with named locking.
type NamedMutex struct {
	mu    sync.Mutex
	locks map[string]*lock
}

type lock struct {
	mutex  chan struct{}
	locked bool
}

// NewNamedMutex initializes a new NamedMutex.
func NewNamedMutex() *NamedMutex {
	return &NamedMutex{
		locks: make(map[string]*lock),
	}
}

// Lock locks the named mutex.
func (n *NamedMutex) Lock(name string) {
	n.mu.Lock()
	if _, exists := n.locks[name]; !exists {
		n.locks[name] = &lock{
			mutex: make(chan struct{}, 1),
		}
		n.locks[name].mutex <- struct{}{}
	}
	lk := n.locks[name]
	n.mu.Unlock()

	<-lk.mutex
	lk.locked = true
}

// Unlock unlocks the named mutex.
func (n *NamedMutex) Unlock(name string) {
	n.mu.Lock()
	lk, exists := n.locks[name]
	n.mu.Unlock()

	if exists && lk.locked {
		lk.locked = false
		lk.mutex <- struct{}{}
	}
}

// TryLock attempts to lock the named mutex and returns true if successful.
func (n *NamedMutex) TryLock(name string) bool {
	n.mu.Lock()
	if _, exists := n.locks[name]; !exists {
		n.locks[name] = &lock{
			mutex: make(chan struct{}, 1),
		}
		n.locks[name].mutex <- struct{}{}
	}
	lk := n.locks[name]
	n.mu.Unlock()

	select {
	case <-lk.mutex:
		lk.locked = true
		return true
	default:
		return false
	}
}

// IsLocked checks if the named mutex is locked.
func (n *NamedMutex) IsLocked(name string) bool {
	n.mu.Lock()
	lk, exists := n.locks[name]
	n.mu.Unlock()

	if exists {
		return lk.locked
	}
	return false
}
