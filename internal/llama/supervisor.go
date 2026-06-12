// Package llama runs a local GGUF model behind an OpenAI-compatible HTTP
// endpoint by supervising a llamafile engine subprocess. Balaur talks to it
// through the same llm.OpenAIClient it uses for remote providers, so there is
// one model seam: everything is an OpenAI-compatible endpoint, and "local"
// just means an endpoint Balaur spawns and tends itself.
//
// This replaces in-process inference (kronk/llama.cpp via purego). The
// trade-off: Balaur now owns a child process (spawn, readiness probe, shutdown)
// instead of loading a shared library, in exchange for a self-contained engine
// and no llama.cpp-head tracking.
package llama

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/alexradunet/balaur/internal/llm"
)

// maxLoad bounds how long a fresh process may take to load a model and answer
// /health before EnsureServer gives up on this attempt. The process is left
// running on timeout so a later call can reuse it once the load finishes.
const maxLoad = 5 * time.Minute

// Supervisor owns at most one running llamafile process for the whole binary.
// Switching to a different GGUF stops the previous server and starts a new one.
type Supervisor struct {
	mu      sync.Mutex
	current *server
}

// Default is the process-wide supervisor. ClientSource builds local clients
// against it so every caller shares the single warm server.
var Default = &Supervisor{}

// EnginePath resolves the llamafile engine binary: BALAUR_LLAMAFILE if set,
// otherwise <dataDir>/bin/llamafile. The file is fetched/installed out of band;
// startup reports a clear error when it is missing.
func EnginePath(dataDir string) string {
	if p := os.Getenv("BALAUR_LLAMAFILE"); p != "" {
		return p
	}
	return filepath.Join(dataDir, "bin", "llamafile")
}

// EnsureServer guarantees a llamafile server is serving modelPath and returns
// its OpenAI base URL (…/v1). It reuses a running server for the same model and
// blocks until the model is loaded and answering /health, honoring ctx.
func (s *Supervisor) EnsureServer(ctx context.Context, enginePath, modelPath string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current != nil && (s.current.modelPath != modelPath || s.current.exited()) {
		s.current.stop()
		s.current = nil
	}
	if s.current == nil {
		srv, err := startServer(enginePath, modelPath)
		if err != nil {
			return "", err
		}
		s.current = srv
	}
	if err := s.current.waitReady(ctx); err != nil {
		return "", err
	}
	return s.current.baseURL, nil
}

// Stop tears down the running server, if any. Safe to call on shutdown.
func (s *Supervisor) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current != nil {
		s.current.stop()
		s.current = nil
	}
}

// server is one supervised llamafile process.
type server struct {
	modelPath string
	baseURL   string // http://127.0.0.1:<port>/v1
	healthURL string
	cmd       *exec.Cmd
	tail      *ringBuffer
	doneCh    chan struct{}
	exitErr   error

	readyMu sync.Mutex
	ready   bool
}

// isFatLlamafile reports whether modelPath is itself a llamafile executable
// (engine + weights in one file), which is run directly, versus a bare GGUF
// that needs a separate engine binary and -m.
func isFatLlamafile(modelPath string) bool {
	return filepath.Ext(modelPath) == ".llamafile"
}

func startServer(enginePath, modelPath string) (*server, error) {
	if _, err := os.Stat(modelPath); err != nil {
		return nil, fmt.Errorf("model file not found at %s", modelPath)
	}

	port, err := freePort()
	if err != nil {
		return nil, fmt.Errorf("allocating port for llamafile: %w", err)
	}

	serverArgs := []string{
		"--server",
		"--host", "127.0.0.1",
		"--port", strconv.Itoa(port),
	}

	// A fat llamafile is its own engine; a bare GGUF needs the separate engine
	// binary and -m.
	var name string
	var args []string
	if isFatLlamafile(modelPath) {
		// A llamafile is an APE (Actually Portable Executable). A raw execve()
		// of it fails with "exec format error" unless the host registered the
		// APE binfmt; running it through /bin/sh lets its polyglot header
		// bootstrap itself, which works everywhere a POSIX shell exists.
		_ = os.Chmod(modelPath, 0o755)
		name = "/bin/sh"
		args = append([]string{modelPath}, serverArgs...)
	} else {
		if enginePath == "" {
			return nil, fmt.Errorf("llamafile engine path is empty")
		}
		if _, err := os.Stat(enginePath); err != nil {
			return nil, fmt.Errorf("llamafile engine not found at %s: set BALAUR_LLAMAFILE or install it under <data>/bin/llamafile", enginePath)
		}
		_ = os.Chmod(enginePath, 0o755)
		name = enginePath
		args = append(serverArgs, "-m", modelPath)
	}

	tail := &ringBuffer{max: 8 * 1024}
	cmd := exec.Command(name, args...)
	cmd.Stdout = tail
	cmd.Stderr = tail
	// Own process group so stop() can reap any helper children llamafile spawns.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting llamafile: %w", err)
	}

	srv := &server{
		modelPath: modelPath,
		baseURL:   fmt.Sprintf("http://127.0.0.1:%d/v1", port),
		healthURL: fmt.Sprintf("http://127.0.0.1:%d/health", port),
		cmd:       cmd,
		tail:      tail,
		doneCh:    make(chan struct{}),
	}
	go func() {
		srv.exitErr = cmd.Wait()
		close(srv.doneCh)
	}()
	return srv, nil
}

