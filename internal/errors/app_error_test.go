package errors

import (
	stdErrors "errors"
	"testing"
)

func withTestCatalog(t *testing.T, entries map[string]CatalogEntry) {
	t.Helper()
	prev := catalog
	catalog = entries
	t.Cleanup(func() {
		catalog = prev
	})
}

func TestNewFromCatalog(t *testing.T) {
	withTestCatalog(t, map[string]CatalogEntry{
		"TF-RB01": {
			Code:     "TF-RB01",
			Category: string(CatRBAC),
			ExitCode: 11,
			Template: "user %s has no permission",
		},
	})

	err := New("TF-RB01", "alice")
	if err.Code != "TF-RB01" {
		t.Fatalf("Code=%q, want TF-RB01", err.Code)
	}
	if err.Category != CatRBAC {
		t.Fatalf("Category=%q, want %q", err.Category, CatRBAC)
	}
	if err.ExitCode != 11 {
		t.Fatalf("ExitCode=%d, want 11", err.ExitCode)
	}
	if err.Message != "user alice has no permission" {
		t.Fatalf("Message=%q", err.Message)
	}
}

func TestNewFallbackForUnknownCode(t *testing.T) {
	withTestCatalog(t, map[string]CatalogEntry{})

	err := New("TF-XX99")
	if err.Code != "TF-XX99" {
		t.Fatalf("Code=%q, want TF-XX99", err.Code)
	}
	if err.Category != CatCLI {
		t.Fatalf("Category=%q, want %q", err.Category, CatCLI)
	}
	if err.ExitCode != 1 {
		t.Fatalf("ExitCode=%d, want 1", err.ExitCode)
	}
}

func TestWrapAndIs(t *testing.T) {
	withTestCatalog(t, map[string]CatalogEntry{
		"TF-GA01": {
			Code:     "TF-GA01",
			Category: string(CatGate),
			ExitCode: 12,
			Template: "gate blocked",
		},
	})

	base := stdErrors.New("file missing")
	err := Wrap("TF-GA01", base)

	if !Is(err, "TF-GA01") {
		t.Fatal("Is should match TF-GA01")
	}
	if Is(err, "TF-RB01") {
		t.Fatal("Is should not match TF-RB01")
	}
	if err.Unwrap() != base {
		t.Fatal("Unwrap should return original cause")
	}
}

func TestGetExitCode(t *testing.T) {
	withTestCatalog(t, map[string]CatalogEntry{
		"TF-AI01": {
			Code:     "TF-AI01",
			Category: string(CatAI),
			ExitCode: 9,
			Template: "provider timeout",
		},
	})

	if got := GetExitCode(nil); got != 0 {
		t.Fatalf("nil exit=%d, want 0", got)
	}
	if got := GetExitCode(stdErrors.New("plain")); got != 1 {
		t.Fatalf("plain exit=%d, want 1", got)
	}
	if got := GetExitCode(New("TF-AI01")); got != 9 {
		t.Fatalf("app exit=%d, want 9", got)
	}
}

func TestToJSONAndError(t *testing.T) {
	withTestCatalog(t, map[string]CatalogEntry{
		"TF-IO01": {
			Code:     "TF-IO01",
			Category: string(CatIO),
			ExitCode: 7,
			Template: "read failed for %s",
		},
	})

	cause := stdErrors.New("permission denied")
	err := Wrap("TF-IO01", cause, "config.yml")

	got := err.ToJSON()
	if got.ErrorCode != "TF-IO01" || got.Category != string(CatIO) || got.ExitCode != 7 {
		t.Fatalf("unexpected json: %+v", got)
	}
	if got.Details != "permission denied" {
		t.Fatalf("Details=%q", got.Details)
	}
	if err.Error() != "[TF-IO01] read failed for config.yml: permission denied" {
		t.Fatalf("Error=%q", err.Error())
	}
}
