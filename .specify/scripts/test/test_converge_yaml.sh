#!/usr/bin/env bash
# test_converge_yaml.sh -- self-contained test for .specify/converge.yaml
set -u

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
YAML="$ROOT/.specify/converge.yaml"
SPEC="$ROOT/.specify/specs/002-ai-workflow-enhancements/spec.md"

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

# Test 1: exactly three keys, no nesting, positive integers
keys=$(grep -E '^[a-z_]+:' "$YAML" | wc -l)
check "case1_exactly_three_keys" test "$keys" -eq 3

check "case1_max_rounds_is_positive_int" grep -qE '^max_rounds:[[:space:]]*[1-9][0-9]*[[:space:]]*(#.*)?$' "$YAML"
check "case1_max_minutes_is_positive_int" grep -qE '^max_minutes:[[:space:]]*[1-9][0-9]*[[:space:]]*(#.*)?$' "$YAML"
check "case1_max_tokens_is_positive_int" grep -qE '^max_tokens:[[:space:]]*[1-9][0-9]*[[:space:]]*(#.*)?$' "$YAML"

check "case1_no_nesting" bash -c "! grep -qE '^[[:space:]]+[a-z_]+:' \"$YAML\""

# Test 2: defaults match spec behavior 10 (read from spec.md)
spec_line=$(grep -E '^10\. Converge has hard caps' "$SPEC")
spec_rounds=$(printf '%s' "$spec_line" | grep -oE 'at most [0-9]+ rounds' | grep -oE '[0-9]+')
spec_minutes=$(printf '%s' "$spec_line" | grep -oE 'at most [0-9]+ minutes' | grep -oE '[0-9]+')
spec_tokens=$(printf '%s' "$spec_line" | grep -oE '[0-9]+K tokens' | grep -oE '[0-9]+')

yaml_rounds=$(grep -E '^max_rounds:' "$YAML" | grep -oE '[0-9]+')
yaml_minutes=$(grep -E '^max_minutes:' "$YAML" | grep -oE '[0-9]+')
yaml_tokens_raw=$(grep -E '^max_tokens:' "$YAML" 2>/dev/null | grep -oE '[0-9]+' || true)
yaml_tokens=0
if [ -n "$yaml_tokens_raw" ]; then
    yaml_tokens=$((yaml_tokens_raw / 1000))
fi

check "case2_max_rounds_matches_spec" test "$yaml_rounds" = "$spec_rounds"
check "case2_max_minutes_matches_spec" test "$yaml_minutes" = "$spec_minutes"
check "case2_max_tokens_matches_spec" test "$yaml_tokens" = "$spec_tokens"

# Test 3: no non-ASCII bytes
check "case3_ascii_only" bash -c "! LC_ALL=C grep -qP '[^\\x00-\\x7F]' \"$YAML\""

# Test 4: the converge command states cap precedence as repo, then home, then
# defaults (spec behavior 12, ADR 0006). "plus" is not an order.
CMD="$ROOT/.claude/commands/speckit-converge.md"
caps_line=$(grep -m1 '^Caps come from' "$CMD")
caps_next=$(grep -m1 -A1 '^Caps come from' "$CMD" | tail -n 1)
pat_order='.specify/converge.yaml`, then `~/.spec-kit/converge.yaml`'

check "case4_caps_repo_then_home" bash -c 'printf "%s\n" "$1" | grep -q -F -- "$2"' _ "$caps_line" "$pat_order"
check "case4_caps_no_plus" bash -c '! printf "%s\n" "$1" | grep -q -w plus' _ "$caps_line"
check "case4_caps_defaults_last" bash -c 'printf "%s\n" "$1" | grep -q "^then the built-in defaults"' _ "$caps_next"

# Test 5: the command says how the caps are enforced (plan.md, tasks.md T6): validate the
# values, record the start time and caps, and check before each round and after each reviewer.
has() { grep -q -F -- "$2" "$1"; }
check "case5_caps_validated_as_positive_integers" has "$CMD" 'Check that every value is a positive integer.'
check "case5_caps_stop_on_invalid_value" has "$CMD" 'stop with an error that names the key'
check "case5_start_time_and_caps_recorded" has "$CMD" 'Record the start time and the caps in effect in `metrics.md`'
check "case5_caps_checked_each_round_and_reviewer" has "$CMD" 'Check the caps before each round and again each time a reviewer returns.'
check "case5_parallel_overshoot_reported" has "$CMD" 'overshoot'

# Test 6: the spec edge cases for clean rounds, repeats, read-only reviewers and fixes
check "case6_unresolved_blocker_blocks_clean" has "$CMD" 'still not clean while the ledger holds an unresolved BLOCKER'
check "case6_fix_that_adds_should_fix_is_new" has "$CMD" 'introduces a new SHOULD-FIX is logged as a new finding'
check "case6_rejected_repeat_with_new_evidence" has "$CMD" 'If it comes with new evidence, log it as a fresh finding'
check "case6_repeat_of_column_named" has "$CMD" 'in the `Repeat of` column'
check "case6_reviewer_source_change_not_accepted" has "$CMD" 'the change is not accepted as part of the round.'
check "case6_reviewer_source_change_is_a_finding" has "$CMD" "Report it as a finding against the reviewer's brief."

# Test 7: the reviewer prompt carries the same rules
REVIEW="$ROOT/.ai-workflow/prompts/06-review.md"
check "case7_review_unresolved_blocker" has "$REVIEW" 'unresolved BLOCKER'
check "case7_review_source_change_is_a_finding" has "$REVIEW" 'finding against your brief'
check "case7_review_repeat_needs_new_evidence" has "$REVIEW" 'unless you have new evidence'

printf '\n%s passed, %s failed\n' "$pass" "$fail"
if [ "$fail" -gt 0 ]; then
    exit 1
fi
exit 0
