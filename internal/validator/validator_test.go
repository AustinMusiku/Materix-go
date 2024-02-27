package validator

import "testing"

func TestNew(t *testing.T) {
	v := New()
	if v == nil {
		t.Error("New returned nil")
	}

	if v != nil && len(v.Errors) > 0 {
		t.Errorf("Expected empty errors map, but got %d", len(v.Errors))
	}
}

func TestValid(t *testing.T) {
	v := New()
	if !v.Valid() {
		t.Error("New validator should be valid")
	}
	v.Errors["test"] = "error"
	if v.Valid() {
		t.Error("Validator with errors should be invalid")
	}
}

func TestAddError(t *testing.T) {
	v := New()
	v.AddError("test", "error")
	if len(v.Errors) != 1 {
		t.Errorf("Expected 1 error, but got %d", len(v.Errors))
	}
}

func TestCheck(t *testing.T) {
	v := New()
	v.Check(true, "test", "error")
	if len(v.Errors) != 0 {
		t.Errorf("Expected 0 errors, but got %d", len(v.Errors))
	}

	v.Check(false, "test", "error")
	if len(v.Errors) != 1 {
		t.Errorf("Expected 1 error, but got %d", len(v.Errors))
	}
}

func TestIn(t *testing.T) {
	if !In("test", "test", "test1") {
		t.Error("Expected true, but got false")
	}

	if In("test", "test1", "test2") {
		t.Error("Expected false, but got true")
	}
}

func TestUnique(t *testing.T) {
	if !Unique([]string{"test", "test1"}) {
		t.Error("Expected true, but got false")
	}

	if Unique([]string{"test", "test"}) {
		t.Error("Expected false, but got true")
	}
}
