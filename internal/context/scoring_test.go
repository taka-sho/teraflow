package context

import (
	"testing"
	"time"

	"github.com/taka-sho/teraflow/internal/index"
)

func TestScoreEntryFormula(t *testing.T) {
	t.Parallel()
	entry := index.Entry{
		NodeID:    "design:arch",
		Title:     "System Architecture",
		DependsOn: []string{"req:base"},
		Tags:      []string{"architecture", "system"},
		UpdatedAt: time.Now().Add(-24 * time.Hour),
	}

	score := ScoreEntry(entry, "please use req:base", "architecture decision")
	want := 0.4*1.0 + 0.4*0.5 + 0.2*1.0
	if diff := score - want; diff < -0.0001 || diff > 0.0001 {
		t.Fatalf("ScoreEntry() = %f, want %f", score, want)
	}
}

func TestScoreEntryRecencyOld(t *testing.T) {
	t.Parallel()
	entry := index.Entry{UpdatedAt: time.Now().Add(-40 * 24 * time.Hour)}
	score := ScoreEntry(entry, "", "")
	want := 0.4*0.2 + 0.4*0.0 + 0.2*0.3
	if diff := score - want; diff < -0.0001 || diff > 0.0001 {
		t.Fatalf("ScoreEntry() = %f, want %f", score, want)
	}
}