func (s *server) exited() bool {
	select {
	case <-s.doneCh:
		return true
	default:
		return false
	}
}

// waitReady polls /health until the server answers, the process exits, or ctx
// is cancelled. It does not kill the process on ctx timeout — the load may
// still be in progress for a later caller.
func (s *server) waitReady(ctx context.Context) error {
	s.readyMu.Lock()
	defer s.readyMu.Unlock()
	if s.ready {
		return nil
	}

	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(maxLoad)
	for {
		select {
		case <-s.doneCh:
			return fmt.Errorf("llamafile exited before serving: %v\n%s", s.exitErr, s.tail.String())
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, s.healthURL, nil)
		resp, err := client.Do(req)
		if err == nil {
			code := resp.StatusCode
			resp.Body.Close()
			if code == http.StatusOK {
				s.ready = true
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out after %s waiting for llamafile to serve %s", maxLoad, filepath.Base(s.modelPath))
		}
		select {
		case <-time.After(300 * time.Millisecond):
		case <-ctx.Done():
			return ctx.Err()
		case <-s.doneCh:
			return fmt.Errorf("llamafile exited before serving: %v\n%s", s.exitErr, s.tail.String())
		}
	}
}

func (s *server) stop() {
	if s.cmd.Process == nil {
		return
	}
	// Kill the whole process group (Setpgid made the child its own leader).
	_ = syscall.Kill(-s.cmd.Process.Pid, syscall.SIGKILL)
	_ = s.cmd.Process.Kill()
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// ringBuffer keeps the last max bytes written to it for crash diagnostics.
type ringBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
	max int
}

func (r *ringBuffer) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buf.Write(p)
	if r.buf.Len() > r.max {
		over := r.buf.Len() - r.max
		r.buf.Next(over)
	}
	return len(p), nil
}

func (r *ringBuffer) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buf.String()
}

// LocalClient implements llm.Client by lazily ensuring a llamafile server for
// its GGUF and delegating to an OpenAIClient pointed at it. The server starts
// on first chat (not at construction), matching the old load-on-first-use
// behavior so building a client to probe config stays cheap.
type LocalClient struct {
	Engine string
	Model  string
	sup    *Supervisor

	mu    sync.Mutex
	inner *llm.OpenAIClient
}

// NewClient builds a local client bound to this supervisor.
func (s *Supervisor) NewClient(enginePath, modelPath string) *LocalClient {
	return &LocalClient{Engine: enginePath, Model: modelPath, sup: s}
}

func (c *LocalClient) ensure(ctx context.Context) (*llm.OpenAIClient, error) {
	base, err := c.sup.EnsureServer(ctx, c.Engine, c.Model)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.inner == nil || c.inner.BaseURL != base {
		// llama-server ignores the model name; send a non-empty placeholder.
		c.inner = &llm.OpenAIClient{BaseURL: base, Model: "local"}
	}
	return c.inner, nil
}

func (c *LocalClient) ChatStream(ctx context.Context, msgs []llm.Message, tools []llm.ToolSpec) (<-chan llm.Chunk, error) {
	inner, err := c.ensure(ctx)
	if err != nil {
		return nil, err
	}
	return inner.ChatStream(ctx, msgs, tools)
}

func (c *LocalClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	inner, err := c.ensure(ctx)
	if err != nil {
		return nil, err
	}
	return inner.Embed(ctx, texts)
}
