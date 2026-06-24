package ext

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dop251/goja"
)

// The JS surface an extension sees. Deliberately tiny (suckless): goja's
// ECMAScript builtins (JSON, Math, RegExp…), console.log (no-op),
// balaur.registerTool, and — inside handlers only — balaur.http. No npm,
// no require, no $os, no DB. OS reach stays the OS tools' concern behind
// its own gate; extensions are for new verbs, not new privileges.

// goja is pinned to a reviewed master commit in go.mod (no upstream
// semver tags exist) because this VM runs untrusted-author extension JS.
// Any goja version bump MUST be gated on `go test ./internal/ext/...`
// (the sandbox-boundary regression suite) and a review of the new commit —
// govulncheck catches CVEs, not behavioral sandbox-escape changes.

// maxToolOutput bounds what one extension call feeds back to the model
// (mirrors internal/tools.maxOutput).
const maxToolOutput = 48 * 1024

// maxHTTPBody bounds a fetched response body before it reaches JS.
const maxHTTPBody = 256 * 1024

// invokeTimeout bounds one handler run; the VM is interrupted after it.
const invokeTimeout = 30 * time.Second

// httpTimeout bounds one balaur.http request inside a handler.
const httpTimeout = 15 * time.Second

// extHTTPClient never follows redirects: an approved extension's reviewed
// code is exactly what runs — a redirect chain must be followed explicitly
// by the handler if it wants to. Local addresses stay deliberately
// reachable (see httpBinding's comment).
var extHTTPClient = &http.Client{
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// ToolDef is one tool an extension registers.
type ToolDef struct {
	Name        string
	Description string
	Parameters  map[string]any
}

type captured struct {
	def     ToolDef
	handler goja.Callable
}

// newVM builds a fresh runtime and runs src, capturing registerTool calls.
// withHTTP=false is extract mode: loading an extension must be free of
// side effects, so balaur.http throws there — effects happen only inside
// an invoked handler, where they are audited per call.
func newVM(ctx context.Context, src, name string, withHTTP bool) (*goja.Runtime, []captured, error) {
	vm := goja.New()
	vm.SetFieldNameMapper(goja.UncapFieldNameMapper())

	var regs []captured
	balaur := vm.NewObject()
	_ = balaur.Set("registerTool", func(call goja.FunctionCall) goja.Value {
		obj := call.Argument(0).ToObject(vm)
		if obj == nil {
			panic(vm.NewTypeError("registerTool: want an object"))
		}
		def := ToolDef{
			Name:        strings.TrimSpace(obj.Get("name").String()),
			Description: strings.TrimSpace(obj.Get("description").String()),
		}
		if params, ok := obj.Get("parameters").Export().(map[string]any); ok {
			def.Parameters = params
		}
		handler, ok := goja.AssertFunction(obj.Get("handler"))
		if def.Name == "" || def.Description == "" || !ok {
			panic(vm.NewTypeError("registerTool: name, description, and handler are required"))
		}
		regs = append(regs, captured{def: def, handler: handler})
		return goja.Undefined()
	})
	if withHTTP {
		_ = balaur.Set("http", httpBinding(ctx, vm))
	} else {
		_ = balaur.Set("http", func(goja.FunctionCall) goja.Value {
			panic(vm.NewTypeError("balaur.http is only available inside a tool handler, never at load time"))
		})
	}
	_ = vm.Set("balaur", balaur)

	console := vm.NewObject()
	_ = console.Set("log", func(goja.FunctionCall) goja.Value { return goja.Undefined() })
	_ = vm.Set("console", console)

	prog, err := goja.Compile(name, src, true)
	if err != nil {
		return nil, nil, fmt.Errorf("compile: %w", err)
	}
	if _, err := vm.RunProgram(prog); err != nil {
		return nil, nil, fmt.Errorf("load: %w", err)
	}
	return vm, regs, nil
}

// extract loads src side-effect-free and returns the tools it registers.
func extract(src, name string) ([]ToolDef, error) {
	_, regs, err := newVM(context.Background(), src, name, false)
	if err != nil {
		return nil, err
	}
	defs := make([]ToolDef, 0, len(regs))
	for _, r := range regs {
		defs = append(defs, r.def)
	}
	return defs, nil
}

// invoke runs src in a fresh VM and calls the named tool's handler.
// Fresh-VM-per-call keeps extensions stateless and goroutine-safe by
// construction; small files compile in well under a millisecond.
func invoke(ctx context.Context, src, name, tool, argsJSON string) (out string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("extension %s: %v", name, r)
		}
	}()

	vm, regs, err := newVM(ctx, src, name, true)
	if err != nil {
		return "", fmt.Errorf("extension %s: %w", name, err)
	}

	var handler goja.Callable
	for _, r := range regs {
		if r.def.Name == tool {
			handler = r.handler
			break
		}
	}
	if handler == nil {
		return "", fmt.Errorf("extension %s registers no tool %q", name, tool)
	}

	args := map[string]any{}
	if strings.TrimSpace(argsJSON) != "" {
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("%s: bad arguments: %w", tool, err)
		}
	}

	// Interrupt on deadline or caller cancellation; never let a handler
	// hold the turn hostage.
	t := time.NewTimer(invokeTimeout)
	defer t.Stop()
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			vm.Interrupt("context cancelled")
		case <-t.C:
			vm.Interrupt("extension timed out")
		case <-done:
		}
	}()

	res, err := handler(goja.Undefined(), vm.ToValue(args))
	if err != nil {
		return "", fmt.Errorf("%s: %w", tool, err)
	}
	return renderResult(res)
}

