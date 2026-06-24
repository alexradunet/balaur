package nodes

import (
	"testing"

	"github.com/alexradunet/balaur/internal/storetest"
)

// --- Pure-function tests (no DB) ---

func TestValidateProps_EmptySchema(t *testing.T) {
	// Empty schema accepts anything.
	if err := ValidateProps(nil, map[string]any{"foo": "bar", "baz": 42.0}); err != nil {
		t.Errorf("empty schema should accept any props, got: %v", err)
	}
	if err := ValidateProps([]PropDef{}, map[string]any{"x": true}); err != nil {
		t.Errorf("empty schema should accept any props, got: %v", err)
	}
}

func TestValidateProps_HappyPath(t *testing.T) {
	defs := []PropDef{
		{Key: "author", Type: PropText},
		{Key: "year", Type: PropNumber, Required: true},
	}
	props := map[string]any{"author": "Tolkien", "year": float64(2020)}
	if err := ValidateProps(defs, props); err != nil {
		t.Errorf("valid book props should pass: %v", err)
	}
}

func TestValidateProps_RequiredMissing(t *testing.T) {
	defs := []PropDef{
		{Key: "year", Type: PropNumber, Required: true},
	}
	err := ValidateProps(defs, map[string]any{})
	if err == nil {
		t.Error("expected error for missing required field")
	}
}

func TestValidateProps_WrongTypeNumber(t *testing.T) {
	defs := []PropDef{
		{Key: "year", Type: PropNumber, Required: true},
	}
	err := ValidateProps(defs, map[string]any{"year": "twenty"})
	if err == nil {
		t.Error("expected error for wrong type (string for number)")
	}
}

func TestValidateProps_SelectBadValue(t *testing.T) {
	defs := []PropDef{
		{Key: "category", Type: PropSelect, Required: true,
			Options: []string{"fact", "preference"}},
	}
	err := ValidateProps(defs, map[string]any{"category": "alien"})
	if err == nil {
		t.Error("expected error for select value not in options")
	}
}

func TestValidateProps_SelectGoodValue(t *testing.T) {
	defs := []PropDef{
		{Key: "category", Type: PropSelect, Required: true,
			Options: []string{"fact", "preference"}},
	}
	if err := ValidateProps(defs, map[string]any{"category": "fact"}); err != nil {
		t.Errorf("valid select value should pass: %v", err)
	}
}

func TestValidateProps_UnknownKeyAllowed(t *testing.T) {
	// Extra keys like use_count / last_used written by internal/knowledge must
	// not fail validation.
	defs := []PropDef{
		{Key: "description", Type: PropText},
	}
	props := map[string]any{
		"description": "some desc",
		"use_count":   float64(5),
		"last_used":   "extra",
	}
	if err := ValidateProps(defs, props); err != nil {
		t.Errorf("unknown extra keys must be allowed, got: %v", err)
	}
}

func TestValidateProps_DateValid(t *testing.T) {
	defs := []PropDef{{Key: "when", Type: PropDate}}
	if err := ValidateProps(defs, map[string]any{"when": "2024-01-01T00:00:00Z"}); err != nil {
		t.Errorf("valid RFC3339 date should pass: %v", err)
	}
}

func TestValidateProps_DateInvalid(t *testing.T) {
	defs := []PropDef{{Key: "when", Type: PropDate}}
	err := ValidateProps(defs, map[string]any{"when": "not-a-date"})
	if err == nil {
		t.Error("expected error for invalid date string")
	}
}

func TestValidateProps_BoolType(t *testing.T) {
	defs := []PropDef{{Key: "active", Type: PropBool}}
	if err := ValidateProps(defs, map[string]any{"active": true}); err != nil {
		t.Errorf("valid bool should pass: %v", err)
	}
	if err := ValidateProps(defs, map[string]any{"active": "yes"}); err == nil {
		t.Error("expected error for string in bool field")
	}
}

// --- ApplyTemplate pure-function tests ---

func TestApplyTemplate_FillsMissingKey(t *testing.T) {
	tmpl := map[string]any{"author": "Unknown"}
	props := map[string]any{}
	_, out := ApplyTemplate(tmpl, "", props)
	if out["author"] != "Unknown" {
		t.Errorf("template should fill missing key, got %v", out["author"])
	}
}

func TestApplyTemplate_LeavesExistingKey(t *testing.T) {
	tmpl := map[string]any{"author": "Unknown"}
	props := map[string]any{"author": "Tolkien"}
	_, out := ApplyTemplate(tmpl, "", props)
	if out["author"] != "Tolkien" {
		t.Errorf("template should not overwrite existing key, got %v", out["author"])
	}
}

func TestApplyTemplate_FillsEmptyBody(t *testing.T) {
	tmpl := map[string]any{"_body": "# Template body"}
	body, _ := ApplyTemplate(tmpl, "", map[string]any{})
	if body != "# Template body" {
		t.Errorf("template should fill empty body, got %q", body)
	}
}

func TestApplyTemplate_DoesNotOverwriteBody(t *testing.T) {
	tmpl := map[string]any{"_body": "template"}
	body, _ := ApplyTemplate(tmpl, "real content", map[string]any{})
	if body != "real content" {
		t.Errorf("template should not overwrite non-empty body, got %q", body)
	}
}

func TestApplyTemplate_NilTemplate(t *testing.T) {
	props := map[string]any{"x": "y"}
	body, out := ApplyTemplate(nil, "body", props)
	if body != "body" || out["x"] != "y" {
		t.Error("nil template should return inputs unchanged")
	}
}

// --- DB-backed tests ---

func TestCreateValidation_RequiredPropMissing(t *testing.T) {
	// memory requires "importance" (number); category is optional (code may write "").
	// Creating without importance should fail.
	app := storetest.NewApp(t)
	_, err := Create(app, "memory", "Test memory", "", "proposed", map[string]any{
		"category":    "fact",
		"when_to_use": "when helpful",
		// intentionally omit importance
	})
	if err == nil {
		t.Error("expected validation error for memory missing required importance")
	}
}

func TestCreateValidation_ValidMemoryProps(t *testing.T) {
	app := storetest.NewApp(t)
	_, err := Create(app, "memory", "Test memory", "some content", "proposed", map[string]any{
		"category":    "fact",
		"importance":  float64(3),
		"when_to_use": "when helpful",
		"source":      "test",
	})
	if err != nil {
		t.Errorf("valid memory props should succeed: %v", err)
	}
}

func TestCreateValidation_EmptySchemaAcceptsAnyProps(t *testing.T) {
	// note has an empty schema — any props must be accepted.
	app := storetest.NewApp(t)
	_, err := Create(app, "note", "Test note", "body", "active", map[string]any{
		"random_key": "random_value",
		"another":    float64(42),
	})
	if err != nil {
		t.Errorf("empty-schema type should accept any props: %v", err)
	}
}

func TestCreateValidation_WikilinkStubNote(t *testing.T) {
	// resolveOrCreateStub creates type=note stubs directly via app.Save (bypassing
	// nodes.Create), so validation does not block stub creation. This test confirms
	// that a note node CAN be created via nodes.Create with no props as well.
	app := storetest.NewApp(t)
	_, err := Create(app, "note", "Stub title", "", "active", nil)
	if err != nil {
		t.Errorf("stub note creation via Create should succeed: %v", err)
	}
}
