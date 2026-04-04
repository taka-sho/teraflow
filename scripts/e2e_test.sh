#!/bin/bash
# teraflow Phase1 E2E Test Script
# Usage: bash scripts/e2e_test.sh

set -e

REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
TERAFLOW="$REPO_DIR/teraflow"
E2E_DIR="/tmp/teraflow-e2e-$$"
PASS=0
FAIL=0
ERRORS=()

cleanup() {
  rm -rf "$E2E_DIR"
}
trap cleanup EXIT

log_pass() { echo "  ✓ $1"; PASS=$((PASS+1)); }
log_fail() { echo "  ✗ $1"; FAIL=$((FAIL+1)); ERRORS+=("$1"); }

check() {
  local desc="$1"; shift
  if "$@" > /dev/null 2>&1; then
    log_pass "$desc"
  else
    log_fail "$desc: $*"
  fi
}

check_output() {
  local desc="$1"; local expected="$2"; shift 2
  local out
  out=$("$@" 2>&1) || true
  if echo "$out" | grep -q "$expected"; then
    log_pass "$desc"
  else
    log_fail "$desc (expected '$expected', got: $out)"
  fi
}

# Build
echo "Building teraflow..."
cd "$REPO_DIR"
go build -o teraflow . || { echo "BUILD FAILED"; exit 1; }
echo "Build OK"

# Setup E2E dir
mkdir -p "$E2E_DIR"
CONFIG="$E2E_DIR/.github/teraflow.yml"

echo ""
echo "=== Scenario 1: Basic Lifecycle Flow ==="
cd "$E2E_DIR"
check "init" "$TERAFLOW" init --name "e2e-project" --non-interactive
check "status" "$TERAFLOW" --config "$CONFIG" status
check_output "status shows stage" "initial_development" "$TERAFLOW" --config "$CONFIG" status
check "phase list" "$TERAFLOW" --config "$CONFIG" phase list
check "stage list" "$TERAFLOW" --config "$CONFIG" stage list
check "stage status" "$TERAFLOW" --config "$CONFIG" stage status
check "phase start" "$TERAFLOW" --config "$CONFIG" phase start requirements
check_output "status shows requirements" "requirements" "$TERAFLOW" --config "$CONFIG" status
check "phase complete" "$TERAFLOW" --config "$CONFIG" phase complete
check_output "phase advanced to basic_design" "basic_design" "$TERAFLOW" --config "$CONFIG" status

echo ""
echo "=== Scenario 2: Rework Flow ==="
check "rework create" "$TERAFLOW" --config "$CONFIG" rework create \
  --group "group-auth" --target-phase "requirements" --reason "API仕様変更"
check "rework list" "$TERAFLOW" --config "$CONFIG" rework list
check_output "rework list shows entry" "rw-001" "$TERAFLOW" --config "$CONFIG" rework list
check "scan" "$TERAFLOW" --config "$CONFIG" scan

echo ""
echo "=== Scenario 3: Operations Flow ==="
check "incident create" "$TERAFLOW" --config "$CONFIG" incident create \
  --title "本番障害テスト" --severity "major"
check "incident list" "$TERAFLOW" --config "$CONFIG" incident list
check_output "incident shows entry" "inc-001" "$TERAFLOW" --config "$CONFIG" incident list
check "incident close" "$TERAFLOW" --config "$CONFIG" incident close --id "inc-001"
check_output "incident closed" "closed" "$TERAFLOW" --config "$CONFIG" incident list
check "changelog add" "$TERAFLOW" --config "$CONFIG" changelog add feat "初期リリース"
check "changelog generate" "$TERAFLOW" --config "$CONFIG" changelog generate

echo ""
echo "=== Scenario 4: Doctor Flow ==="
check "doctor" "$TERAFLOW" --config "$CONFIG" doctor
check_output "doctor json status" "status" "$TERAFLOW" --config "$CONFIG" --format json doctor

echo ""
echo "=== Scenario 5: Config & Schedule Flow ==="
check "config show" "$TERAFLOW" --config "$CONFIG" config show
check "config set" "$TERAFLOW" --config "$CONFIG" config set project.description "E2Eテストプロジェクト"
check_output "config set reflected" "E2Eテストプロジェクト" "$TERAFLOW" --config "$CONFIG" config show
check "schedule show" "$TERAFLOW" --config "$CONFIG" schedule show
check "schedule update" "$TERAFLOW" --config "$CONFIG" schedule update

echo ""
echo "=== Scenario 6: Dashboard & JSON Output ==="
check "dashboard show" "$TERAFLOW" --config "$CONFIG" dashboard show
check_output "status --format json" "stage" "$TERAFLOW" --config "$CONFIG" --format json status
check_output "phase list --format json" "phases" "$TERAFLOW" --config "$CONFIG" --format json phase list

echo ""
echo "================================"
echo "E2E Results: PASS=$PASS FAIL=$FAIL"
if [ ${#ERRORS[@]} -gt 0 ]; then
  echo "Failed checks:"
  for e in "${ERRORS[@]}"; do echo "  - $e"; done
fi
echo "================================"

[ $FAIL -eq 0 ] && exit 0 || exit 1
