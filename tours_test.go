package main

// TestTours validates every .tours/*.tour file:
//   - valid JSON
//   - title non-empty
//   - every step with a "file" field → file exists at repo root-relative path
//     and "line" (when present) is ≥ 1 and ≤ the file's actual line count
//   - every step with a "directory" field → directory exists
//
// A failing tour/step prints the tour title, step title, and what broke so a
// CI failure message is self-explaining.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type tourFile struct {
	Title string     `json:"title"`
	Steps []tourStep `json:"steps"`
}

type tourStep struct {
	Title     string `json:"title"`
	File      string `json:"file"`
	Directory string `json:"directory"`
	Line      int    `json:"line"`
}

// lineCount returns the number of newline-terminated lines in a file.
// It is an O(file-size) scan; fine for source files in tests.
func lineCount(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	n := 0
	for sc.Scan() {
		n++
	}
	return n, sc.Err()
}

func TestTours(t *testing.T) {
	// Resolve repository root (where go.mod lives).
	// When run via "go test .", the working directory is already the module root.
	repoRoot, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("cannot resolve repo root: %v", err)
	}

	tourGlob := filepath.Join(repoRoot, ".tours", "*.tour")
	paths, err := filepath.Glob(tourGlob)
	if err != nil {
		t.Fatalf("glob %s: %v", tourGlob, err)
	}
	if len(paths) == 0 {
		t.Fatalf("no .tour files found under .tours/ — expected at least 11")
	}

	t.Logf("checking %d tour file(s)", len(paths))

	for _, path := range paths {
		name := filepath.Base(path)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("[%s] cannot read file: %v", name, err)
			continue
		}

		var tf tourFile
		if err := json.Unmarshal(raw, &tf); err != nil {
			t.Errorf("[%s] invalid JSON: %v", name, err)
			continue
		}
		if strings.TrimSpace(tf.Title) == "" {
			t.Errorf("[%s] tour has no title", name)
		}

		for i, step := range tf.Steps {
			stepID := step.Title
			if stepID == "" {
				stepID = fmt.Sprintf("step[%d]", i)
			}

			// Directory step check.
			if step.Directory != "" {
				dir := filepath.Join(repoRoot, step.Directory)
				if info, err := os.Stat(dir); err != nil {
					t.Errorf("[%s] %s: directory %q does not exist: %v",
						name, stepID, step.Directory, err)
				} else if !info.IsDir() {
					t.Errorf("[%s] %s: directory %q is not a directory",
						name, stepID, step.Directory)
				}
			}

			// File step check.
			if step.File == "" {
				continue
			}
			absFile := filepath.Join(repoRoot, step.File)
			if _, err := os.Stat(absFile); err != nil {
				t.Errorf("[%s] %s: file %q does not exist: %v",
					name, stepID, step.File, err)
				continue
			}
			if step.Line == 0 {
				continue
			}
			if step.Line < 1 {
				t.Errorf("[%s] %s: line %d < 1 (invalid anchor)",
					name, stepID, step.Line)
				continue
			}
			lc, err := lineCount(absFile)
			if err != nil {
				t.Errorf("[%s] %s: cannot count lines in %q: %v",
					name, stepID, step.File, err)
				continue
			}
			if step.Line > lc {
				t.Errorf("[%s] %s: line %d exceeds file length %d in %q",
					name, stepID, step.Line, lc, step.File)
			}
		}
	}
}
