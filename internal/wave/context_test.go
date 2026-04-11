package wave

import (
	"strings"
	"testing"
)

func TestTrimToTokenBudget(t *testing.T) {
	in := strings.Repeat("abcd", 20)
	out := trimToTokenBudget(in, 10)
	if estimateTokens(out) > 10 {
		t.Fatalf("token budget exceeded: %d", estimateTokens(out))
	}
	if !strings.HasSuffix(out, "...") {
		t.Fatalf("expected truncation suffix, got %q", out)
	}
}

func TestSummarizeNodeIDs(t *testing.T) {
	got := summarizeNodeIDs([]string{"b", "a", "a", ""})
	if got != "a, b" {
		t.Fatalf("unexpected summary: %q", got)
	}
}
