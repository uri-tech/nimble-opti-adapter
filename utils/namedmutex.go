// internal/controller/named_mutex.go
package utils

import (
	"sync"
)

// NamedMutex provides a mechanism for named locking. This allows for fine-grained locking
// based on string names, ensuring that only one goroutine can hold a lock for a given name at a time.
type NamedMutex struct {
	mu    sync.Mutex       // mu protects concurrent access to the locks map.
	locks map[string]*lock // locks holds the individual named mutexes.
}

// lock represents an individual named mutex.
type lock struct {
	mutex  chan struct{} // mutex is a channel used for the actual locking mechanism.
	locked bool          // locked indicates whether the mutex is currently held.
}

// NewNamedMutex initializes and returns a new NamedMutex.
func NewNamedMutex() *NamedMutex {
	return &NamedMutex{
		locks: make(map[string]*lock),
	}
}

// Lock acquires the named mutex. If the mutex is already locked, the calling goroutine
// blocks until the mutex is available. If the named mutex does not exist, it is created.
func (n *NamedMutex) Lock(name string) {
	n.mu.Lock() // Acquire the global mutex to safely access the locks map.

	// Check if the named lock exists.
	if _, exists := n.locks[name]; !exists {
		// If the named lock doesn't exist, initialize it.
		n.locks[name] = &lock{
			mutex: make(chan struct{}, 1),
		}
		n.locks[name].mutex <- struct{}{} // "Unlock" the mutex by sending an empty struct to its channel.
	}
	lk := n.locks[name]

	n.mu.Unlock() // Release the global mutex.

	<-lk.mutex       // Block until we can receive from the channel, effectively acquiring the lock.
	lk.locked = true // Mark the mutex as locked.
}

// Unlock releases the named mutex. It's a runtime error to unlock a mutex that is
// not locked or does not exist. However, for simplicity, this implementation does not panic in such cases.
func (n *NamedMutex) Unlock(name string) {
	n.mu.Lock() // Acquire the global mutex to safely access the locks map.

	// Retrieve the named lock.
	lk, exists := n.locks[name]

	n.mu.Unlock() // Release the global mutex.

	if exists && lk.locked {
		lk.locked = false      // Mark the mutex as unlocked.
		lk.mutex <- struct{}{} // "Unlock" the mutex by sending an empty struct to its channel.
	}
}

// TryLock attempts to acquire the named mutex without waiting.
// If the mutex is already locked, the function returns false.
// If the mutex is not locked, the function locks it and returns true.
func (n *NamedMutex) TryLock(name string) bool {
	n.mu.Lock() // Acquire the global mutex to safely access the locks map.

	// Check if the named lock exists.
	lk, exists := n.locks[name]
	if !exists {
		// If the named lock doesn't exist, initialize it.
		lk = &lock{
			mutex: make(chan struct{}, 1),
		}
		n.locks[name] = lk
		lk.mutex <- struct{}{} // "Unlock" the mutex by sending an empty struct to its channel.
	} else if lk.locked {
		// If the mutex exists and is already locked, release the global mutex and return false immediately.
		n.mu.Unlock()
		return false
	}

	n.mu.Unlock() // Release the global mutex.

	// Try to acquire the named mutex.
	select {
	case <-lk.mutex: // If we can receive from the channel, it means the mutex is "unlocked".
		lk.locked = true // Mark the mutex as locked.
		return true
	default: // If we can't receive from the channel, it means the mutex is already locked by another goroutine.
		return false
	}
}

// IsLocked checks the lock status of a named mutex.
// It returns true if the named mutex is currently locked, and false otherwise.
// If the named mutex does not exist, it also returns false.
func (n *NamedMutex) IsLocked(name string) bool {
	n.mu.Lock() // Acquire the global mutex to safely access the locks map.

	// Retrieve the named lock.
	lk, exists := n.locks[name]

	n.mu.Unlock() // Release the global mutex.

	// If the named lock exists, return its locked status.
	// Otherwise, return false.
	if exists {
		return lk.locked
	}
	return false
}
