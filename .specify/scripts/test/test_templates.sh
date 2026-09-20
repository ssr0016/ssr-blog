#!/usr/bin/env bash
# test_templates.sh -- checks for template directories and field coverage
set -u

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
TPL_A="$ROOT/.specify/templates"
TPL_B="$ROOT/.ai-workflow/templates"
M="$TPL_A/metrics.md"
F="$TPL_A/findings.md"
C="$TPL_A/checklist.md"
FEAT_M="$ROOT/.specify/specs/002-ai-workflow-enhancements/metrics.md"

pass=0
fail=0

check() {
    local name="$1"; shift
    if "$@" >/dev/null 2>&1; then
        printf 'PASS %s\n' "$name"
        pass=$((pass + 1))
    else
        printf 'FAIL %s\n' "$name"
        fail=$((fail + 1))
    fi
}

# Template dirs identical
check "dirs_identical" diff -rq "$TPL_A" "$TPL_B"

# metrics.md has all required fields (spec behaviors 29-31)
for label in "Tickets" "Context clears" "Converge rounds" "Tickets reopened after done" "Wall-clock minutes" "Tokens" "100K crossed" "Caps in effect" "Stop condition"; do
    check "metrics_field_$label" grep -qF "$label" "$M"
done

# metrics.md summary block
for key in feature status tickets tickets_closed context_clears converge_rounds converge_minutes converge_tokens findings_blocker findings_should_fix findings_nit tickets_reopened; do
    check "metrics_summary_$key" grep -qE "^${key}:" "$M"
done

# findings.md covers behavior 1, 7, 13
check "findings_behavior_1_range" grep -qF "Range reviewed" "$F"
check "findings_behavior_1_spec"  grep -qF "Spec reviewed" "$F"
check "findings_behavior_7_ledger" grep -qF "Ledger" "$F"
check "findings_behavior_7_columns" bash -c "grep -qE '\\| Round \\| Severity \\| Summary \\| Disposition \\| Reason \\| Repeat of \\|' '$F'"
check "findings_behavior_13_status" grep -qF "Stop condition" "$F"

# findings.md Reviewer brief options follow the fixed rotation (spec behavior 2), in both copies
for tpl in "$TPL_A" "$TPL_B"; do
    tag="$(basename "$(dirname "$tpl")")"
    tag="${tag#.}"
    brief=$(grep -m1 -F 'Reviewer brief:' "$tpl/findings.md")
    check "findings_brief_round1_${tag}" bash -c 'printf "%s\n" "$1" | grep -q -F "Round 1: spec conformance and rules/security"' _ "$brief"
    check "findings_brief_round2_${tag}" bash -c 'printf "%s\n" "$1" | grep -q -F "Round 2: fix verification, then general"' _ "$brief"
    check "findings_brief_round3_${tag}" bash -c 'printf "%s\n" "$1" | grep -q -F "Round 3: adversarial and test quality"' _ "$brief"
    check "findings_brief_round4_${tag}" bash -c 'printf "%s\n" "$1" | grep -q -F "Round 4: no-context full review"' _ "$brief"
    check "findings_brief_no_round1_general_${tag}" bash -c '! printf "%s\n" "$1" | grep -q -F "Round 1: general"' _ "$brief"
done

# checklist.md has evidence rule
check "checklist_evidence_rule" grep -qF "Evidence rule" "$C"

# feature 002 metrics has every field of the finished template
for key in feature status tickets tickets_closed context_clears converge_rounds converge_minutes converge_tokens findings_blocker findings_should_fix findings_nit tickets_reopened; do
    check "feat002_summary_$key" grep -qE "^${key}:" "$FEAT_M"
done

# ASCII only
check "metrics_ascii"  bash -c "! LC_ALL=C grep -qP '[^\\x00-\\x7F]' '$M'"
check "findings_ascii" bash -c "! LC_ALL=C grep -qP '[^\\x00-\\x7F]' '$F'"
check "checklist_ascii" bash -c "! LC_ALL=C grep -qP '[^\\x00-\\x7F]' '$C'"

printf '\n%s passed, %s failed\n' "$pass" "$fail"
if [ "$fail" -gt 0 ]; then
    exit 1
fi
exit 0
