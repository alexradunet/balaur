package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Model is one model present in Ollama's local store. Field names mirror
// gguf.FileInfo (Name, Size, Path) so existing templates bind unchanged; Path
// is always empty (Ollama owns the blob store, Balaur has no file path).
type Model struct {
	Name string
	Size int64
	Path string
}

// PullProgress is one streamed line of `ollama pull` progress.
type PullProgress struct {
	Status    string `json:"status"`
	Completed int64  `json:"completed"`
	Total     int64  `json:"total"`
}

// api is a thin HTTP client for Ollama's native /api endpoints. Inference uses
// /v1 via llm.OpenAIClient instead; this is only for model + readiness ops.
type api struct {
	host  string // host:port, no scheme
	httpc *http.Client
}

func newAPI() *api {
	return &api{host: Host(), httpc: &http.Client{Timeout: 0}}
}

func (a *api) base() string { return "http://" + a.host }

func (a *api) up(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, a.base()+"/api/tags", nil)
	resp, err := a.httpc.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (a *api) tags(ctx context.Context) ([]Model, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, a.base()+"/api/tags", nil)
	resp, err := a.httpc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama /api/tags: status %d", resp.StatusCode)
	}
	var body struct {
		Models []struct {
			Name string `json:"name"`
			Size int64  `json:"size"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	out := make([]Model, 0, len(body.Models))
	for _, m := range body.Models {
		out = append(out, Model{Name: m.Name, Size: m.Size})
	}
	return out, nil
}

func (a *api) pull(ctx context.Context, tag string, onProgress func(PullProgress)) error {
	payload, _ := json.Marshal(map[string]any{"model": tag, "stream": true})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, a.base()+"/api/pull", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.httpc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama /api/pull: status %d", resp.StatusCode)
	}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var p PullProgress
		if err := json.Unmarshal(line, &p); err != nil {
			continue
		}
		// Ollama reports errors as {"error":"..."}; surface them.
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(line, &e)
		if e.Error != "" {
			return fmt.Errorf("ollama pull: %s", e.Error)
		}
		if onProgress != nil {
			onProgress(p)
		}
	}
	return scanner.Err()
}

func (a *api) delete(ctx context.Context, tag string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	payload, _ := json.Marshal(map[string]any{"model": tag})
	req, _ := http.NewRequestWithContext(ctx, http.MethodDelete, a.base()+"/api/delete", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.httpc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama /api/delete: status %d", resp.StatusCode)
	}
	return nil
}
