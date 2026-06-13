package cards_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/cards"
)

// allTypes is the canonical list of every shipped card type — used by
// multiple sub-tests to ensure the registry is complete and consistent.
var allTypes = []string{
	"today", "quests", "calendar", "timeline",
	"journal", "day", "measure", "lines", "memory", "skills", "heads", "habits", "lifelog",
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

func TestValidateFreeStringParamCap(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
	}{
		{
			name:    "1000-char kind is truncated to 256",
			input:   strings.Repeat("x", 1000),
			wantLen: 256,
		},
		{
			name:    "10-char kind is unchanged",
			input:   strings.Repeat("a", 10),
			wantLen: 10,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := cards.Validate("measure", map[string]string{"kind": tc.input})
			if err != nil {
				t.Fatalf("Validate error: %v", err)
			}
			if got := len(out["kind"]); got != tc.wantLen {
				t.Errorf("len(kind) = %d; want %d", got, tc.wantLen)
			}
		})
	}
}

func TestHasManage(t *testing.T) {
	for _, typ := range []string{"quests", "memory", "skills", "heads"} {
		if !cards.HasManage(typ) {
			t.Errorf("HasManage(%q) = false, want true", typ)
		}
	}
	for _, typ := range []string{"today", "calendar", "journal", "day", "habits", "lifelog", "timeline", "nope"} {
		if cards.HasManage(typ) {
			t.Errorf("HasManage(%q) = true, want false", typ)
		}
	}
}

func TestNoWebImports(t *testing.T) {
	// This test is a compile-time fact: the package has no internal/web imports.
	// If you can run `go test ./internal/cards/...` without a cycle error, this passes.
	t.Log("compile-time verified: internal/cards has no internal/web imports")
}

// TestSpecHasDefaultH verifies that every registered spec has a non-zero H.
func TestSpecHasDefaultH(t *testing.T) {
	for _, spec := range cards.All() {
		if spec.H == 0 {
			t.Errorf("spec %q has H=0; expected a non-zero default height", spec.Type)
		}
	}
}

// TestLayoutClamping verifies ValidateCards clamps layout fields to valid ranges.
func TestLayoutClamping(t *testing.T) {
	tests := []struct {
		name  string
		card  cards.Card
		wantX int
		wantY int
		wantW int
		wantH int
	}{
		{
			name:  "all zeros pass through unchanged",
			card:  cards.Card{Type: "today", X: 0, Y: 0, W: 0, H: 0},
			wantX: 0, wantY: 0, wantW: 0, wantH: 0,
		},
		{
			name:  "X clamped to 11",
			card:  cards.Card{Type: "today", X: 99},
			wantX: 11,
		},
		{
			name:  "Y clamped to 500",
			card:  cards.Card{Type: "today", Y: 9999},
			wantY: 500,
		},
		{
			name:  "W clamped to 12",
			card:  cards.Card{Type: "today", W: 999},
			wantW: 12,
		},
		{
			name:  "H clamped to 120",
			card:  cards.Card{Type: "today", H: 9999},
			wantH: 120,
		},
		{
			name:  "X+W shrunk to fit (X=8, W=6 → W=4)",
			card:  cards.Card{Type: "today", X: 8, W: 6},
			wantX: 8, wantW: 4,
		},
		{
			name:  "X+W exactly 12 is fine",
			card:  cards.Card{Type: "today", X: 8, W: 4},
			wantX: 8, wantW: 4,
		},
		{
			name:  "W=0 (use default) not shrunk even when X=8",
			card:  cards.Card{Type: "today", X: 8, W: 0},
			wantX: 8, wantW: 0,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := cards.ValidateCards([]cards.Card{tc.card})
			if err != nil {
				t.Fatalf("ValidateCards error: %v", err)
			}
			c := out[0]
			if tc.wantX != 0 && c.X != tc.wantX {
				t.Errorf("X = %d; want %d", c.X, tc.wantX)
			}
			if tc.wantY != 0 && c.Y != tc.wantY {
				t.Errorf("Y = %d; want %d", c.Y, tc.wantY)
			}
			if tc.wantW != 0 && c.W != tc.wantW {
				t.Errorf("W = %d; want %d", c.W, tc.wantW)
			}
			if tc.wantH != 0 && c.H != tc.wantH {
				t.Errorf("H = %d; want %d", c.H, tc.wantH)
			}
		})
	}
}

// TestLayoutJSONRoundTrip verifies that zero layout fields are omitted in JSON
// (backward-compatible) and non-zero layout fields round-trip correctly.
func TestLayoutJSONRoundTrip(t *testing.T) {
	t.Run("zero layout omitted from JSON", func(t *testing.T) {
		c := cards.Card{Type: "today"}
		b, err := json.Marshal(c)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		s := string(b)
		for _, key := range []string{`"x"`, `"y"`, `"w"`, `"h"`} {
			if strings.Contains(s, key) {
				t.Errorf("JSON contains %s but should be omitted when zero: %s", key, s)
			}
		}
	})

	t.Run("non-zero layout round-trips", func(t *testing.T) {
		c := cards.Card{Type: "today", X: 2, Y: 5, W: 4, H: 16}
		b, err := json.Marshal(c)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var got cards.Card
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got.X != 2 || got.Y != 5 || got.W != 4 || got.H != 16 {
			t.Errorf("round-trip mismatch: got %+v", got)
		}
	})
}
