#!/bin/bash
# Wave4 E2E validation: scheduled/maintenance workflow templates
set -euo pipefail

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
PASS=0
FAIL=0

check() {
  local desc="$1"
  local cmd="$2"
  if eval "$cmd" >/dev/null 2>&1; then
    echo "PASS: $desc"
    PASS=$((PASS + 1))
  else
    echo "FAIL: $desc"
    FAIL=$((FAIL + 1))
  fi
}

echo "=== Phase2 Wave4 E2E Test ==="

WAVE4_WORKFLOWS=(
  "teraflow-dashboard-deploy.yml"
  "teraflow-stuck-monitor.yml"
  "teraflow-maintenance-agent.yml"
  "teraflow-schedule-predict.yml"
)
for wf in "${WAVE4_WORKFLOWS[@]}"; do
  check "template exists: $wf" "test -f $REPO_ROOT/internal/actions/templates/$wf"
done

for wf in "${WAVE4_WORKFLOWS[@]}"; do
  check "no TODO in $wf" "! grep -q 'TODO(Wave4' $REPO_ROOT/internal/actions/templates/$wf"
done

check "dashboard-deploy has configure-pages" "grep -q 'configure-pages' $REPO_ROOT/internal/actions/templates/teraflow-dashboard-deploy.yml"
check "dashboard-deploy has deploy-pages" "grep -q 'deploy-pages' $REPO_ROOT/internal/actions/templates/teraflow-dashboard-deploy.yml"
check "stuck-monitor has git log check" "grep -q 'git log --since' $REPO_ROOT/internal/actions/templates/teraflow-stuck-monitor.yml"
check "maintenance-agent has doctor check" "grep -q 'teraflow doctor' $REPO_ROOT/internal/actions/templates/teraflow-maintenance-agent.yml"
check "schedule-predict has schedule update" "grep -q 'teraflow schedule update' $REPO_ROOT/internal/actions/templates/teraflow-schedule-predict.yml"
check "github-projects-setup.md exists" "test -f $REPO_ROOT/docs/github-projects-setup.md"

echo ""
echo "=== Wave4 Results: PASS=$PASS, FAIL=$FAIL ==="
if [ "$FAIL" -eq 0 ]; then
  echo "Wave4 E2E: ALL PASS"
  exit 0
fi

echo "Wave4 E2E: $FAIL FAILURES"
exit 1
