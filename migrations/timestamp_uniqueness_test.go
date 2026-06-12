package migrations_test

import (
	"os"
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
