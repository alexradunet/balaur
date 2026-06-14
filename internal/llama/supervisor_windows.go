//go:build windows

package llama

import "os/exec"

// setProcessGroup is a no-op on Windows: process-group semantics differ and the
// plain Process.Kill() in stop() is the available shutdown path. Windows is not
// a deployment target; this exists so the binary cross-compiles.
func setProcessGroup(cmd *exec.Cmd) {}

// killProcessGroup is a no-op on Windows; stop() falls back to Process.Kill().
func killProcessGroup(cmd *exec.Cmd) {}
