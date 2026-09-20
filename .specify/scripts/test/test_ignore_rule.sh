#!/usr/bin/env bash
# test_ignore_rule.sh -- verifies .specify/specs/.gitignore ignores *.bak.*
# but does NOT ignore spec.md or metrics.md.
set -eu

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
cd "$ROOT"

fail=0

if git check-ignore -q ".specify/specs/002-ai-workflow-enhancements/checklist.md.bak.2026-09-20T15-00-00"; then
    echo "PASS: backup name is ignored"
else
    echo "FAIL: backup name is NOT ignored"
    fail=1
fi

for f in ".specify/specs/002-ai-workflow-enhancements/spec.md" \
         ".specify/specs/002-ai-workflow-enhancements/metrics.md"; do
    if git check-ignore -q "$f"; then
        echo "FAIL: $f should NOT be ignored"
        fail=1
    else
        echo "PASS: $f not ignored"
    fi
done

exit "$fail"
