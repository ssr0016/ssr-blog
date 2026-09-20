#!/usr/bin/env bash
# test_verify_edit.sh -- self-contained test for verify-edit.sh
set -u

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
VE="$ROOT/.specify/scripts/verify-edit.sh"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

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

refuse() {
    local name="$1"; shift
    if "$@" >/dev/null 2>&1; then
        printf 'FAIL %s (expected non-zero, got success)\n' "$name"
        fail=$((fail + 1))
    else
        printf 'PASS %s\n' "$name"
        pass=$((pass + 1))
    fi
}

expect_exit() {
    local name="$1"; local want="$2"; shift 2
    "$@" >/dev/null 2>&1
    local got=$?
    if [ "$got" -eq "$want" ]; then
        printf 'PASS %s\n' "$name"
        pass=$((pass + 1))
    else
        printf 'FAIL %s (want exit %s, got %s)\n' "$name" "$want" "$got"
        fail=$((fail + 1))
    fi
}

target="$tmp/file.txt"
printf "alpha\\nbeta\\ngamma\\n" > "$target"

# Case 1: --contains passes when present, exits 1 when absent
expect_exit "case1_contains_present" 0 "$VE" "$target" --contains beta
expect_exit "case1_contains_absent" 1 "$VE" "$target" --contains zzz

# Case 2: --absent passes when gone, exits 1 when still there
expect_exit "case2_absent_gone" 0 "$VE" "$target" --absent zzz
expect_exit "case2_absent_present" 1 "$VE" "$target" --absent beta

# Case 3: --count passes on exact count, exits 1 on different, including more
printf "aa\\nbb\\naa\\ncc\\naa\\n" > "$target"
expect_exit "case3_count_exact" 0 "$VE" "$target" --count "aa=3"
expect_exit "case3_count_low" 1 "$VE" "$target" --count "aa=2"
expect_exit "case3_count_high" 1 "$VE" "$target" --count "aa=4"
expect_exit "case3_count_zero" 0 "$VE" "$target" --count "zzz=0"

# Case 4: sed -i that matches nothing -> file unchanged, verify-edit exits 1
printf "keep this\\n" > "$target"
before=$(sha256sum "$target" | awk "{print \$1}")
sed -i "s/NEVER_MATCHES/xxx/" "$target"
after=$(sha256sum "$target" | awk "{print \$1}")
check "case4_sed_no_change" test "$before" = "$after"
expect_exit "case4_edit_not_landed" 1 "$VE" "$target" --contains xxx

# Case 5: missing/unreadable file exits 2; no args or unknown option exits 64
expect_exit "case5_missing_file" 2 "$VE" "$tmp/nope.txt" --contains x
printf "hi\\n" > "$target"
chmod 000 "$target"
if [ -r "$target" ]; then
    printf 'SKIP case5_unreadable (running as root)\n'
else
    expect_exit "case5_unreadable" 2 "$VE" "$target" --contains hi
fi
chmod 644 "$target"
expect_exit "case5_no_args" 64 "$VE"
expect_exit "case5_file_only" 64 "$VE" "$target"
expect_exit "case5_unknown_opt" 64 "$VE" "$target" --bogus x
expect_exit "case5_bad_count" 64 "$VE" "$target" --count "noequals"

# Case 6: special characters (regex metachars, slashes, quotes) matched literally
printf "a.b*c\npath/to/x\nquote'x\n" > "$target"
expect_exit "case6_dot_star" 0 "$VE" "$target" --contains "a.b*c"
expect_exit "case6_dot_literal_only" 1 "$VE" "$target" --contains "aXbXc"
expect_exit "case6_slash" 0 "$VE" "$target" --contains "path/to/x"
expect_exit "case6_quote" 0 "$VE" "$target" --contains "quote'x"

# Case 7: file never modified by verify-edit (checksum before/after)
printf "stable\\ncontent\\n" > "$target"
before=$(sha256sum "$target" | awk "{print \$1}")
"$VE" "$target" --contains stable >/dev/null 2>&1 || true
"$VE" "$target" --contains zzz >/dev/null 2>&1 || true
"$VE" "$target" --absent stable >/dev/null 2>&1 || true
after=$(sha256sum "$target" | awk "{print \$1}")
check "case7_readonly" test "$before" = "$after"

printf '\n%s passed, %s failed\n' "$pass" "$fail"
if [ "$fail" -gt 0 ]; then
    exit 1
fi
exit 0
