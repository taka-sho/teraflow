#!/bin/bash
# scripts/push-with-retry.sh
# Commit and push with retry for GitHub Actions concurrent execution (W-005)
set -e

MAX_RETRIES=${1:-3}
COMMIT_MSG=${2:-"chore: update project state [skip ci]"}
FILES=${3:-.github/project-state.yml}

git config user.name "github-actions[bot]"
git config user.email "github-actions[bot]@users.noreply.github.com"

# Stage files (ignore if nothing to stage)
git add $FILES || true
git diff --cached --quiet && echo "No changes to commit" && exit 0

git commit -m "$COMMIT_MSG"

for i in $(seq 1 $MAX_RETRIES); do
  git pull --rebase origin main || true
  if git push origin main; then
    echo "Push succeeded on attempt $i"
    exit 0
  fi
  echo "Push attempt $i/$MAX_RETRIES failed, retrying in 5s..."
  sleep 5
done

echo "ERROR: Push failed after $MAX_RETRIES attempts"
exit 1
