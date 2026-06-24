package migrations_test

import (
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestMigrationTimestampsAreUnique(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		parts := strings.SplitN(e.Name(), "_", 2)
		if len(parts) < 2 {
			continue
		}
		prefix := parts[0]
		if prev, ok := seen[prefix]; ok {
			t.Errorf("duplicate timestamp prefix %s: %s and %s", prefix, prev, e.Name())
		}
		seen[prefix] = e.Name()
	}
}

func TestMigrationTimestampsAreStrictlyIncreasing(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		if len(strings.SplitN(e.Name(), "_", 2)) < 2 {
			continue
		}
		files = append(files, e.Name())
	}
	slices.Sort(files)

	prev := -1
	prevName := ""
	for _, name := range files {
		prefix := strings.SplitN(name, "_", 2)[0]
		if len(prefix) != 10 {
			t.Errorf("migration prefix %q in %s is not 10 digits", prefix, name)
			continue
		}
		cur, err := strconv.Atoi(prefix)
		if err != nil {
			t.Errorf("migration prefix %q in %s is not an integer: %v", prefix, name, err)
			continue
		}
		if cur <= prev {
			t.Errorf("migration prefixes not strictly increasing: %s (%d) <= previous %s (%d)", name, cur, prevName, prev)
		}
		prev = cur
		prevName = name
	}
}
