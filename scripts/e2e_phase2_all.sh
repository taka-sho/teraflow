#!/bin/bash
# Full E2E runner for Phase2 Wave1-Wave4
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)

echo "======================================"
echo "  Phase2 Full E2E Test Suite"
echo "======================================"
echo ""

TOTAL_FAIL=0

run_suite() {
  local script="$1"
  echo "Running: $script"
  if bash "$SCRIPT_DIR/$script"; then
    echo "$script PASSED"
  else
    echo "$script FAILED"
    TOTAL_FAIL=$((TOTAL_FAIL + 1))
  fi
  echo ""
}

run_suite "e2e_phase2_wave1.sh"
run_suite "e2e_phase2_wave2.sh"
run_suite "e2e_phase2_wave3.sh"
run_suite "e2e_phase2_wave4.sh"

echo "======================================"
if [ "$TOTAL_FAIL" -eq 0 ]; then
  echo "Phase2 Full E2E: ALL SUITES PASSED"
  echo "======================================"
  exit 0
fi

echo "Phase2 Full E2E: $TOTAL_FAIL SUITE(S) FAILED"
echo "======================================"
exit 1
