package turn

import (
	"sync"
	"testing"
)

func TestTryBeginFirstSucceeds(t *testing.T) {
	end, ok := TryBegin()
	if !ok {
		t.Fatal("TryBegin: first call must succeed (ok=true)")
	}
	defer end()
}

func TestTryBeginSecondRejected(t *testing.T) {
	end, ok := TryBegin()
	if !ok {
		t.Fatal("TryBegin: first call must succeed")
	}
	defer end()

	_, ok2 := TryBegin()
	if ok2 {
		t.Error("TryBegin: second call while held must return ok=false")
	}
}

func TestTryBeginSucceedsAfterEnd(t *testing.T) {
	end, ok := TryBegin()
	if !ok {
		t.Fatal("TryBegin: first call must succeed")
	}
	end()

	end2, ok2 := TryBegin()
	if !ok2 {
		t.Error("TryBegin: after end(), must succeed again (ok=true)")
	}
	end2()
}

func TestTryBeginEndIdempotent(t *testing.T) {
	end, ok := TryBegin()
	if !ok {
		t.Fatal("TryBegin: must succeed")
	}
	end()
	end() // idempotent via sync.Once — must not panic or deadlock

	end2, ok2 := TryBegin()
	if !ok2 {
		t.Error("TryBegin: after double end(), guard must be free")
	}
	end2()
}

// TestTryBeginRace fires N goroutines concurrently. The race detector verifies
// no data races; the held counter (protected by mu) proves exactly one
// goroutine holds the guard at a time.
func TestTryBeginRace(t *testing.T) {
	const n = 64
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		held    int
		maxHeld int
	)
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			end, ok := TryBegin()
			if !ok {
				return
			}
			mu.Lock()
			held++
			if held > maxHeld {
				maxHeld = held
			}
			if held > 1 {
				t.Errorf("concurrent holders: %d", held)
			}
			mu.Unlock()
			// Decrement before releasing the guard so held drops to 0 before
			// the next goroutine can increment it.
			mu.Lock()
			held--
			mu.Unlock()
			end()
		}()
	}
	wg.Wait()
	if maxHeld > 1 {
		t.Errorf("max concurrent holders = %d, want 1", maxHeld)
	}
}
