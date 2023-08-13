package utils

// Section: help functions
// func TestAuditMutex(t *testing.T) {
// 	iw, err := ingresswatcher.SetupIngressWatcher(nil)
// 	if err != nil {
// 		t.Fatalf("Failed to setup IngressWatcher: %v", err)
// 	}

// 	t.Run("TestLockingMechanism", func(t *testing.T) {
// 		// Check if the lock is not acquired for the "default" namespace.
// 		locked := iw.auditMutex.IsLocked("default")
// 		assert.False(t, locked, "Expected the default namespace to not be locked")

// 		// Check if the lock is acquired for the "default" namespace.
// 		iw.auditMutex.Lock("default")
// 		locked = iw.auditMutex.IsLocked("default")
// 		assert.True(t, locked, "Expected the default namespace to be locked")

// 		// Check the function TryLock for when the lock is already acquired.
// 		if b := iw.auditMutex.TryLock("default"); b {
// 			t.Fatalf("Expected the default namespace to be locked")
// 		}

// 		// Check if the lock is not acquired for the "default" namespace.
// 		iw.auditMutex.Unlock("default")
// 		locked = iw.auditMutex.IsLocked("default")
// 		assert.False(t, locked, "Expected the default namespace to not be locked")

// 		// Check the function TryLock for when the lock is not acquired.
// 		if b := iw.auditMutex.TryLock("default"); !b {
// 			t.Fatalf("Expected the default namespace to not be locked")
// 		}
// 	})
// }
