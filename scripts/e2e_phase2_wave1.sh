#!/bin/bash
# Wave1 E2E validation: setup actions/templates
set -euo pipefail

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

cd "$REPO_ROOT"

echo "=== Wave1 E2E Test ==="

# Setup project
mkdir -p "$TMP/.github"
echo 'version: "1"' > "$TMP/.github/teraflow.yml"

# Test setup actions
echo "[1/3] Testing setup actions..."
GOTOOLCHAIN=auto go run . setup actions --config "$TMP/.github/teraflow.yml"
count=$(ls "$TMP/.github/workflows/" | wc -l | tr -d ' ')
[ "$count" -eq 16 ] || { echo "FAIL: expected 16 workflows, got $count"; exit 1; }
echo "  OK: $count workflow files generated"

# Validate YAML syntax
echo "[2/3] Validating YAML syntax..."
validate_yaml() {
  local file="$1"
  if python3 -c "import yaml" >/dev/null 2>&1; then
    python3 -c "import yaml; yaml.safe_load(open('$file'))" >/dev/null 2>&1
  else
    ruby -e "require 'yaml'; YAML.safe_load(File.read('$file'), aliases: true)" >/dev/null 2>&1
  fi
}

for f in "$TMP/.github/workflows/"*.yml; do
  validate_yaml "$f" || { echo "FAIL: invalid YAML: $f"; exit 1; }
done
echo "  OK: all 16 YAMLs are valid"

# Test setup templates
echo "[3/3] Testing setup templates..."
GOTOOLCHAIN=auto go run . setup templates --config "$TMP/.github/teraflow.yml"
issue_count=$(ls "$TMP/.github/ISSUE_TEMPLATE/" | wc -l | tr -d ' ')
disc_count=$(ls "$TMP/.github/DISCUSSION_TEMPLATE/" | wc -l | tr -d ' ')
[ "$issue_count" -eq 6 ] || { echo "FAIL: expected 6 issue templates, got $issue_count"; exit 1; }
[ "$disc_count" -eq 3 ] || { echo "FAIL: expected 3 discussion templates, got $disc_count"; exit 1; }
echo "  OK: $issue_count issue templates, $disc_count discussion templates generated"

echo ""
echo "=== Wave1 E2E: ALL PASS ==="
