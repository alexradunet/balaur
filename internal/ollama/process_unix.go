//go:build unix

package ollama

import (
	"os/exec"
	"syscall"
)

// setProcessGroup makes the child its own process-group leader so
// killProcessGroup can reap any helper children `ollama serve` spawns.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessGroup SIGKILLs the whole process group led by the child.
// Caller guarantees cmd.Process != nil.
func killProcessGroup(cmd *exec.Cmd) {
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
