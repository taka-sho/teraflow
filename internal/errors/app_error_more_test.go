package errors

import (
	"testing"
)

func TestErrorNilReceiver(t *testing.T) {
	var e *AppError
	if got := e.Error(); got != "" {
		t.Fatalf("nil Error() = %q, want empty", got)
	}
}

func TestUnwrapNilReceiver(t *testing.T) {
	var e *AppError
	if got := e.Unwrap(); got != nil {
		t.Fatalf("nil Unwrap() = %v, want nil", got)
	}
}

func TestToJSONNilReceiver(t *testing.T) {
	var e *AppError
	got := e.ToJSON()
	if got.ErrorCode != "" || got.Category != "" || got.ExitCode != 0 {
		t.Fatalf("nil ToJSON() = %+v, want zero value", got)
	}
}

func TestToJSONWithoutCause(t *testing.T) {
	withTestCatalog(t, map[string]CatalogEntry{
		"TF-IO01": {Code: "TF-IO01", Category: string(CatIO), ExitCode: 7, Template: "read failed"},
	})
	err := New("TF-IO01")
	got := err.ToJSON()
	if got.Details != "" {
		t.Fatalf("Details = %q, want empty", got.Details)
	}
}

func TestLookupCatalogEntry(t *testing.T) {
	entry, ok := LookupCatalogEntry("TF-AI01")
	if !ok {
		t.Fatal("should find TF-AI01")
	}
	if entry.Code != "TF-AI01" {
		t.Fatalf("Code = %q", entry.Code)
	}
	if entry.Category != "ai" {
		t.Fatalf("Category = %q", entry.Category)
	}
}

func TestLookupCatalogEntryNormalization(t *testing.T) {
	entry, ok := LookupCatalogEntry("  tf-ai01  ")
	if !ok {
		t.Fatal("should find with normalization")
	}
	if entry.Code != "TF-AI01" {
		t.Fatalf("Code = %q", entry.Code)
	}
}

func TestLookupCatalogEntryNotFound(t *testing.T) {
	_, ok := LookupCatalogEntry("TF-XX99")
	if ok {
		t.Fatal("should not find TF-XX99")
	}
}

func TestListCatalogEntries(t *testing.T) {
	entries := ListCatalogEntries()
	if len(entries) == 0 {
		t.Fatal("should return catalog entries")
	}
	// Verify sorted
	for i := 1; i < len(entries); i++ {
		if entries[i].Code < entries[i-1].Code {
			t.Fatalf("not sorted: %s < %s", entries[i].Code, entries[i-1].Code)
		}
	}
}

func TestIsWithNonAppError(t *testing.T) {
	if Is(nil, "TF-XX99") {
		t.Fatal("Is(nil) should be false")
	}
}

func TestErrorWithCause(t *testing.T) {
	withTestCatalog(t, map[string]CatalogEntry{
		"TF-IO01": {Code: "TF-IO01", Category: string(CatIO), ExitCode: 7, Template: "read failed"},
	})
	err := New("TF-IO01")
	msg := err.Error()
	if msg != "[TF-IO01] read failed" {
		t.Fatalf("Error() = %q", msg)
	}
}

func TestGetExitCodeZeroExitCode(t *testing.T) {
	err := &AppError{Code: "TEST", ExitCode: 0}
	if got := GetExitCode(err); got != 1 {
		t.Fatalf("GetExitCode(zero) = %d, want 1 (fallback)", got)
	}
}
