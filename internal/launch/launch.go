// Package launch holds the no-args loopback launcher: the smallest slice that
// lets a non-developer start Balaur without a shell. A bare `balaur` invocation
// (no subcommand, no flags) defaults the data dir to the XDG data dir, finds a
// free loopback port, and opens the browser — then hands control to the existing
// `serve` path by rewriting argv (see main.go). Every helper here is pure or
// trivially testable; the package never constructs a non-loopback address.
package launch

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// DataDir is the data directory a no-args launch uses. It mirrors
// kronk.ModelsDir()/LibRoot() exactly: an explicit BALAUR_DATA_DIR override wins
// (the same env name the Makefile already passes to --dir), else the XDG data dir
// ~/.local/share/balaur/pb_data (sibling of the existing models/ and kronk/lib
// XDG dirs, and the same path make run/the README use), with a relative "pb_data"
// fallback when the home dir cannot be resolved.
func DataDir() string {
	if d := os.Getenv("BALAUR_DATA_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "pb_data"
	}
	return filepath.Join(home, ".local", "share", "balaur", "pb_data")
}

// IsFirstRun reports whether dir does not yet exist — the cheap "this is the
// owner's first boot" signal the design note (Q1) reserves for Phase 2
// onboarding. It only stats; it never creates the dir, and the launcher never
// gates the browser-open on it (the browser opens on every no-args boot). Any
// stat error other than "not exist" (e.g. a permission error) is treated as
// "not first run" — onboarding should not trigger on an ambiguous filesystem.
func IsFirstRun(dir string) bool {
	_, err := os.Stat(dir)
	return errors.Is(err, fs.ErrNotExist)
}

// IsLauncherInvocation reports whether the process was invoked as a bare
// `balaur` with no arguments — the only case the no-args launcher fires. Pass
// os.Args[1:]. ANY argument (a subcommand, a flag, -h, `serve`, a CLI verb) is
// treated as an explicit, hands-off invocation, so the launcher can NEVER
// clobber an explicit `serve …` or any CLI command. Its safety rests entirely on
// "any arg ⇒ hands off".
func IsLauncherInvocation(args []string) bool {
	return len(args) == 0
}

// FreeLoopbackPort binds 127.0.0.1:0, reads the kernel-assigned port, closes the
// listener, and returns the port. There is a tiny inherent TOCTOU window between
// closing here and serve re-binding the port; that is acceptable for a localhost
// launcher (documented in the design note) and is not worth over-engineering.
func FreeLoopbackPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("finding a free loopback port: %w", err)
	}
	defer l.Close()
	addr, ok := l.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("finding a free loopback port: unexpected addr type %T", l.Addr())
	}
	return addr.Port, nil
}

// DefaultPort is the stable loopback port a no-args launch tries first, so the
// URL (http://127.0.0.1:8099/) is bookmarkable instead of changing every boot.
// It is deliberately in Balaur's 808x/809x family but is neither documented
// bind — make run (prod) uses 8080 and make dev uses 8090 — so a normal dev box
// never collides with a no-args launch. SelectPort falls back to a free port if
// 8099 is already taken, so this only needs to be a sensible default.
const DefaultPort = 8099

// SelectPort returns the launcher's loopback port: DefaultPort when it is free,
// otherwise a kernel-assigned free loopback port. It probes the default by
// binding 127.0.0.1:DefaultPort and closing immediately; on any bind error
// (port in use, permission) it falls back to FreeLoopbackPort. Like
// FreeLoopbackPort there is a tiny TOCTOU window before serve re-binds — that is
// acceptable for a localhost launcher (see docs/first-run-design.md).
func SelectPort() (int, error) {
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", DefaultPort))
	if err == nil {
		l.Close()
		return DefaultPort, nil
	}
	return FreeLoopbackPort()
}

// openCommand returns the OS-specific browser-open command for url. It is a pure
// function of (goos, url) so it is unit-testable without spawning a process
// (AGENTS.md: test platform logic through seams, not by mutating runtime.GOOS).
func openCommand(goos, url string) (name string, args []string) {
	switch goos {
	case "darwin":
		return "open", []string{url}
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default: // linux and other unix-likes
		return "xdg-open", []string{url}
	}
}

// OpenBrowser fires the OS browser-open command for url and returns immediately
// (Start, not Run — do not block on the browser). A failure is non-fatal to
// serving: the caller logs it and prints the URL so the owner can open it
// manually.
func OpenBrowser(url string) error {
	name, args := openCommand(runtime.GOOS, url)
	if err := exec.Command(name, args...).Start(); err != nil {
		return fmt.Errorf("opening browser for %s: %w", url, err)
	}
	return nil
}

// waitForListener dials addr on a short ticker until it accepts a connection or
// the deadline elapses. It returns nil once the server is reachable, or an error
// on timeout. Run it from its own goroutine — the ticker bounds the wait without
// blocking the serve goroutine.
func waitForListener(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("waiting for %s to accept connections: %w", addr, err)
		}
		<-ticker.C
	}
}

// OpenAfterReady waits for the loopback server at addr to start accepting, then
// opens the browser at http://<addr>/. Run it in a goroutine: it must not block
// the serve goroutine. On timeout or open failure it returns the error; the
// caller (main, pre-New) prints a manual-open URL to stderr — structured
// app.Logger() does not exist that early, the one allowed exception.
func OpenAfterReady(addr string) error {
	if err := waitForListener(addr, 15*time.Second); err != nil {
		return err
	}
	return OpenBrowser("http://" + addr + "/")
}
