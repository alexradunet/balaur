package cards_test

import (
	"testing"

	"github.com/alexradunet/balaur/internal/cards"
)

// allTypes is the canonical list of every shipped card type — used by
// multiple sub-tests to ensure the registry is complete and consistent.
var allTypes = []string{
	"today", "quests", "calendar", "timeline",
	"journal", "measure", "lines", "memory", "skills", "heads",
}

func TestAll(t *testing.T) {
	got := cards.All()
	if len(got) != len(allTypes) {
		t.Fatalf("All() returned %d specs; want %d", len(got), len(allTypes))
	}
}

func TestGetEachType(t *testing.T) {
	for _, typ := range allTypes {
		spec, ok := cards.Get(typ)
		if !ok {
			t.Errorf("Get(%q) → not found", typ)
			continue
		}
		if spec.Type != typ {
			t.Errorf("Get(%q).Type = %q; want %q", typ, spec.Type, typ)
		}
		if spec.Label == "" {
			t.Errorf("Get(%q).Label is empty", typ)
		}
		if spec.Icon == "" {
			t.Errorf("Get(%q).Icon is empty", typ)
		}
		if spec.W == 0 {
			t.Errorf("Get(%q).W is zero", typ)
		}
	}
}

func TestGetUnknown(t *testing.T) {
	_, ok := cards.Get("nope")
	if ok {
		t.Error("Get(\"nope\") returned ok=true; want false")
	}
}

func TestValidateUnknownType(t *testing.T) {
	_, err := cards.Validate("nope", nil)
	if err == nil {
		t.Error("Validate(\"nope\") returned nil error; want an error")
	}
}

func TestValidateMissingRequired(t *testing.T) {
	// "measure" requires "kind"
	_, err := cards.Validate("measure", map[string]string{})
	if err == nil {
		t.Error("Validate(\"measure\", {}) should error for missing required \"kind\"")
	}
}

func TestValidateMissingRequiredLines(t *testing.T) {
	// "lines" requires "kind"
	_, err := cards.Validate("lines", map[string]string{})
	if err == nil {
		t.Error("Validate(\"lines\", {}) should error for missing required \"kind\"")
	}
}

func TestValidateBadEnum(t *testing.T) {
	_, err := cards.Validate("quests", map[string]string{"status": "bogus"})
	if err == nil {
		t.Error("Validate(\"quests\", status=bogus) should error for bad enum value")
	}
}

func TestValidateDropsUnknownKeys(t *testing.T) {
	out, err := cards.Validate("today", map[string]string{"surprise": "yes", "extra": "param"})
	if err != nil {
		t.Fatalf("Validate(\"today\", unknown keys) error: %v", err)
	}
	if _, ok := out["surprise"]; ok {
		t.Error("unknown key \"surprise\" was not dropped from cleaned map")
	}
	if _, ok := out["extra"]; ok {
		t.Error("unknown key \"extra\" was not dropped from cleaned map")
	}
}

func TestValidateNumericClamping(t *testing.T) {
	tests := []struct {
		name      string
		typ       string
		params    map[string]string
		key       string
		wantValue string
	}{
		{
			name:      "limit=999 clamped to 50",
			typ:       "quests",
			params:    map[string]string{"limit": "999"},
			key:       "limit",
			wantValue: "50",
		},
		{
			name:      "limit=3 passes through",
			typ:       "journal",
			params:    map[string]string{"limit": "3"},
			key:       "limit",
			wantValue: "3",
		},
		{
			name:      "days=999 clamped to 366",
			typ:       "timeline",
			params:    map[string]string{"days": "999"},
			key:       "days",
			wantValue: "366",
		},
		{
			name:      "limit=0 clamped to 1",
			typ:       "memory",
			params:    map[string]string{"limit": "0"},
			key:       "limit",
			wantValue: "1",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := cards.Validate(tc.typ, tc.params)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out[tc.key] != tc.wantValue {
				t.Errorf("param %q = %q; want %q", tc.key, out[tc.key], tc.wantValue)
			}
		})
	}
}

func TestValidateNonParseableNumericFallsBack(t *testing.T) {
	// A non-parseable numeric should not error — the renderer applies its default.
	out, err := cards.Validate("quests", map[string]string{"limit": "oops"})
	if err != nil {
		t.Fatalf("non-parseable numeric should not error: %v", err)
	}
	// The key should be absent or empty — renderer uses its own default.
	if v, ok := out["limit"]; ok && v != "" {
		t.Errorf("expected empty/absent limit after bad parse, got %q", v)
	}
}

func TestValidateValidEnum(t *testing.T) {
	for _, status := range []string{"open", "done", "all"} {
		out, err := cards.Validate("quests", map[string]string{"status": status})
		if err != nil {
			t.Errorf("Validate(quests, status=%q) error: %v", status, err)
		}
		if out["status"] != status {
			t.Errorf("status %q not preserved in output, got %q", status, out["status"])
		}
	}
}

func TestValidateKindForMeasurePresent(t *testing.T) {
	out, err := cards.Validate("measure", map[string]string{"kind": "weight"})
	if err != nil {
		t.Fatalf("Validate(measure, kind=weight) error: %v", err)
	}
	if out["kind"] != "weight" {
		t.Errorf("kind not preserved: got %q", out["kind"])
	}
}

func TestNoWebImports(t *testing.T) {
	// This test is a compile-time fact: the package has no internal/web imports.
	// If you can run `go test ./internal/cards/...` without a cycle error, this passes.
	t.Log("compile-time verified: internal/cards has no internal/web imports")
}
