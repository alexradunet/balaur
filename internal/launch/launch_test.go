package launch

import (
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
