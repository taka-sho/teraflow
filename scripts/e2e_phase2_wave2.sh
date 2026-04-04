#!/bin/bash
# Wave2 E2E validation: workflow YAML implementations
set -euo pipefail

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$REPO_ROOT"

echo "=== Wave2 E2E Test ==="

echo "[1/5] Build check..."
GOTOOLCHAIN=auto go build ./...
echo "  OK: build passed"

echo "[2/5] Unit tests..."
GOTOOLCHAIN=auto go test ./... 2>&1 | tail -8
echo "  OK: tests finished"

echo "[3/5] YAML validation (internal/actions/templates/*.yml)..."
FAIL=0
validate_yaml() {
  local file="$1"
  if python3 -c "import yaml" >/dev/null 2>&1; then
    python3 -c "import yaml; yaml.safe_load(open('$file'))" >/dev/null 2>&1
  else
    ruby -e "require 'yaml'; YAML.safe_load(File.read('$file'), aliases: true)" >/dev/null 2>&1
  fi
}
for f in internal/actions/templates/*.yml; do
  validate_yaml "$f" || {
    echo "  FAIL: invalid YAML: $f"
    FAIL=1
  }
done
if [ "$FAIL" -ne 0 ]; then
  echo "YAML validation failed"
  exit 1
fi
echo "  OK: all YAML files valid"

echo "[4/5] Wave2 workflow implementation check..."
WORKFLOWS=(
  "teraflow-phase-transition.yml:teraflow phase"
  "teraflow-phase-gate.yml:teraflow scan"
  "teraflow-permission-guard.yml:permission"
  "teraflow-artifact-finalize.yml:teraflow changelog add"
  "teraflow-changelog-update.yml:teraflow changelog add"
  "teraflow-rework-impact.yml:teraflow rework"
)
for item in "${WORKFLOWS[@]}"; do
  file="${item%%:*}"
  pattern="${item##*:}"
  count=$(grep -c "$pattern" "internal/actions/templates/$file" 2>/dev/null || true)
  if [ "$count" -eq 0 ]; then
    echo "  FAIL: $file missing implementation for '$pattern'"
    exit 1
  fi
  echo "  OK: $file has $count line(s) matching '$pattern'"
done

echo "[5/5] codd scan..."
GOTOOLCHAIN=auto go run . scan --config .github/teraflow.yml 2>&1

echo ""
echo "=== Wave2 E2E: ALL PASS ==="
