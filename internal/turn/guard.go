package turn

import "sync"

// turnMu is the process-wide in-flight guard. TryBegin acquires it with
// TryLock (immediate "busy" — never blocking). One guard for v1's single
// master conversation; key by conversation id if multiple turn-bearing
// conversations are ever added.
var turnMu sync.Mutex

// TryBegin acquires the process-wide "a turn is in flight" guard. Returns an
// end func to release it and ok=true on success; ok=false means another turn
// is already running — the caller must reject with a medium-appropriate "busy"
// response and must NOT start a turn.
//
// The guard lives here, not inside Run, because each gateway must decide its
// busy response before setting up its medium: web must reject before cs.start
// (otherwise the user bubble is already painted when the guard fires), CLI
// before JSON output, messenger before body parse. Acquire at the gateway's
// very top, then defer end().
//
// Keyed globally for v1's single master conversation. If multiple
// turn-bearing conversations are ever added, key by conversation id instead.
func TryBegin() (end func(), ok bool) {
	if !turnMu.TryLock() {
		return nil, false
	}
	var once sync.Once
	return func() { once.Do(turnMu.Unlock) }, true
}
