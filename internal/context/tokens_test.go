package context

import "testing"

func TestEstimateTokensJapanese(t *testing.T) {
	t.Parallel()
	text := "これは日本語の文章です"
	got := EstimateTokens(text)
	if got != 5 {
		t.Fatalf("EstimateTokens() = %d, want 5", got)
	}
}

func TestEstimateTokensEnglish(t *testing.T) {
	t.Parallel()
	text := "This is an english sentence"
	got := EstimateTokens(text)
	if got != 6 {
		t.Fatalf("EstimateTokens() = %d, want 6", got)
	}
}

func TestEstimateTokensMixed(t *testing.T) {
	t.Parallel()
	text := "日本語 and English"
	got := EstimateTokens(text)
	if got != 4 {
		t.Fatalf("EstimateTokens() = %d, want 4", got)
	}
}
