package id

import "testing"

func TestNewReturnsUniqueNonEmptyID(t *testing.T) {
	a := New()
	b := New()

	if a == "" {
		t.Fatal("expected non-empty id")
	}

	if len(a) != 32 {
		t.Fatalf("expected 32-char hex id, got %d", len(a))
	}

	if a == b {
		t.Fatal("expected unique ids")
	}
}
