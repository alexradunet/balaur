//go:build windows

package ollama

import "os/exec"

// setProcessGroup is a no-op on Windows: process-group semantics differ and the
// plain Process.Kill() in Stop() is the available shutdown path. Windows is not
// a deployment target; this exists so the binary cross-compiles.
func setProcessGroup(cmd *exec.Cmd) {}

// killProcessGroup is a no-op on Windows; Stop() falls back to Process.Kill().
func killProcessGroup(cmd *exec.Cmd) {}