func renderResult(v goja.Value) (string, error) {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return "", nil
	}
	if s, ok := v.Export().(string); ok {
		return clip(s), nil
	}
	raw, err := json.Marshal(v.Export())
	if err != nil {
		return "", fmt.Errorf("result not serializable: %w", err)
	}
	return clip(string(raw)), nil
}

func clip(s string) string {
	if len(s) <= maxToolOutput {
		return s
	}
	return s[:maxToolOutput] + "\n…(truncated)"
}

// httpBinding implements balaur.http({url, method?, headers?, body?}) →
// {status, body, headers}. Errors throw as JS exceptions so handlers can
// try/catch. Local addresses are deliberately reachable: a personal box
// talks to its own services; the audit log carries every invocation.
func httpBinding(ctx context.Context, vm *goja.Runtime) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		opts, _ := call.Argument(0).Export().(map[string]any)
		if opts == nil {
			panic(vm.NewTypeError("balaur.http: want {url, method?, headers?, body?}"))
		}
		url, _ := opts["url"].(string)
		if url == "" {
			panic(vm.NewTypeError("balaur.http: url is required"))
		}
		method, _ := opts["method"].(string)
		if method == "" {
			method = http.MethodGet
		}
		var body io.Reader
		if b, ok := opts["body"].(string); ok && b != "" {
			body = strings.NewReader(b)
		}

		reqCtx, cancel := context.WithTimeout(ctx, httpTimeout)
		defer cancel()
		req, err := http.NewRequestWithContext(reqCtx, strings.ToUpper(method), url, body)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		if headers, ok := opts["headers"].(map[string]any); ok {
			for k, v := range headers {
				if s, ok := v.(string); ok {
					req.Header.Set(k, s)
				}
			}
		}
		resp, err := extHTTPClient.Do(req)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		defer resp.Body.Close()
		raw, err := io.ReadAll(io.LimitReader(resp.Body, maxHTTPBody))
		if err != nil {
			panic(vm.NewGoError(err))
		}
		flat := map[string]any{}
		for k := range resp.Header {
			flat[k] = resp.Header.Get(k)
		}
		return vm.ToValue(map[string]any{
			"status":  resp.StatusCode,
			"body":    string(raw),
			"headers": flat,
		})
	}
}
