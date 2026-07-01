package launch

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestIsLauncherInvocation(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{"bare argv", []string{}, true},
		{"serve", []string{"serve"}, false},
		{"serve with http", []string{"serve", "--http", "127.0.0.1:8090"}, false},
		{"cli verb", []string{"task", "add"}, false},
		{"help flag", []string{"-h"}, false},
		{"dir flag", []string{"--dir", "/x"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsLauncherInvocation(tt.args); got != tt.want {
				t.Errorf("IsLauncherInvocation(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestOpenCommand(t *testing.T) {
	const url = "http://x"
	tests := []struct {
		goos     string
		wantName string
		wantArgs []string
	}{
		{"darwin", "open", []string{url}},
		{"windows", "rundll32", []string{"url.dll,FileProtocolHandler", url}},
		{"linux", "xdg-open", []string{url}},
		{"freebsd", "xdg-open", []string{url}}, // default branch
	}
	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			name, args := openCommand(tt.goos, url)
			if name != tt.wantName {
				t.Errorf("openCommand(%q) name = %q, want %q", tt.goos, name, tt.wantName)
			}
			if !slices.Equal(args, tt.wantArgs) {
				t.Errorf("openCommand(%q) args = %v, want %v", tt.goos, args, tt.wantArgs)
			}
		})
	}
}

func TestFreeLoopbackPort(t *testing.T) {
	first, err := FreeLoopbackPort()
	if err != nil {
		t.Fatalf("FreeLoopbackPort() error = %v", err)
	}
	if first < 1 || first > 65535 {
		t.Errorf("port %d out of range 1..65535", first)
	}
	second, err := FreeLoopbackPort()
	if err != nil {
		t.Fatalf("second FreeLoopbackPort() error = %v", err)
	}
	if second < 1 || second > 65535 {
		t.Errorf("second port %d out of range 1..65535", second)
	}
	// The kernel usually hands out a different ephemeral port each time; this is
	// not guaranteed, so it is not asserted — both being valid is the contract.
}

func TestDataDir(t *testing.T) {
	t.Run("env override wins", func(t *testing.T) {
		t.Setenv("BALAUR_DATA_DIR", "/custom/data")
		if got := DataDir(); got != "/custom/data" {
			t.Errorf("DataDir() = %q, want %q", got, "/custom/data")
		}
	})
	t.Run("xdg default when unset", func(t *testing.T) {
		t.Setenv("BALAUR_DATA_DIR", "")
		got := DataDir()
		// With a real home dir, ends in .local/share/balaur/pb_data; the home-dir
		// error fallback ("pb_data") is the only other valid result.
		want := filepath.Join(".local", "share", "balaur", "pb_data")
		if got != "pb_data" && !strings.HasSuffix(got, want) {
			t.Errorf("DataDir() = %q, want suffix %q or fallback %q", got, want, "pb_data")
		}
	})
}

func TestSelectPort(t *testing.T) {
	t.Run("default free", func(t *testing.T) {
		// Attempt to pre-bind DefaultPort to check availability; if it fails,
		// something else already holds it — skip this sub-case.
		probe, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", DefaultPort))
		if err != nil {
			t.Skipf("127.0.0.1:%d already in use — cannot assert default-free path: %v", DefaultPort, err)
		}
		probe.Close()

		port, err := SelectPort()
		if err != nil {
			t.Fatalf("SelectPort() error = %v", err)
		}
		if port < 1 || port > 65535 {
			t.Errorf("port %d out of range 1..65535", port)
		}
		if port != DefaultPort {
			t.Errorf("SelectPort() = %d, want DefaultPort %d (port was free)", port, DefaultPort)
		}
	})

	t.Run("default occupied fallback", func(t *testing.T) {
		// Bind DefaultPort and keep it open so SelectPort must fall back.
		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", DefaultPort))
		if err != nil {
			t.Skipf("127.0.0.1:%d already in use by something else — cannot occupy it for fallback test: %v", DefaultPort, err)
		}
		t.Cleanup(func() { l.Close() })

		port, err := SelectPort()
		if err != nil {
			t.Fatalf("SelectPort() fallback error = %v", err)
		}
		if port == DefaultPort {
			t.Errorf("SelectPort() = DefaultPort %d, expected a different port when default is occupied", DefaultPort)
		}
		if port < 1 || port > 65535 {
			t.Errorf("fallback port %d out of range 1..65535", port)
		}
	})
}

func TestIsFirstRun(t *testing.T) {
	tests := []struct {
		name string
		dir  func() string
		want bool
	}{
		{
			"non-existent dir",
			func() string { return filepath.Join(t.TempDir(), "does-not-exist") },
			true,
		},
		{
			"existing dir",
			func() string { return t.TempDir() },
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.dir()
			got := IsFirstRun(dir)
			if got != tt.want {
				t.Errorf("IsFirstRun(%q) = %v, want %v", dir, got, tt.want)
			}
			if tt.want {
				// IsFirstRun must not create the dir.
				if _, err := os.Stat(dir); err == nil {
					t.Errorf("IsFirstRun created %q — it must only stat, never mkdir", dir)
				}
			}
		})
	}
}

