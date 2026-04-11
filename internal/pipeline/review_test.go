package pipeline

import "testing"

func TestDetermineReviewLevelSecurityEscalatesToApprove(t *testing.T) {
	got := DetermineReviewLevel(ModuleSpec{Path: "internal/auth/token.go"}, "auto", 1)
	if got != "approve" {
		t.Fatalf("DetermineReviewLevel() = %s, want approve", got)
	}
}

func TestDetermineReviewLevelExternalAPIEscalatesToReview(t *testing.T) {
	got := DetermineReviewLevel(ModuleSpec{Path: "internal/client/api.go"}, "auto", 1)
	if got != "review" {
		t.Fatalf("DetermineReviewLevel() = %s, want review", got)
	}
}

func TestDetermineReviewLevelWideChangeEscalatesToReview(t *testing.T) {
	got := DetermineReviewLevel(ModuleSpec{Path: "internal/core/feature.go"}, "auto", 5)
	if got != "review" {
		t.Fatalf("DetermineReviewLevel() = %s, want review", got)
	}
}

func TestDetermineReviewLevelDBEscalatesToApprove(t *testing.T) {
	got := DetermineReviewLevel(ModuleSpec{Path: "internal/db/migrations/001_init.sql"}, "review", 2)
	if got != "approve" {
		t.Fatalf("DetermineReviewLevel() = %s, want approve", got)
	}
}
