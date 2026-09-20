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

printf '\n%s passed, %s failed\n' "$pass" "$fail"
if [ "$fail" -gt 0 ]; then
    exit 1
fi
exit 0