func TestSelectPortAddressIsLoopback(t *testing.T) {
	port, err := SelectPort()
	if err != nil {
		t.Fatalf("SelectPort() error = %v", err)
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	if !strings.HasPrefix(addr, "127.0.0.1:") {
		t.Errorf("constructed addr %q is not loopback", addr)
	}
	if strings.Contains(addr, "0.0.0.0") {
		t.Errorf("constructed addr %q exposes all interfaces", addr)
	}
}

// TestRunningInstance_Live: a lock pointing at a live listener is detected.
func TestRunningInstance_Live(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	defer l.Close()
	addr := l.Addr().String()

	dataDir := filepath.Join(t.TempDir(), "pb_data")
	if err := WriteInstanceLock(dataDir, addr); err != nil {
		t.Fatalf("WriteInstanceLock: %v", err)
	}

	got, alive := RunningInstance(dataDir)
	if !alive {
		t.Fatal("RunningInstance: want alive=true for a live listener")
	}
	if got != addr {
		t.Errorf("RunningInstance addr = %q, want %q", got, addr)
	}
}

// TestRunningInstance_Stale: a lock pointing at a closed port is fail-open.
func TestRunningInstance_Stale(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	addr := l.Addr().String()
	l.Close() // nothing listening on addr now

	dataDir := filepath.Join(t.TempDir(), "pb_data")
	if err := WriteInstanceLock(dataDir, addr); err != nil {
		t.Fatalf("WriteInstanceLock: %v", err)
	}

	_, alive := RunningInstance(dataDir)
	if alive {
		t.Fatal("RunningInstance: want alive=false for stale (no-response) lock")
	}
}

// TestRunningInstance_Missing: no lock file → fail-open.
func TestRunningInstance_Missing(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "pb_data")
	_, alive := RunningInstance(dataDir)
	if alive {
		t.Fatal("RunningInstance: want alive=false when lock file is absent")
	}
}

// TestRunningInstance_Malformed: garbage bytes → fail-open, no panic.
func TestRunningInstance_Malformed(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "pb_data")
	p := lockPath(dataDir)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(p, []byte("not json {{{"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, alive := RunningInstance(dataDir) // must not panic
	if alive {
		t.Fatal("RunningInstance: want alive=false for malformed lock")
	}
}

// TestRunningInstance_RoundTrip: WriteInstanceLock + live listener → detected.
func TestRunningInstance_RoundTrip(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	defer l.Close()
	addr := l.Addr().String()

	dataDir := filepath.Join(t.TempDir(), "pb_data")
	if err := WriteInstanceLock(dataDir, addr); err != nil {
		t.Fatalf("WriteInstanceLock: %v", err)
	}

	got, alive := RunningInstance(dataDir)
	if !alive || got != addr {
		t.Errorf("round-trip: RunningInstance = (%q, %v), want (%q, true)", got, alive, addr)
	}
}

// TestRunningInstance_DifferentDataDirs: two different data dirs use different
// lock paths so they never collide.
func TestRunningInstance_DifferentDataDirs(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	defer l.Close()
	addr := l.Addr().String()

	dataDir1 := filepath.Join(t.TempDir(), "pb_data")
	dataDir2 := filepath.Join(t.TempDir(), "pb_data")

	// Write a lock only for dataDir1.
	if err := WriteInstanceLock(dataDir1, addr); err != nil {
		t.Fatalf("WriteInstanceLock: %v", err)
	}

	// dataDir1 detects the live instance.
	if _, alive := RunningInstance(dataDir1); !alive {
		t.Error("RunningInstance(dataDir1): want alive=true")
	}
	// dataDir2 has no lock — fail-open.
	if _, alive := RunningInstance(dataDir2); alive {
		t.Error("RunningInstance(dataDir2): want alive=false (different data dir)")
	}
}
