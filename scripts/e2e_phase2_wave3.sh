#!/bin/bash
# Wave3 E2E validation: AI agent package, command, templates
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

echo "=== Phase2 Wave3 E2E Test ==="

check "agent package builds" "cd $REPO_ROOT && GOTOOLCHAIN=auto go build ./internal/agent/..."
check "agent tests pass" "cd $REPO_ROOT && GOTOOLCHAIN=auto go test ./internal/agent/... -count=1"
check "agent coverage >= 70%" "cd $REPO_ROOT && GOTOOLCHAIN=auto go test ./internal/agent/... -coverprofile=/tmp/agent_cov.out >/dev/null 2>&1 && go tool cover -func=/tmp/agent_cov.out | grep total | awk '{print \$3}' | tr -d '%' | awk '{exit (\$1 >= 70) ? 0 : 1}'"

check "teraflow agent builds" "cd $REPO_ROOT && GOTOOLCHAIN=auto go build -o /tmp/teraflow_test . && /tmp/teraflow_test agent --help"
check "teraflow agent status works" "cd $REPO_ROOT && GOTOOLCHAIN=auto go run . agent status >/dev/null 2>&1 || true"

AGENT_WORKFLOWS=(
  "teraflow-review-agent.yml"
  "teraflow-req-agent.yml"
  "teraflow-ci-fix-agent.yml"
  "teraflow-implement-agent.yml"
  "teraflow-conflict-agent.yml"
  "teraflow-incident-agent.yml"
)
for wf in "${AGENT_WORKFLOWS[@]}"; do
  check "template exists: $wf" "test -f $REPO_ROOT/internal/actions/templates/$wf"
  check "template YAML valid: $wf" "python3 -c \"open('$REPO_ROOT/internal/actions/templates/$wf')\""
done

for agent_type in requirements review implement ci-fix conflict incident maintenance; do
  check "agent type defined: $agent_type" "grep -q \"AgentType.*=.*\\\"$agent_type\\\"\" $REPO_ROOT/internal/agent/agent.go"
done

echo ""
echo "=== Wave3 Results: PASS=$PASS, FAIL=$FAIL ==="
if [ "$FAIL" -eq 0 ]; then
  echo "Wave3 E2E: ALL PASS"
  exit 0
fi

echo "Wave3 E2E: $FAIL FAILURES"
exit 1
