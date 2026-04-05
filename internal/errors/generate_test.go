package errors

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGenerateCatalogIdempotent(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))

	gen := exec.Command("go", "generate", "./internal/errors/...")
	gen.Dir = repoRoot
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("go generate failed: %v\n%s", err, string(out))
	}

	diff := exec.Command("git", "diff", "--exit-code", "--", "internal/errors/catalog_gen.go")
	diff.Dir = repoRoot
	if out, err := diff.CombinedOutput(); err != nil {
		t.Fatalf("catalog_gen.go changed after go generate (not idempotent): %v\n%s", err, string(out))
	}
}
