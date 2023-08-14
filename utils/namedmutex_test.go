package utils

import (
	"sync"
	"testing"
	"time"
)

// TestNamedMutex provides a table-driven test for the NamedMutex functions.
// It tests basic locking, unlocking, and the TryLock functionality.
func TestNamedMutex(t *testing.T) {
	tests := []struct {
		name     string   // Name of the test case.
		sequence []string // Sequence of actions to perform.
		want     []bool   // Expected results after each step in the sequence.
	}{
		{"Lock and Unlock", []string{"Lock", "IsLocked", "Unlock", "IsLocked"}, []bool{true, true, true, false}},
		{"TryLock when unlocked", []string{"TryLock", "IsLocked", "Unlock", "IsLocked"}, []bool{true, true, true, false}},
		{"TryLock when locked", []string{"Lock", "TryLock", "Unlock", "IsLocked"}, []bool{true, false, true, false}},
		{"IsLocked when Unlock", []string{"Unlock", "IsLocked", "TryLock", "IsLocked"}, []bool{true, false, true, true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nm := NewNamedMutex()
			for i, action := range tt.sequence {
				switch action {
				case "Lock":
					nm.Lock(tt.name)
				case "Unlock":
					nm.Unlock(tt.name)
				case "TryLock":
					if got := nm.TryLock(tt.name); got != tt.want[i] {
						t.Errorf("TryLock() = %v; want %v", got, tt.want[i])
					}
				case "IsLocked":
					if got := nm.IsLocked(tt.name); got != tt.want[i] {
						t.Errorf("IsLocked() = %v; want %v", got, tt.want[i])
					}
				}
			}
		})
	}
}

// TestConcurrentAccess tests the behavior of NamedMutex under concurrent access.
// It ensures that when one goroutine has locked a mutex, another goroutine cannot lock it.
func TestSimpleConcurrentAccess(t *testing.T) {
	nm := NewNamedMutex()
	var wg sync.WaitGroup
	wg.Add(2)

	// First goroutine locks the mutex.
	go func() {
		defer wg.Done()
		nm.Lock("concurrent")
		// Introduce a delay to ensure that the second goroutine has started and is waiting on the TryLock.
		time.Sleep(100 * time.Millisecond)
		nm.Unlock("concurrent")
	}()

	// Second goroutine tries to lock the mutex after a short delay.
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond) // Give the first goroutine a head start.
		if nm.TryLock("concurrent") {
			t.Error("Expected TryLock to fail when already locked by another goroutine")
		}
	}()

	wg.Wait()
}

// testMultipleNamedLocks tests that different named locks operate independently.
// func TestMultipleNamedLocks(t *testing.T) {
// 	nm := NewNamedMutex()
// 	var wg sync.WaitGroup
// 	wg.Add(2)

// 	// First goroutine locks the mutex.
// 	go func() {
// 		defer wg.Done()
// 		nm.Lock("lock1")
// 		// Introduce a delay to ensure that the second goroutine has started and is waiting on the TryLock.
// 		time.Sleep(100 * time.Millisecond)
// 		nm.Unlock("lock1")
// 	}()

// 	// Second goroutine tries to lock the mutex after a short delay.
// 	go func() {
// 		defer wg.Done()
// 		time.Sleep(10 * time.Millisecond) // Give the first goroutine a head start.
// 		if !nm.TryLock("lock2") {
// 			t.Error("Expected TryLock to succeed when not locked by another goroutine")
// 		}
// 		nm.Unlock("lock2")
// 	}()

// 	wg.Wait()
// }

func TestMultipleNamedLocks(t *testing.T) {
	tests := []struct {
		name     string   // Name of the test case.
		sequence []string // Sequence of actions to perform.
		want     []bool   // Expected results after each step in the sequence.
	}{
		{"Concurrent Different Locks", []string{"ConcurrentLock1", "ConcurrentTryLock2", "ConcurrentUnlock1", "ConcurrentUnlock2", "IsLocked1", "IsLocked2"}, []bool{true, true, true, true, false, false}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nm := NewNamedMutex()
			var wg sync.WaitGroup

			for idx, action := range tt.sequence {
				switch action {
				case "ConcurrentLock1":
					wg.Add(1)
					go func(idx int) {
						defer wg.Done()
						nm.Lock("lock1")
						time.Sleep(100 * time.Millisecond)
					}(idx)
				case "ConcurrentUnlock1":
					wg.Add(1)
					go func(idx int) {
						defer wg.Done()
						time.Sleep(150 * time.Millisecond)
						nm.Unlock("lock1")
					}(idx)
				case "ConcurrentTryLock2":
					wg.Add(1)
					go func(idx int) { // Pass idx as an argument
						defer wg.Done()
						time.Sleep(50 * time.Millisecond)
						if got := nm.TryLock("lock2"); got != tt.want[idx] {
							t.Errorf("ConcurrentTryLock2 = %v; want %v", got, tt.want[idx])
						}
					}(idx) // Pass idx as an argument
				case "ConcurrentUnlock2":
					wg.Add(1)
					go func(idx int) {
						defer wg.Done()
						time.Sleep(200 * time.Millisecond)
						nm.Unlock("lock2")
					}(idx)
				case "IsLocked1":
					wg.Wait() // Ensure all goroutines have finished
					if got := nm.IsLocked("lock1"); got != tt.want[idx] {
						t.Errorf("IsLocked1 = %v; want %v", got, tt.want[idx])
					}
				case "IsLocked2":
					if got := nm.IsLocked("lock2"); got != tt.want[idx] {
						t.Errorf("IsLocked2 = %v; want %v", got, tt.want[idx])
					}
				}
			}
		})
	}
}
